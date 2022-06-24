package server

import (
	"context"

	"github.com/nireo/distsql/engine"
	store "github.com/nireo/distsql/proto"
	"google.golang.org/grpc"
)

type Config struct {
	db *engine.Engine
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
	results, err := s.db.Exec(req)
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
	results, err := s.db.Query(req.Request)
	if err != nil {
		return nil, err
	}

	return &store.StoreQueryResponse{
		Results: results,
	}, nil
}

func NewGRPCServer(conf *Config) (*grpc.Server, error) {
	gsrv := grpc.NewServer()
	srv, err := newgrpcServer(conf)
	if err != nil {
		return nil, err
	}

	store.RegisterStoreServer(gsrv, srv)
	return gsrv, nil
}
