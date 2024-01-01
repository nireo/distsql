package service

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime"
	"strings"

	"github.com/nireo/distsql/consensus"
	"github.com/nireo/distsql/pb"
	"github.com/nireo/distsql/pb/encoding"
	"go.uber.org/zap"
)

var (
	ErrBadJson = errors.New("bad json couldn't be parsed properly")
	ErrEmpty   = errors.New("no statements were provided")
)

type Manager interface {
	GetNodeAPIAddr(nodeAddr string) (string, error)
}

//go:generate mockgen -destination=store_mock.go -package=service github.com/nireo/distsql/service Store

type Store interface {
	Exec(req *pb.Request) ([]*pb.ExecRes, error)
	Query(req *pb.QueryReq) ([]*pb.QueryRes, error)
	LeaderAddr() string
	Join(id, addr string) error
	Leave(id string) error
	GetServers() ([]*pb.Server, error)
	Metrics() (map[string]any, error)
}

// Service handles the http front to the store.
type Service struct {
	listener net.Listener
	addr     string

	manager   Manager
	store     Store
	closeChan chan struct{}

	// security
	CAFile   string
	CertFile string
	KeyFile  string

	log *zap.Logger

	pprof bool
}

type DBResponse struct {
	ExecRes  []*pb.ExecRes
	QueryRes []*pb.QueryRes
}

func (d *DBResponse) MarshalJSON() ([]byte, error) {
	if d.ExecRes != nil {
		return encoding.ProtoToJSON(d.ExecRes)
	} else if d.QueryRes != nil {
		return encoding.ProtoToJSON(d.QueryRes)
	}
	return json.Marshal(make([]interface{}, 0))
}

