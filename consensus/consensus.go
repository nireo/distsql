package consensus

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/nireo/distsql/engine"
	"github.com/nireo/distsql/pb"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type RequestType uint8

const (
	ExecRequest RequestType = iota

	RaftRPC int = 1
)

var (
	ErrNotLeader = errors.New("not raft leader")
	ErrJoinSelf  = errors.New("trying to join self")
)

type Config struct {
	Raft struct {
		raft.Config
		Bootstrap         bool
		SnapshotThreshold uint64
	}
	Transport *Transport
}

type Consensus struct {
	config  *Config
	running bool
	raftDir string
	raft    *raft.Raft
	log     *zap.Logger
	db      *engine.Engine
	dbPath  string
	txmu    sync.RWMutex
	lastIdx uint64
	idxmu   sync.RWMutex
	tn      *raft.NetworkTransport
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

type execResponse struct {
	res   []*pb.ExecRes
	error error
}

type queryResponse struct {
	res   []*pb.QueryRes
	error error
}

func (f *snapshot) compressDatabase() ([]byte, error) {
	if f.database == nil {
		return nil, nil
	}

	var buf bytes.Buffer

	gzipWriter, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := gzipWriter.Write(f.database); err != nil {
		return nil, err
	}

	if err := gzipWriter.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (snapshot *snapshot) Persist(sink raft.SnapshotSink) error {
	err := func() error {
		b := new(bytes.Buffer)

		// idea taken from rqlite. If the snapshot database is compressed, the size is
		// marked using the largest uint64 number.
		if err := binary.Write(b, binary.LittleEndian, uint64(math.MaxUint64)); err != nil {
			return err
		}

		if _, err := sink.Write(b.Bytes()); err != nil {
			return err
		}
		b.Reset()

		compressedDB, err := snapshot.compressDatabase()
		if err != nil {
			return err
		}

		if compressedDB == nil {
			if err := binary.Write(b, binary.LittleEndian, uint64(0)); err != nil {
				return err
			}

			if _, err := sink.Write(b.Bytes()); err != nil {
				return err
			}
		} else {
			if err := binary.Write(b, binary.LittleEndian, uint64(len(compressedDB))); err != nil {
				return err
			}

			if _, err := sink.Write(b.Bytes()); err != nil {
				return err
			}

			if _, err := sink.Write(compressedDB); err != nil {
				return err
			}
		}

		return sink.Close()
	}()
	if err != nil {
		sink.Cancel()
		return err
	}

	return nil
}

func (snapshot *snapshot) Release() {}

func NewDB(dataDir string, config *Config) (*Consensus, error) {
	c := &Consensus{
		config: config,
	}

	if err := c.setupDB(dataDir); err != nil {
		return nil, err
	}

	if err := c.setupRaft(dataDir); err != nil {
		return nil, err
	}

	var err error
	c.log, err = zap.NewProduction()
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Consensus) setupDB(dataDir string) error {
	dbDir := filepath.Join(dataDir, "db")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return err
	}

	var err error
	c.db, err = engine.Open(filepath.Join(dbDir, "sqlite.db"))
	return err
}

func (c *Consensus) setupRaft(dataDir string) error {
	if err := os.Mkdir(filepath.Join(dataDir, "raft"), os.ModePerm); err != nil {
		return err
	}

	stablepb, err := raftboltdb.NewBoltStore(filepath.Join(dataDir, "raft", "raft.db"))
	if err != nil {
		return err
	}

	snapshotpb, err := raft.NewFileSnapshotStore(filepath.Join(dataDir, "raft"), 1, os.Stderr)
	if err != nil {
		return err
	}

	maxPool := 5
	timeout := 10 * time.Second

	c.tn = raft.NewNetworkTransport(
		c.config.Transport,
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

	c.raft, err = raft.NewRaft(config, c, stablepb, stablepb, snapshotpb, c.tn)
	if err != nil {
		return err
	}

	hasState, err := raft.HasExistingState(stablepb, stablepb, snapshotpb)
	if err != nil {
		return err
	}

	if c.config.Raft.Bootstrap && !hasState {
		config := raft.Configuration{
			Servers: []raft.Server{{
				ID:      config.LocalID,
				Address: raft.ServerAddress(c.config.Transport.Addr().String()),
			}},
		}
		err = c.raft.BootstrapCluster(config).Error()
	}
	return err
}

func (c *Consensus) Apply(record *raft.Log) interface{} {
	defer func() {
		c.idxmu.Lock()
		c.lastIdx = record.Index
		defer c.idxmu.Unlock()
	}()

	return c.applyAction(record.Data, &c.db)
}

func (c *Consensus) WaitForIndex(idx uint64, timeout time.Duration) (uint64, error) {
	tck := time.NewTicker(100 * time.Millisecond)
	defer tck.Stop()
	tmr := time.NewTimer(timeout)
	defer tmr.Stop()

	var idx2 uint64
	for {
		select {
		case <-tck.C:
			c.idxmu.RLock()
			idx2 = c.lastIdx
			c.idxmu.RUnlock()

			if idx2 >= idx {
				return idx2, nil
			}
		case <-tmr.C:
			return 0, fmt.Errorf("timeout expired")
		}
	}
}

func (c *Consensus) applyAction(data []byte, dbPtr **engine.Engine) interface{} {
	var ac pb.Action
	db := *dbPtr

	if err := proto.Unmarshal(data, &ac); err != nil {
		panic("failed to unmarshal cluster command")
	}

	switch ac.Type {
	case pb.Action_ACTION_EXECUTE:
		var execAc pb.Request
		if err := proto.Unmarshal(ac.Body, &execAc); err != nil {
			panic("failed to unmarshal execute request")
		}
		res, err := db.Exec(&execAc)
		return &execResponse{res: res, error: err}
	case pb.Action_ACTION_QUERY:
		var execAc pb.QueryReq
		if err := proto.Unmarshal(ac.Body, &execAc); err != nil {
			panic("failed to unmarshal execute request")
		}
		res, err := db.Query(execAc.Request)
		return &queryResponse{res: res, error: err}
	case pb.Action_ACTION_NO:
		return nil
	}
	return nil
}

func (c *Consensus) apply(ty pb.Action_Type, data proto.Message) (interface{}, error) {
	messageBytes, err := proto.Marshal(data)
	if err != nil {
		return nil, err
	}

	action := &pb.Action{
		Type: ty,
		Body: messageBytes,
	}

	encodedAction, err := proto.Marshal(action)
	if err != nil {
		return nil, err
	}

	timeout := 10 * time.Second
	future := c.raft.Apply(encodedAction, timeout)
	if future.Error() != nil {
		if future.Error() == raft.ErrNotLeader {
			return nil, ErrNotLeader
		}

		return nil, future.Error()
	}

	res := future.Response()
	if err, ok := res.(error); ok {
		return nil, err
	}

	return res, nil
}

func (c *Consensus) Exec(req *pb.Request) ([]*pb.ExecRes, error) {
	if !c.IsLeader() {
		return nil, ErrNotLeader
	}

	res, err := c.apply(pb.Action_ACTION_EXECUTE, req)
	if err != nil {
		return nil, err
	}

	fsmRes := res.(*execResponse)
	return fsmRes.res, fsmRes.error
}

func (c *Consensus) Query(q *pb.QueryReq) ([]*pb.QueryRes, error) {
	if q.StrongConsistency {
		// query from the leader ensuring that entries are up-to-date
		if !c.IsLeader() {
			return nil, ErrNotLeader
		}

		b, err := proto.Marshal(q)
		if err != nil {
			return nil, err
		}

		action := &pb.Action{
			Type: pb.Action_ACTION_QUERY,
			Body: b,
		}

		b, err = proto.Marshal(action)
		if err != nil {
			return nil, err
		}

		future := c.raft.Apply(b, 10*time.Second).(raft.ApplyFuture)
		if future.Error() != nil {
			if future.Error() == raft.ErrNotLeader {
				return nil, ErrNotLeader
			}

			return nil, future.Error()
		}

		res := future.Response().(*queryResponse)
		return res.res, res.error
	}

	if q.Request.Transaction {
		c.txmu.RLock()
		defer c.txmu.RUnlock()
	}

	return c.db.Query(q.Request)
}

func (c *Consensus) Restore(rc io.ReadCloser) error {
	start := time.Now()

	bytes, err := snapshotToBytes(rc)
	if err != nil {
		return fmt.Errorf("repb failed: %s", err)
	}

	if bytes == nil {
		c.log.Info("no database found in snapshot")
	}

	if err := c.db.Close(); err != nil {
		return fmt.Errorf("failed to close pre-repb database: %s", err)
	}

	db, err := createDiskDatabase(bytes, c.dbPath)
	if err != nil {
		return fmt.Errorf("cannot open disk database %s", err)
	}
	c.db = db
	c.log.Info(fmt.Sprintf("node restoration took: %s", time.Since(start)))
	return nil
}

func (c *Consensus) Snapshot() (raft.FSMSnapshot, error) {
	// stop query transactions while this is running
	c.txmu.Lock()
	defer c.txmu.Unlock()

	snap := newSnapshot(c.db)
	c.log.Info(fmt.Sprintf("node snapshot created in %s", time.Since(snap.created)))

	return snap, nil
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

func (c *Consensus) GetServers() ([]*pb.Server, error) {
	future := c.raft.GetConfiguration()
	if err := future.Error(); err != nil {
		return nil, err
	}

	var servers []*pb.Server
	for _, server := range future.Configuration().Servers {
		servers = append(servers, &pb.Server{
			Id:       string(server.ID),
			RpcAddr:  string(server.Address),
			IsLeader: c.raft.Leader() == server.Address,
		})
	}

	return servers, nil
}

func (c *Consensus) Join(id, addr string) error {
	c.log.Info("request to join node", zap.String("id", id), zap.String("addr", addr))

	if !c.IsLeader() {
		return ErrNotLeader
	}

	// convert types
	serverID := raft.ServerID(id)
	serverAddr := raft.ServerAddress(addr)

	if serverID == c.config.Raft.LocalID {
		return ErrJoinSelf
	}

	future := c.raft.GetConfiguration()
	if err := future.Error(); err != nil {
		return err
	}

	for _, srv := range future.Configuration().Servers {
		if srv.ID == serverID || srv.Address == serverAddr {
			if srv.ID == serverID && srv.Address == serverAddr {
				c.log.Info("node already in cluster")
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
		if err == raft.ErrNotLeader {
			return ErrNotLeader
		}

		return err
	}

	c.log.Info("node joined the cluster", zap.String("id", id), zap.String("addr", addr))
	return nil
}

func (c *Consensus) Leave(id string) error {
	if c.raft.State() != raft.Leader {
		return ErrNotLeader
	}

	f := c.raft.RemoveServer(raft.ServerID(id), 0, 0)
	if f.Error() != nil {
		if f.Error() == raft.ErrNotLeader {
			return ErrNotLeader
		}

		return f.Error()
	}

	return nil
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

func snapshotToBytes(rc io.ReadCloser) ([]byte, error) {
	var offset int64
	b, err := ioutil.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("read all error: %s", err)
	}

	var size uint64
	if err := binary.Read(bytes.NewReader(b[offset:offset+8]), binary.LittleEndian, &size); err != nil {
		return nil, fmt.Errorf("binary read error: %s", err)
	}
	offset += 8

	isCompressed := false

	if size == uint64(math.MaxUint64) {
		isCompressed = true

		var actualSize uint64
		err := binary.Read(
			bytes.NewReader(b[offset:offset+8]),
			binary.LittleEndian,
			&actualSize,
		)
		if err != nil {
			return nil, fmt.Errorf("read compressed size: %s", err)
		}
		offset = offset + 8
		size = actualSize
	}

	var res []byte
	if size > 0 {
		if isCompressed {
			buf := new(bytes.Buffer)

			gzipReader, err := gzip.NewReader(bytes.NewReader(b[offset : offset+int64(size)]))
			if err != nil {
				return nil, err
			}

			if _, err := io.Copy(buf, gzipReader); err != nil {
				return nil, fmt.Errorf("sqlite database decompress %s", err)
			}

			if err := gzipReader.Close(); err != nil {
				return nil, err
			}
			res = buf.Bytes()
		} else {
			res = b[offset : offset+int64(size)]
		}
	} else {
		res = nil
	}

	return res, nil
}

func createDiskDatabase(b []byte, path string) (*engine.Engine, error) {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	if b != nil {
		if err := ioutil.WriteFile(path, b, 0660); err != nil {
			return nil, err
		}
	}

	return engine.Open(path)
}

func (c *Consensus) LeaderAddr() string {
	return string(c.raft.Leader())
}

func (c *Consensus) Metrics() (map[string]any, error) {
	databaseSize, err := c.db.Size()
	if err != nil {
		return nil, err
	}

	raftDirSize, err := getDirectorySize(c.raftDir)
	if err != nil {
		return nil, err
	}

	finalStatus := map[string]any{
		"node_id":             c.config.Raft.LocalID,
		"leader":              c.LeaderAddr(),
		"directory":           c.raftDir,
		"raft_directory_size": raftDirSize,
		"database_size":       databaseSize,
	}

	return finalStatus, nil
}

func getDirectorySize(path string) (int64, error) {
	var size int64

	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})

	return size, err
}
