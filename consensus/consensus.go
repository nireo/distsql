package consensus

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/nireo/distsql/engine"
	store "github.com/nireo/distsql/proto"
	"go.uber.org/zap"
)

type RequestType uint8

const (
	ExecRequest RequestType = iota

	RaftRPC int = 1
)

type Config struct {
	Raft struct {
		raft.Config
		StreamLayer       *StreamLayer
		Bootstrap         bool
		SnapshotThreshold uint64
	}
}

type StreamLayer struct {
	listener        net.Listener
	serverTLSConfig *tls.Config
	peerTLSConfig   *tls.Config
}

type Consensus struct {
	config  *Config
	running bool
	raftDir string
	raft    *raft.Raft
	log     *zap.Logger
	db      *engine.Engine
}

type snapshot struct {
	created  time.Time
	database []byte // the serialized database
}

func newSnapshot(db *engine.Engine) *snapshot {
	data, _ := db.Serialize()

	return &snapshot{
		created:  time.Now(),
		database: data,
	}
}

func (snapshot *snapshot) Persist(sink raft.SnapshotSink) error {
	return errors.New("not implemented")
}

func (snapshot *snapshot) Release() {}

func NewStreamLayer(ln net.Listener, serverTLSConfig,
	peerTLSConfig *tls.Config) *StreamLayer {
	return &StreamLayer{
		listener:        ln,
		serverTLSConfig: serverTLSConfig,
		peerTLSConfig:   peerTLSConfig,
	}
}

func (s *StreamLayer) Dial(addr raft.ServerAddress, timeout time.Duration) (
	net.Conn, error,
) {
	dialer := &net.Dialer{Timeout: timeout}
	var conn, err = dialer.Dial("tcp", string(addr))
	if err != nil {
		return nil, err
	}

	_, err = conn.Write([]byte{byte(RaftRPC)})
	if err != nil {
		return nil, err
	}

	if s.peerTLSConfig != nil {
		conn = tls.Client(conn, s.peerTLSConfig)
	}
	return conn, err
}

func (s *StreamLayer) Accept() (net.Conn, error) {
	conn, err := s.listener.Accept()
	if err != nil {
		return nil, err
	}

	b := make([]byte, 1)
	if _, err := conn.Read(b); err != nil {
		return nil, err
	}

	if bytes.Compare([]byte{(byte(RaftRPC))}, b) != 0 {
		return nil, fmt.Errorf("not a raft rpc")
	}

	if s.serverTLSConfig != nil {
		return tls.Server(conn, s.serverTLSConfig), nil
	}

	return conn, nil
}

func (s *StreamLayer) Close() error {
	return s.listener.Close()
}

func (s *StreamLayer) Addr() net.Addr {
	return s.listener.Addr()
}

func (c *Consensus) setupRaft(dataDir string) error {
	if err := os.Mkdir(filepath.Join(dataDir, "raft"), os.ModePerm); err != nil {
		return err
	}

	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(dataDir, "raft", "raft.db"))
	if err != nil {
		return err
	}

	snapshotStore, err := raft.NewFileSnapshotStore(filepath.Join(dataDir, "raft"), 1, os.Stderr)
	if err != nil {
		return err
	}

	maxPool := 5
	timeout := 10 * time.Second
	transport := raft.NewNetworkTransport(
		c.config.Raft.StreamLayer,
		maxPool,
		timeout,
		os.Stderr,
	)

	config := raft.DefaultConfig()
	config.SnapshotThreshold = c.config.Raft.SnapshotThreshold
	config.LocalID = c.config.Raft.LocalID

	if c.config.Raft.HeartbeatTimeout != 0 {
		config.HeartbeatTimeout = c.config.Raft.HeartbeatTimeout
	}

	if c.config.Raft.ElectionTimeout != 0 {
		config.ElectionTimeout = c.config.Raft.ElectionTimeout
	}

	if c.config.Raft.LeaderLeaseTimeout != 0 {
		config.LeaderLeaseTimeout = c.config.Raft.LeaderLeaseTimeout
	}

	if c.config.Raft.CommitTimeout != 0 {
		config.CommitTimeout = c.config.Raft.CommitTimeout
	}

	c.raft, err = raft.NewRaft(config, c, stableStore, stableStore, snapshotStore, transport)
	if err != nil {
		return err
	}

	hasState, err := raft.HasExistingState(stableStore, stableStore, snapshotStore)
	if err != nil {
		return err
	}

	if c.config.Raft.Bootstrap && !hasState {
		config := raft.Configuration{
			Servers: []raft.Server{{
				ID:      config.LocalID,
				Address: transport.LocalAddr(),
			}},
		}
		err = c.raft.BootstrapCluster(config).Error()
	}
	return err
}

func (c *Consensus) Apply(record *raft.Log) interface{} {
	return nil
}

func (c *Consensus) Restore(rc io.ReadCloser) error {
	return nil
}

func (c *Consensus) Snapshot() (raft.FSMSnapshot, error) {
	return nil, nil
}

func (c *Consensus) IsLeader() bool {
	return c.raft.State() == raft.Leader
}

// Close closes the raft node and it also closes the database connections
func (c *Consensus) Close() error {
	future := c.raft.Shutdown()
	if err := future.Error(); err != nil {
		return err
	}

	return c.db.Close()
}

func (c *Consensus) GetServers() ([]*store.Server, error) {
	future := c.raft.GetConfiguration()
	if err := future.Error(); err != nil {
		return nil, err
	}

	var servers []*store.Server
	for _, server := range future.Configuration().Servers {
		servers = append(servers, &store.Server{
			Id:       string(server.ID),
			RpcAddr:  string(server.Address),
			IsLeader: c.raft.Leader() == server.Address,
		})
	}

	return servers, nil
}

func (c *Consensus) Join(id, addr string) error {
	future := c.raft.GetConfiguration()
	if err := future.Error(); err != nil {
		return err
	}

	// convert types
	serverID := raft.ServerID(id)
	serverAddr := raft.ServerAddress(addr)

	for _, srv := range future.Configuration().Servers {
		if srv.ID == serverID || srv.Address == serverAddr {
			if srv.ID == serverID && srv.Address == serverAddr {
				return nil
			}

			removeFuture := c.raft.RemoveServer(serverID, 0, 0)
			if err := removeFuture.Error(); err != nil {
				return err
			}
		}
	}

	addFuture := c.raft.AddVoter(serverID, serverAddr, 0, 0)
	if err := addFuture.Error(); err != nil {
		return err
	}
	return nil
}

func (c *Consensus) Leave(id string) error {
	removeFuture := c.raft.RemoveServer(raft.ServerID(id), 0, 0)
	return removeFuture.Error()
}

func (c *Consensus) WaitForLeader(timeout time.Duration) error {
	timeoutc := time.After(timeout)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutc:
			return fmt.Errorf("timed out")
		case <-ticker.C:
			if l := c.raft.Leader(); l != "" {
				return nil
			}
		}
	}
}
