package service

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	"github.com/nireo/distsql/consensus"
	store "github.com/nireo/distsql/proto"
	"go.uber.org/zap"
)

var (
	ErrBadJson = errors.New("bad json couldn't be parsed properly")
	ErrEmpty   = errors.New("no statements were provided")
)

// Store represents a raft store.
type Store interface {
	Execute(req *store.Request) ([]*store.ExecRes, error)
	Query(req *store.QueryReq) ([]*store.QueryRes, error)
	LeaderAddr() string
	Join(id, addr string) error
	Leave(id string) error
	GetServers() ([]*store.Server, error)
}

type Service struct {
	listener net.Listener
	addr     string

	store     Store
	closeChan chan struct{}

	// security
	CAFile   string
	CertFile string
	KeyFile  string

	log *zap.Logger
}

type DBResponse struct {
	ExecRes  []*store.ExecRes
	QueryRes []*store.QueryRes
}

type DataResponse struct {
	Results *DBResponse `json:"results,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func NewService(addr string, store Store) (*Service, error) {
	logger, err := zap.NewProduction()
	return &Service{
		addr:  addr,
		store: store,
		log:   logger,
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
		ca, err := ioutil.ReadFile(caFile)
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
	}
}

func (s *Service) join(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	b, err := ioutil.ReadAll(r.Body)
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
			// TODO: redirect to leader
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Service) leave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	b, err := ioutil.ReadAll(r.Body)
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
			// TODO: redirect to leader
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (s *Service) execHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	b, err := ioutil.ReadAll(r.Body)
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

	req := &store.Request{
		Statements: statements,
	}

	response := &DataResponse{
		Results: &DBResponse{},
	}

	res, err := s.store.Execute(req)
	if err != nil {
		response.Error = err.Error()
	} else {
		response.Results.ExecRes = res
	}

	b, err = json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = w.Write(b)
	if err != nil {
		s.log.Info("response write failed", zap.Error(err))
	}
}

func (s *Service) redirectAddr(r *http.Request, url string) string {
	rq := r.URL.RawQuery
	if rq != "" {
		rq = fmt.Sprintf("?%s", rq)
	}
	return fmt.Sprintf("%s%s%s", url, r.URL.Path, rq)
}

func parseStatements(b []byte) ([]*store.Statement, error) {
	var statements []string
	err := json.Unmarshal(b, &statements)
	if err == nil {
		if len(statements) == 0 {
			return nil, ErrEmpty
		}

		res := make([]*store.Statement, len(statements))
		for i := range statements {
			res[i] = &store.Statement{
				Sql: statements[i],
			}
		}

		return res, nil
	}

	var withParams [][]interface{}
	if err := json.Unmarshal(b, &withParams); err != nil {
		return nil, ErrBadJson
	}

	res := make([]*store.Statement, len(withParams))
	for i := range withParams {
		if len(withParams[i]) == 0 {
			return nil, ErrEmpty
		}

		sql, ok := withParams[i][0].(string)
		if !ok {
			return nil, ErrBadJson
		}

		res[i] = &store.Statement{
			Sql:    sql,
			Params: nil,
		}

		if len(withParams[i]) == 1 {
			continue
		}

		res[i].Params = make([]*store.Parameter, 0)
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

func constructParam(name string, val any) (*store.Parameter, error) {
	switch v := val.(type) {
	case int:
	case int64:
		return &store.Parameter{
			Value: &store.Parameter_I{
				I: v,
			},
			Name: name,
		}, nil
	case float64:
		return &store.Parameter{
			Value: &store.Parameter_D{
				D: v,
			},
			Name: name,
		}, nil
	case bool:
		return &store.Parameter{
			Value: &store.Parameter_B{
				B: v,
			},
			Name: name,
		}, nil
	case []byte:
		return &store.Parameter{
			Value: &store.Parameter_Y{
				Y: v,
			},
			Name: name,
		}, nil
	case string:
		return &store.Parameter{
			Value: &store.Parameter_S{
				S: v,
			},
			Name: name,
		}, nil
	}

	return nil, errors.New("unrecognized type. Valid types: int, float, byte, string, bool")
}
