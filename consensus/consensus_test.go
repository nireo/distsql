package consensus

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/raft"
	"github.com/nireo/distsql/pb"
	"github.com/nireo/distsql/pb/encoding"
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

func newTestStore(t *testing.T, port, id int, bootstrap bool) (*Consensus, error) {
	datadir, err := os.MkdirTemp("", "store-test")
	require.NoError(t, err)

	conf := &Config{}
	conf.Raft.LocalID = raft.ServerID(fmt.Sprintf("%d", id))
	conf.Raft.HeartbeatTimeout = 50 * time.Millisecond
	conf.Raft.ElectionTimeout = 50 * time.Millisecond
	conf.Raft.Bootstrap = bootstrap
	conf.Raft.LeaderLeaseTimeout = 50 * time.Millisecond
	conf.Raft.CommitTimeout = 5 * time.Millisecond
	conf.Raft.SnapshotThreshold = 10000

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	require.NoError(t, err)
	conf.Transport = &Transport{
		ln: ln,
	}

	s, err := NewDB(datadir, conf)
	if err != nil {
		return nil, err
	}

	t.Cleanup(func() {
		os.RemoveAll(datadir)
		s.Close()
	})

	return s, nil
}

func TestSimpleUsage(t *testing.T) {
	var err error
	nodeCount := 3
	dbs := make([]*Consensus, nodeCount)

	for i := 0; i < nodeCount; i++ {
		port, err := getFreePort()
		require.NoError(t, err)

		dbs[i], err = newTestStore(t, port, i, i == 0)
		require.NoError(t, err)

		if i != 0 {
			err = dbs[0].Join(fmt.Sprintf("%d", i), dbs[i].config.Transport.Addr().String())
			require.NoError(t, err)
		} else {
			err = dbs[0].WaitForLeader(3 * time.Second)
			require.NoError(t, err)
		}
	}

	executes := &pb.Request{
		Statements: []*pb.Statement{
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
				r, err := dbs[j].Query(&pb.QueryReq{
					StrongConsistency: false, // don't query everything from the leader
					Request: &pb.Request{
						Statements: []*pb.Statement{
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
					fmt.Printf(
						"On query: %s | Node: %d\n\twanted: %s\n\tgot:%s\n",
						q.query,
						j,
						want,
						got,
					)
					return false
				}
			}
			return true
		}, 3*time.Second, 150*time.Millisecond)
	}

	err = dbs[0].Leave("1")
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)

	executes = &pb.Request{
		Statements: []*pb.Statement{
			{
				Sql: `INSERT INTO test(name) VALUES("ctest")`,
			},
		},
	}
	_, err = dbs[0].Exec(executes)
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	qr := &pb.QueryReq{
		StrongConsistency: false,
		Request: &pb.Request{
			Statements: []*pb.Statement{
				{
					Sql: `SELECT * FROM test WHERE name="ctest"`,
				},
			},
		},
	}

	rr, err := dbs[1].Query(qr)
	require.NoError(t, err)
	require.Equal(t, len(rr[0].Values), 0)

	rr, err = dbs[2].Query(qr)
	require.NoError(t, err)
	require.Equal(t, len(rr[0].Values), 1)
}

func convertToJSON(a any) string {
	j, err := encoding.ProtoToJSON(a)
	if err != nil {
		return ""
	}

	return string(j)
}
