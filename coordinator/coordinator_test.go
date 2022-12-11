package coordinator_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/nireo/distsql/coordinator"
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

	return body
}

func setupNServices(t *testing.T, n int) []*coordinator.Coordinator {
	var services []*coordinator.Coordinator

	for i := 0; i < 3; i++ {
		ports := genNPorts(2)
		bindaddr := fmt.Sprintf("%s:%d", "127.0.0.1", ports[0])
		rpcPort := ports[1]

		datadir, err := os.MkdirTemp("", "service-test")
		require.NoError(t, err)

		var startJoinAddrs []string
		if i != 0 {
			startJoinAddrs = append(startJoinAddrs, services[0].Config.BindAddr)
		}

		service, err := coordinator.New(coordinator.Config{
			NodeName:          fmt.Sprintf("%d", i),
			Bootstrap:         i == 0,
			StartJoinAddrs:    startJoinAddrs,
			BindAddr:          bindaddr,
			DataDir:           datadir,
			CommunicationPort: rpcPort,
		})
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
	time.Sleep(2 * time.Second)

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
