package server

import (
	"context"
	"io/ioutil"
	"net"
	"path/filepath"
	"testing"

	"github.com/nireo/distsql/auth"
	"github.com/nireo/distsql/config"
	"github.com/nireo/distsql/engine"
	store "github.com/nireo/distsql/proto"
	"github.com/nireo/distsql/proto/encoding"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

func setupTest(t *testing.T, fn func(*Config)) (
	rootClient store.StoreClient,
	nobodyClient store.StoreClient,
	cfg *Config,
	teardown func(),
) {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	newClient := func(crtPath, keyPath string) (
		*grpc.ClientConn,
		store.StoreClient,
		[]grpc.DialOption,
	) {
		tlsConfig, err := config.SetupTLSConfig(config.TLSConfig{
			CertFile: crtPath,
			KeyFile:  keyPath,
			CAFile:   config.CAFile,
			Server:   false,
		})
		require.NoError(t, err)

		tlsCreds := credentials.NewTLS(tlsConfig)
		opts := []grpc.DialOption{grpc.WithTransportCredentials(tlsCreds)}
		conn, err := grpc.Dial(l.Addr().String(), opts...)
		require.NoError(t, err)
		client := store.NewStoreClient(conn)

		return conn, client, opts
	}

	var rootConn *grpc.ClientConn
	rootConn, rootClient, _ = newClient(
		config.RootClientCertFile,
		config.RootClientKeyFile,
	)
	var nobodyConn *grpc.ClientConn
	nobodyConn, nobodyClient, _ = newClient(
		config.NobodyClientCertFile,
		config.NobodyClientKeyFile,
	)

	serverTLSConfig, err := config.SetupTLSConfig(config.TLSConfig{
		CertFile:      config.ServerCertFile,
		KeyFile:       config.ServerKeyFile,
		CAFile:        config.CAFile,
		ServerAddress: l.Addr().String(),
		Server:        true,
	})
	require.NoError(t, err)

	serverCreds := credentials.NewTLS(serverTLSConfig)

	dir, err := ioutil.TempDir("", "server-test-")
	require.NoError(t, err)
	dbpath := filepath.Join(dir, "test.db")

	cdb, err := engine.Open(dbpath)
	require.NoError(t, err)

	authorizer := auth.New(config.ACLModelFile, config.ACLPolicyFile)
	cfg = &Config{DB: cdb, Authorizer: authorizer}
	if fn != nil {
		fn(cfg)
	}
	server, err := NewGRPCServer(cfg, grpc.Creds(serverCreds))
	require.NoError(t, err)

	go func() {
		server.Serve(l)
	}()

	return rootClient, nobodyClient, cfg, func() {
		server.Stop()
		rootConn.Close()
		nobodyConn.Close()
		l.Close()
		cdb.Close()
	}
}

func TestServer(t *testing.T) {
	for scenario, fn := range map[string]func(
		t *testing.T,
		rootClient store.StoreClient,
		nobodyClient store.StoreClient,
		config *Config,
	){
		"execute/query a record into the database": testExecQuery,
		"table not found":                          testNotFound,
		"unauthorized access failed":               testUnauthorized,
	} {
		t.Run(scenario, func(t *testing.T) {
			rootClient, nobodyClient, config, teardown := setupTest(t, nil)
			defer teardown()
			fn(t, rootClient, nobodyClient, config)
		})
	}
}

func testExecQuery(t *testing.T, client, _ store.StoreClient, config *Config) {
	ctx := context.Background()
	results, err := client.ExecString(ctx, &store.ExecStringReq{
		Exec: "CREATE TABLE test (id INTEGER NOT NULL PRIMARY KEY, name TEXT)",
	})
	require.NoError(t, err)

	jsonRes := convertToJSON(results.Results)
	if jsonRes != "[{}]" {
		t.Fatalf("unrecognized output. want=%s got=%s", "[{}]", jsonRes)
	}

	query, err := client.QueryString(ctx, &store.QueryStringReq{
		Query: "SELECT * FROM test",
	})
	require.NoError(t, err)

	qjson := convertToJSON(query.Results)
	if qjson != `[{"columns":["id","name"],"types":["integer","text"]}]` {
		t.Fatalf("results don't match: %s", qjson)
	}
}

func testNotFound(t *testing.T, client, _ store.StoreClient, config *Config) {
	ctx := context.Background()

	query, err := client.QueryString(ctx, &store.QueryStringReq{
		Query: "SELECT * FROM test",
	})
	require.NoError(t, err)

	qjson := convertToJSON(query.Results)
	if qjson != `[{"error":"no such table: test"}]` {
		t.Fatalf("results don't match: %s", qjson)
	}
}

func testUnauthorized(t *testing.T, _, client store.StoreClient, config *Config) {
	ctx := context.Background()

	exec, err := client.ExecString(ctx, &store.ExecStringReq{
		Exec: "CREATE TABLE test (id INTEGER NOT NULL PRIMARY KEY, name TEXT)",
	})
	require.Nil(t, exec, "exec response should be nil")

	gotCode, wantCode := status.Code(err), codes.PermissionDenied
	if gotCode != wantCode {
		t.Fatalf("got code: %d, want: %d", gotCode, wantCode)
	}

	query, err := client.QueryString(ctx, &store.QueryStringReq{
		Query: "SELECT * FROM test",
	})
	require.Nil(t, query, "query should be nil")

	gotCode, wantCode = status.Code(err), codes.PermissionDenied
	if gotCode != wantCode {
		t.Fatalf("got code: %d, want: %d", gotCode, wantCode)
	}
}

func convertToJSON(a any) string {
	j, err := encoding.ProtoToJSON(a)
	if err != nil {
		return ""
	}

	return string(j)
}
