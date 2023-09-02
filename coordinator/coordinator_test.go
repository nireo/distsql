package coordinator_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nireo/distsql/coordinator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}

func genNPorts(n int) []int {
	ports := make([]int, n)
	for i := 0; i < n; i++ {
		ports[i], _ = getFreePort()
	}
	return ports
}

func httpGetHelper(t *testing.T, addr string) []byte {
	t.Helper()

	resp, err := http.Get(addr)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	return body
}

func sendExecStmts(t *testing.T, addr string, statements []string) []byte {
	t.Helper()

	body, err := json.Marshal(statements)
	require.NoError(t, err)

	r, err := http.NewRequest("POST", strings.TrimSpace(addr+"/execute"), bytes.NewBuffer(body))
	require.NoError(t, err)

	client := &http.Client{}
	res, err := client.Do(r)
	require.NoError(t, err)

	require.Equal(t, res.StatusCode, http.StatusOK)

	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	return b
}

func sendQueryStmts(t *testing.T, addr string, statements []string) []byte {
	t.Helper()

	body, err := json.Marshal(statements)
	require.NoError(t, err)

	r, err := http.NewRequest("POST", addr+"/query", bytes.NewBuffer(body))
	require.NoError(t, err)

	client := &http.Client{}
	res, err := client.Do(r)
	require.NoError(t, err)

	require.Equal(t, res.StatusCode, http.StatusOK)

	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	return b
}

func setupNServices(t *testing.T, n int) []*coordinator.Coordinator {
	var services []*coordinator.Coordinator

	for i := 0; i < 3; i++ {
		ports := genNPorts(2)
		bindaddr := fmt.Sprintf("%s:%d", "localhost", ports[0])
		rpcPort := ports[1]

		datadir, err := os.MkdirTemp("", "service-test")
		require.NoError(t, err)

		var startJoinAddrs []string
		if i != 0 {
			startJoinAddrs = append(startJoinAddrs, services[0].Config.BindAddr)
		}

		conf := coordinator.Config{
			NodeName:          fmt.Sprintf("%d", i),
			Bootstrap:         i == 0,
			StartJoinAddrs:    startJoinAddrs,
			BindAddr:          bindaddr,
			DataDir:           datadir,
			CommunicationPort: rpcPort,
		}
		if i != 0 {
			conf.LeaderStartAddr, err = services[0].Config.HTTPAddr()
			require.NoError(t, err)
		}

		service, err := coordinator.New(conf)
		require.NoError(t, err)

		services = append(services, service)
	}

	t.Cleanup(func() {
		for _, s := range services {
			s.Close()
			require.NoError(t, os.RemoveAll(s.Config.DataDir))
		}
	})

	return services
}

func TestGetMetrics(t *testing.T) {
	coordinators := setupNServices(t, 3)
	time.Sleep(3 * time.Second)

	leaderaddr, err := coordinators[0].Config.HTTPAddr()
	require.NoError(t, err)

	var jsonMap map[string]any
	data := httpGetHelper(t, leaderaddr+"/metric")

	err = json.Unmarshal(data, &jsonMap)
	require.NoError(t, err)

	_, ok := jsonMap["store"]
	require.True(t, ok)

	_, ok = jsonMap["runtime"]
	require.True(t, ok)
}

func TestLeaderOperations(t *testing.T) {
	coordinators := setupNServices(t, 3)
	time.Sleep(2 * time.Second)

	leaderAddr, err := coordinators[0].Config.HTTPAddr()
	require.NoError(t, err)

	data := sendExecStmts(t, leaderAddr, []string{
		`CREATE TABLE test (id INTEGER NOT NULL PRIMARY KEY, name TEXT)`,
		`INSERT INTO test(name) VALUES("atest")`,
		`INSERT INTO test(name) VALUES("btest")`,
	})

	data2 := sendQueryStmts(t, leaderAddr, []string{
		`SELECT * FROM test`,
	})

	require.Equal(t, `{"results":[{},{"last_insert_id":1,"rows_affected":1},{"last_insert_id":2,"rows_affected":1}]}`, string(data))
	require.Equal(t, `{"results":[{"columns":["id","name"],"types":["integer","text"],"values":[[1,"atest"],[2,"btest"]]}]}`, string(data2))
}

// func TestWriteRequestsToLeaderWork(t *testing.T) {
// 	coordinators := setupNServices(t, 3)
// 	time.Sleep(6 * time.Second)
//
// 	followerAddr1, err := coordinators[1].Config.HTTPAddr()
// 	require.NoError(t, err)
//
// 	fmt.Println("sending exec addr to", followerAddr1)
// 	data := sendExecStmts(t, followerAddr1, []string{
// 		`CREATE TABLE test (id INTEGER NOT NULL PRIMARY KEY, name TEXT)`,
// 		`INSERT INTO test(name) VALUES("atest")`,
// 		`INSERT INTO test(name) VALUES("btest")`,
// 	})
//
// 	require.Equal(t, `{"results":[{},{"last_insert_id":1,"rows_affected":1},{"last_insert_id":2,"rows_affected":1}]}`, string(data))
// }
