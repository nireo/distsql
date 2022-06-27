package server

import (
	"context"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/nireo/distsql/engine"
	store "github.com/nireo/distsql/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

const (
	objectWildcard = "*"
	execAction     = "execute"
	queryAction    = "query"
)

type Authorizer interface {
	Authorize(subject, object, action string) error
}

type Config struct {
	DB         *engine.Engine
	Authorizer Authorizer
}

type grpcServer struct {
	store.UnimplementedStoreServer
	*Config
}

var _ store.StoreServer = (*grpcServer)(nil)

func newgrpcServer(conf *Config) (srv *grpcServer, err error) {
	srv = &grpcServer{
		Config: conf,
	}
	err = nil
	return
}

func (s *grpcServer) Execute(ctx context.Context, req *store.Request) (
	*store.StoreExecResponse, error,
) {
	if err := s.Authorizer.Authorize(subject(ctx), objectWildcard, execAction); err != nil {
		return nil, err
	}

	results, err := s.DB.Exec(req)
	if err != nil {
		return nil, err
	}

	return &store.StoreExecResponse{
		Results: results,
	}, nil
}

func (s *grpcServer) Query(ctx context.Context, req *store.QueryReq) (
	*store.StoreQueryResponse, error,
) {
	if err := s.Authorizer.Authorize(subject(ctx), objectWildcard, queryAction); err != nil {
		return nil, err
	}

	results, err := s.DB.Query(req.Request)
	if err != nil {
		return nil, err
	}

	return &store.StoreQueryResponse{
		Results: results,
	}, nil
}

func (s *grpcServer) QueryString(ctx context.Context, req *store.QueryStringReq) (
	*store.StoreQueryResponse, error,
) {
	if err := s.Authorizer.Authorize(subject(ctx), objectWildcard, queryAction); err != nil {
		return nil, err
	}

	results, err := s.DB.QueryString(req.Query)
	if err != nil {
		return nil, err
	}

	return &store.StoreQueryResponse{
		Results: results,
	}, nil
}

func (s *grpcServer) ExecString(ctx context.Context, req *store.ExecStringReq) (
	*store.StoreExecResponse, error,
) {
	if err := s.Authorizer.Authorize(subject(ctx), objectWildcard, execAction); err != nil {
		return nil, err
	}

	results, err := s.DB.ExecString(req.Exec)
	if err != nil {
		return nil, err
	}

	return &store.StoreExecResponse{
		Results: results,
	}, nil
}

func authenticate(ctx context.Context) (context.Context, error) {
	peer, ok := peer.FromContext(ctx)
	if !ok {
		return ctx, status.New(codes.Unknown, "couldn't find peer info").Err()
	}

	if peer.AuthInfo == nil {
		return context.WithValue(ctx, subjectContextKey{}, ""), nil
	}

	tlsInfo := peer.AuthInfo.(credentials.TLSInfo)
	subject := tlsInfo.State.VerifiedChains[0][0].Subject.CommonName
	ctx = context.WithValue(ctx, subjectContextKey{}, subject)

	return ctx, nil
}

func subject(ctx context.Context) string {
	return ctx.Value(subjectContextKey{}).(string)
}

type subjectContextKey struct{}

func NewGRPCServer(conf *Config, opts ...grpc.ServerOption) (*grpc.Server, error) {
	opts = append(opts, grpc.StreamInterceptor(
		grpc_middleware.ChainStreamServer(
			grpc_auth.StreamServerInterceptor(authenticate),
		)), grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
		grpc_auth.UnaryServerInterceptor(authenticate),
	)))

	gsrv := grpc.NewServer(opts...)
	srv, err := newgrpcServer(conf)
	if err != nil {
		return nil, err
	}

	store.RegisterStoreServer(gsrv, srv)
	return gsrv, nil
}