type DataResponse struct {
	Results *DBResponse `json:"results,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type Config struct {
	EnablePPROF bool
}

func NewService(addr string, store Store, conf Config) (*Service, error) {
	logger, err := zap.NewProduction()
	return &Service{
		addr:  addr,
		store: store,
		log:   logger,
		pprof: conf.EnablePPROF,
	}, err
}

func (s *Service) Close() error {
	close(s.closeChan)
	return s.listener.Close()
}

func (s *Service) Start() error {
	var ln net.Listener
	var err error

	server := http.Server{
		Handler: s,
	}

	if s.CertFile == "" || s.KeyFile == "" {
		ln, err = net.Listen("tcp", s.addr)
		if err != nil {
			return err
		}
	} else {
		// setup https
		conf, err := createTLS(s.CertFile, s.KeyFile, s.CAFile)
		if err != nil {
			return err
		}

		ln, err = tls.Listen("tcp", s.addr, conf)
		if err != nil {
			return err
		}
		s.log.Info("running server as secure HTTPS",
			zap.String("cert_file", s.CertFile),
			zap.String("key_file", s.KeyFile),
		)
	}
	s.listener = ln
	s.closeChan = make(chan struct{})

	go func() {
		err := server.Serve(s.listener)
		if err != nil {
			s.log.Info("server.Serve() failed", zap.Error(err))
		}
	}()
	s.log.Info("service running at addr", zap.String("addr", s.listener.Addr().String()))

	return nil
}

func (s *Service) StartWithListener(ln net.Listener) error {
	server := http.Server{
		Handler: s,
	}
	s.listener = ln
	s.closeChan = make(chan struct{})

	go func() {
		err := server.Serve(s.listener)
		if err != nil {
			s.log.Info("server.Serve() failed", zap.Error(err))
		}
	}()
	s.log.Info("service running at addr", zap.String("addr", s.listener.Addr().String()))

	return nil
}

func createTLS(certFile, keyFile, caFile string) (*tls.Config, error) {
	config := &tls.Config{
		NextProtos: []string{"h2", "http/1.1"},
	}
	var err error
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	if caFile != "" {
		ca, err := os.ReadFile(caFile)
		if err != nil {
			return nil, err
		}

		config.RootCAs = x509.NewCertPool()
		ok := config.RootCAs.AppendCertsFromPEM(ca)
		if !ok {
			return nil, fmt.Errorf("failed to parse root certificate")
		}
	}

	return config, nil
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasPrefix(r.URL.Path, "/join"):
		s.join(w, r)
	case strings.HasPrefix(r.URL.Path, "/leave"):
		s.leave(w, r)
	case strings.HasPrefix(r.URL.Path, "/execute"):
		s.execHandler(w, r)
	case strings.HasPrefix(r.URL.Path, "/query"):
		s.queryHandler(w, r)
	case strings.HasPrefix(r.URL.Path, "/pprof") && s.pprof:
		s.pprofHandler(w, r)
	case strings.HasPrefix(r.URL.Path, "/metric"):
		s.metricHandler(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (s *Service) join(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	b, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	jsonMap := make(map[string]interface{})
	if err := json.Unmarshal(b, &jsonMap); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	id, ok := jsonMap["id"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	addr, ok := jsonMap["addr"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := s.store.Join(id.(string), addr.(string)); err != nil {
		if err == consensus.ErrNotLeader {
			leaderAddr := s.store.LeaderAddr()
			if leaderAddr == "" {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}

			redirect := s.redirectAddr(r, leaderAddr)
			http.Redirect(w, r, redirect, http.StatusMovedPermanently)
			return
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Service) leave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	b, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	jsonMap := make(map[string]interface{})
	if err := json.Unmarshal(b, &jsonMap); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	id, ok := jsonMap["id"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := s.store.Leave(id.(string)); err != nil {
		if err == consensus.ErrNotLeader {
			leaderAddr := s.store.LeaderAddr()
			if leaderAddr == "" {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}

			redirect := s.redirectAddr(r, leaderAddr)
			http.Redirect(w, r, redirect, http.StatusMovedPermanently)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (s *Service) writeResponse(w http.ResponseWriter, r *http.Request, resp *DataResponse) {
	b, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err = w.Write(b); err != nil {
		s.log.Error("response write failed", zap.Error(err))
	}
}

func (s *Service) execHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	b, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	r.Body.Close()

	statements, err := parseStatements(b)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	req := &pb.Request{
		Statements: statements,
	}

	response := &DataResponse{
		Results: &DBResponse{},
	}

	res, err := s.store.Exec(req)
	if err != nil {
		if err == consensus.ErrNotLeader {
			leaderAddr := s.store.LeaderAddr()
			if leaderAddr == "" {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}

			redirect := s.redirectAddr(r, leaderAddr)
			fmt.Println("REDIRECT", redirect)
			http.Redirect(w, r, redirect, http.StatusMovedPermanently)
			return
		}
		response.Error = err.Error()
	} else {
		response.Results.ExecRes = res
	}
	s.writeResponse(w, r, response)
}

func (s *Service) redirectAddr(r *http.Request, url string) string {
	rq := r.URL.RawQuery
	if rq != "" {
		rq = fmt.Sprintf("?%s", rq)
	}
	return fmt.Sprintf("http://%s%s%s", url, r.URL.Path, rq)
}

func parseStatements(b []byte) ([]*pb.Statement, error) {
	var statements []string
	err := json.Unmarshal(b, &statements)
	if err == nil {
		if len(statements) == 0 {
			return nil, ErrEmpty
		}

		res := make([]*pb.Statement, len(statements))
		for i := range statements {
			res[i] = &pb.Statement{
				Sql: statements[i],
			}
		}

		return res, nil
	}

	var withParams [][]interface{}
	if err := json.Unmarshal(b, &withParams); err != nil {
		return nil, ErrBadJson
	}

	res := make([]*pb.Statement, len(withParams))
	for i := range withParams {
		if len(withParams[i]) == 0 {
			return nil, ErrEmpty
		}

		sql, ok := withParams[i][0].(string)
		if !ok {
			return nil, ErrBadJson
		}

		res[i] = &pb.Statement{
			Sql:    sql,
			Params: nil,
		}

		if len(withParams[i]) == 1 {
			continue
		}

		res[i].Params = make([]*pb.Parameter, 0)
		for j := range withParams[i][1:] {
			m, ok := withParams[i][j+1].(map[string]any)
			if ok {
				for k, v := range m {
					param, err := constructParam(k, v)
					if err != nil {
						return nil, err
					}
					res[i].Params = append(res[i].Params, param)
				}
			} else {
				param, err := constructParam("", withParams[i][j+1])
				if err != nil {
					return nil, err
				}
				res[i].Params = append(res[i].Params, param)
			}
		}
	}
	return res, nil
}

func constructParam(name string, val any) (*pb.Parameter, error) {
	switch v := val.(type) {
	case int:
	case int64:
		return &pb.Parameter{
			Value: &pb.Parameter_I{
				I: v,
			},
			Name: name,
		}, nil
	case float64:
		return &pb.Parameter{
			Value: &pb.Parameter_D{
				D: v,
			},
			Name: name,
		}, nil
	case bool:
		return &pb.Parameter{
			Value: &pb.Parameter_B{
				B: v,
			},
			Name: name,
		}, nil
	case []byte:
		return &pb.Parameter{
			Value: &pb.Parameter_Y{
				Y: v,
			},
			Name: name,
		}, nil
	case string:
		return &pb.Parameter{
			Value: &pb.Parameter_S{
				S: v,
			},
			Name: name,
		}, nil
	}

	return nil, errors.New("unrecognized type. Valid types: int, float, byte, string, bool")
}

func (s *Service) IsHTTPS() bool {
	return s.KeyFile != "" && s.CertFile != ""
}

func (s *Service) Addr() net.Addr {
	return s.listener.Addr()
}

func (s *Service) GetLeaderAPIAddr() string {
	nAddr := s.store.LeaderAddr()

	apiAddr, err := s.manager.GetNodeAPIAddr(nAddr)
	if err != nil {
		return ""
	}

	return apiAddr
}

func (s *Service) queryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	resp := &DataResponse{
		Results: &DBResponse{},
	}

	consistency, err := determineConsistency(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	isTransaction, err := checkBoolParam(r, "transaction")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	queries, err := getReqQueries(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	qr := &pb.QueryReq{
		Request: &pb.Request{
			Transaction: isTransaction,
			Statements:  queries,
		},
		StrongConsistency: consistency,
	}

	results, err := s.store.Query(qr)
	if err != nil {
		if err == consensus.ErrNotLeader {
			leaderAddr := s.store.LeaderAddr()
			if leaderAddr == "" {
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}

			redirect := s.redirectAddr(r, leaderAddr)
			http.Redirect(w, r, redirect, http.StatusMovedPermanently)
			return
		}
		resp.Error = err.Error()
	} else {
		resp.Results = &DBResponse{
			QueryRes: results,
		}
	}

	s.writeResponse(w, r, resp)
}

func getReqQueries(r *http.Request) ([]*pb.Statement, error) {
	if r.Method == http.MethodGet {
		query := r.URL.Query()
		stmt := strings.TrimSpace(query.Get("q"))

		if stmt == "" {
			return nil, errors.New("bad query GET request")
		}

		return []*pb.Statement{
			{
				Sql: stmt,
			},
		}, nil
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, errors.New("bad query POST request")
	}
	r.Body.Close()

	return parseStatements(body)
}

func (s *Service) pprofHandler(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/pprof/cmdline":
		pprof.Cmdline(w, r)
	case "/pprof/profile":
		pprof.Profile(w, r)
	case "/pprof/symbol":
		pprof.Symbol(w, r)
	default:
		pprof.Index(w, r)
	}
}

func (s *Service) metricHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	runtimeData := map[string]any{
		"GOARCH":          runtime.GOARCH,
		"GOOS":            runtime.GOOS,
		"GOMAXPROCS":      runtime.GOMAXPROCS(0),
		"cpu_count":       runtime.NumCPU(),
		"goroutine_count": runtime.NumGoroutine(),
		"version":         runtime.Version(),
	}

	storeMetrics, err := s.store.Metrics()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	metricTable := map[string]any{
		"runtime": runtimeData,
		"store":   storeMetrics,
	}

	b, err := json.Marshal(metricTable)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := w.Write([]byte(b)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func checkBoolParam(req *http.Request, param string) (bool, error) {
	err := req.ParseForm()
	if err != nil {
		return false, err
	}

	if _, ok := req.Form[param]; ok {
		return true, nil
	}

	return false, nil
}

// false = no strong consistency; true = strong consistency
func determineConsistency(req *http.Request) (bool, error) {
	queries := req.URL.Query()
	consistency := strings.TrimSpace(queries.Get("consistency"))

	return strings.ToLower(consistency) == "strong", nil
}
