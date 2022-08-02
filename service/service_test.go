package service

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	store "github.com/nireo/distsql/proto"
	"github.com/stretchr/testify/require"
)

func TestEmpty(t *testing.T) {
	b := []byte("[]")

	_, err := parseStatements(b)
	require.NotNil(t, err, "didn't return empty statement")

	b = []byte("[[]]")
	_, err = parseStatements(b)
	require.NotNil(t, err, "didn't return empty statement")
}

func TestSingleStatement(t *testing.T) {
	stmt := "SELECT * FROM test"
	b := []byte(fmt.Sprintf(`["%s"]`, stmt))

	parsed, err := parseStatements(b)
	require.NoError(t, err)
	require.Equal(t, 1, len(parsed))
	require.Equal(t, parsed[0].Sql, stmt)
	require.Nil(t, parsed[0].Params)
}

func TestInvalid(t *testing.T) {
	stmts := [][]byte{
		[]byte(`["SELECT * FROM test]`),
		[]byte(`["SELECT * FROM test"`),
	}

	for _, stmt := range stmts {
		_, err := parseStatements(stmt)
		require.Equal(t, err, ErrBadJson)
	}
}

type testStore struct {
	leaderAddr string
}

func (m *testStore) Execute(er *store.Request) ([]*store.ExecRes, error) {
	return nil, nil
}

func (m *testStore) Query(qr *store.QueryReq) ([]*store.QueryRes, error) {
	return nil, nil
}

func (m *testStore) Join(id, addr string) error {
	return nil
}

func (m *testStore) Leave(id string) error {
	return nil
}

func (m *testStore) LeaderAddr() string {
	return m.leaderAddr
}

func (m *testStore) GetServers() ([]*store.Server, error) {
	return nil, nil
}

func (m *testStore) Metrics() (map[string]any, error) {
	return nil, nil
}

func TestServiceOpen(t *testing.T) {
	ts := &testStore{}
	s, err := NewService("127.0.0.1:0", ts, Config{
		EnablePPROF: true,
	})
	require.NoError(t, err)
	require.NotNil(t, s)

	require.False(t, s.IsHTTPS(), "service shouldn't use HTTPS")
}

func Test_404Routes(t *testing.T) {
	m := &testStore{}
	s, err := NewService("127.0.0.1:0", m, Config{
		EnablePPROF: true,
	})
	require.NoError(t, err)

	if err := s.Start(); err != nil {
		t.Fatalf("failed to start service")
	}
	defer s.Close()
	host := fmt.Sprintf("http://%s", s.Addr().String())

	client := &http.Client{}
	resp, err := client.Get(host + "/blahblahblah")
	require.NoError(t, err)
	require.Equal(t, resp.StatusCode, http.StatusNotFound)
	resp, err = client.Post(host+"/xxx", "", nil)

	require.NoError(t, err)
	require.Equal(t, resp.StatusCode, http.StatusNotFound)
}

func TestPPROFRoutes(t *testing.T) {
	m := &testStore{}
	s, err := NewService("127.0.0.1:0", m, Config{
		EnablePPROF: true,
	})
	require.NoError(t, err)

	if err := s.Start(); err != nil {
		t.Fatalf("failed to start service")
	}
	defer s.Close()

	host := fmt.Sprintf("http://%s", s.Addr().String())

	client := &http.Client{}
	resp, err := client.Get(host + "/pprof")
	require.NoError(t, err)
	require.Equal(t, resp.StatusCode, http.StatusOK)
}

func TestPPROFFail(t *testing.T) {
	m := &testStore{}
	s, err := NewService("127.0.0.1:0", m, Config{
		EnablePPROF: false,
	})
	require.NoError(t, err)

	if err := s.Start(); err != nil {
		t.Fatalf("failed to start service")
	}
	defer s.Close()

	host := fmt.Sprintf("http://%s", s.Addr().String())
	resp, err := http.Get(host + "/pprof")
	require.NoError(t, err)

	require.Equal(t, resp.StatusCode, http.StatusNotFound)
}

func TestMetrics(t *testing.T) {
	m := &testStore{}

	s, err := NewService("127.0.0.1:0", m, Config{
		EnablePPROF: false,
	})
	require.NoError(t, err)

	if err := s.Start(); err != nil {
		t.Fatalf("failed to start service")
	}
	defer s.Close()

	host := fmt.Sprintf("http://%s", s.Addr().String())
	resp, err := http.Get(host + "/metric")
	require.NoError(t, err)

	require.Equal(t, resp.Header.Get("Content-Type"), "application/json; charset=utf-8")
	require.Equal(t, resp.StatusCode, http.StatusOK)

	data, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	var jsonData map[string]any
	err = json.Unmarshal(data, &jsonData)
	require.NoError(t, err)

	fields := []string{
		"GOARCH",
		"GOOS",
		"GOMAXPROCS",
		"cpu_count",
		"goroutine_count",
		"version",
	}

	fmt.Println(jsonData)
	runtimeMap := jsonData["runtime"].(map[string]any)

	for _, field := range fields {
		_, ok := runtimeMap[field]
		require.True(t, ok)
	}
}
