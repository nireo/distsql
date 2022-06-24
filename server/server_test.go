package server

import (
	"io/ioutil"
	"net"
	"path/filepath"
	"testing"

	"github.com/nireo/distsql/engine"
	store "github.com/nireo/distsql/proto"
	"google.golang.org/grpc"
)

type setupFunc = func(*Config)

func setupTest(t *testing.T, fn setupFunc) (
	client store.StoreClient, cfg *Config, teardown func(),
) {
	t.Helper()

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	opts := []grpc.DialOption{grpc.WithInsecure()}
	cc, err := grpc.Dial(listener.Addr().String(), opts...)
	if err != nil {
		t.Fatal(err)
	}

	dir, err := ioutil.TempDir("", "server-test-")
	if err != nil {
		t.Fatal(err)
	}
	dbpath := filepath.Join(dir, "data.db")

	db, err := engine.Open(dbpath)
	if err != nil {
		t.Fatal(err)
	}

	cfg = &Config{
		db: db,
	}

	if fn != nil {
		fn(cfg)
	}

	srv, err := NewGRPCServer(cfg)
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		srv.Serve(listener)
	}()

	client = store.NewStoreClient(cc)
	return client, cfg, func() {
		srv.Stop()
		cc.Close()
		listener.Close()
		db.Close()
	}
}

func TestServer(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
		client store.StoreClient,
		config *Config,
	){
		"execute/query a record into the database": testExecQuery,
	} {
		t.Run(scenario, func(t *testing.T) {
			client, config, teardown := setupTest(t, nil)
			defer teardown()
			fn(t, client, config)
		})
	}
}

func testExecQuery(t *testing.T, client store.StoreClient, config *Config) {
}
