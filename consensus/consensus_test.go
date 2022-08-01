package consensus

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/raft"
	store "github.com/nireo/distsql/proto"
	"github.com/nireo/distsql/proto/encoding"
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

func TestSimpleUsage(t *testing.T) {
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

	executes := &store.Request{
		Statements: []*store.Statement{
			{
				Sql: "CREATE TABLE test (id INTEGER NOT NULL PRIMARY KEY, name TEXT)",
			},
			{
				Sql: `INSERT INTO test(name) VALUES("atest")`,
			},
			{
				Sql: `INSERT INTO test(name) VALUES("btest")`,
			},
		},
	}

	_, err = dbs[0].Exec(executes)
	require.NoError(t, err)

	type queryWanted struct {
		query        string
		wantedOutput string
	}

	queries := []queryWanted{
		{
			query:        `SELECT * FROM test`,
			wantedOutput: `[{"columns":["id","name"],"types":["integer","text"],"values":[[1,"atest"],[2,"btest"]]}]`,
		},
		{
			query:        `SELECT * FROM test WHERE name="atest"`,
			wantedOutput: `[{"columns":["id","name"],"types":["integer","text"],"values":[[1,"atest"]]}]`,
		},
		{
			query:        `SELECT * FROM test WHERE name="btest"`,
			wantedOutput: `[{"columns":["id","name"],"types":["integer","text"],"values":[[2,"btest"]]}]`,
		},
	}

	for _, q := range queries {
		require.Eventually(t, func() bool {
			for j := 0; j < nodeCount; j++ {
				r, err := dbs[j].Query(&store.QueryReq{
					StrongConsistency: false, // don't query everything from the leader
					Request: &store.Request{
						Statements: []*store.Statement{
							{Sql: q.query},
						},
					},
				})
				if err != nil {
					return false
				}

				if r == nil {
					return false
				}

				js := convertToJSON(r)
				if got, want := js, q.wantedOutput; got != want {
					fmt.Printf("On query: %s | Node: %d\n\twanted: %s\n\tgot:%s\n", q.query, j, want, got)
					return false
				}
			}
			return true
		}, 3*time.Second, 150*time.Millisecond)
	}

	err = dbs[0].Leave("1")
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)

	executes = &store.Request{
		Statements: []*store.Statement{
			{
				Sql: `INSERT INTO test(name) VALUES("ctest")`,
			},
		},
	}
	_, err = dbs[0].Exec(executes)
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	qr := &store.QueryReq{
		StrongConsistency: false,
		Request: &store.Request{
			Statements: []*store.Statement{
				{
					Sql: `SELECT * FROM test WHERE name="ctest"`,
				},
			},
		},
	}

	rr, err := dbs[1].Query(qr)
	require.Equal(t, len(rr[0].Values), 0)

	rr, err = dbs[2].Query(qr)
	require.Equal(t, len(rr[0].Values), 1)
}

func convertToJSON(a any) string {
	j, err := encoding.ProtoToJSON(a)
	if err != nil {
		return ""
	}

	return string(j)
}
