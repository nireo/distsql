package consensus

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/raft"
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

func TestSetupNodes(t *testing.T) {
	var dbs []*Consensus
	var err error
	nodeCount := 3

	ports := make([]int, nodeCount)
	for i := 0; i < nodeCount; i++ {
		ports[i], err = getFreePort()
		require.NoError(t, err)
	}

	for i := 0; i < nodeCount; i++ {
		dataDir, err := ioutil.TempDir("", "consensus-test-")
		require.NoError(t, err)

		defer func(dir string) {
			_ = os.RemoveAll(dir)
		}(dataDir)

		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", ports[i]))
		require.NoError(t, err)

		config := Config{}
		config.Raft.StreamLayer = NewStreamLayer(ln, nil, nil)
		config.Raft.LocalID = raft.ServerID(fmt.Sprintf("%d", i))
		config.Raft.HeartbeatTimeout = 50 * time.Millisecond
		config.Raft.ElectionTimeout = 50 * time.Millisecond
		config.Raft.LeaderLeaseTimeout = 50 * time.Millisecond
		config.Raft.CommitTimeout = 5 * time.Millisecond

		if i == 0 {
			config.Raft.Bootstrap = true
		}

		db, err := NewDB(dataDir, &config)
		require.NoError(t, err)

		if i != 0 {
			err = dbs[0].Join(fmt.Sprintf("%d", i), ln.Addr().String())
			require.NoError(t, err)
		} else {
			err = db.WaitForLeader(3 * time.Second)
			require.NoError(t, err)
		}

		dbs = append(dbs, db)
	}

	// executes := &store.Request{
	//	Statements: []*store.Statement{
	//		{
	//			Sql: "CREATE TABLE test (id INTEGER NOT NULL PRIMARY KEY, name TEXT)",
	//		},
	//		{
	//			Sql: `INSERT INTO foo(name) VALUES("aoife")`,
	//		},
	//		{
	//			Sql: `INSERT INTO foo(name) VALUSE("fiona)`,
	//		},
	//	},
	// }
}
