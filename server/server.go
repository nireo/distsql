package server

import (
	"context"

	"github.com/nireo/distsql/engine"
	store "github.com/nireo/distsql/proto"
)

type grpcServer struct {
	store.UnimplementedStoreServer
	db *engine.Engine
}

var _ store.StoreServer = (*grpcServer)(nil)

func newgrpcServer(db *engine.Engine) (srv *grpcServer, err error) {
	srv = &grpcServer{
		db: db,
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
	return nil, nil
}
