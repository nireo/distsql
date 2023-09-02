package coordinator

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/hashicorp/raft"
	"github.com/nireo/distsql/consensus"
	"github.com/nireo/distsql/manager"
	"github.com/nireo/distsql/service"
	"github.com/soheilhy/cmux"
)

// The coordinator hosts services on two ports. The configured BindAddr and the port
// of that address will be used for serf communication while the CommunicationPort will
// be used for raft and http communications. Raft connections are matched by a identifying
// byte at the start of requests and every other connection not matched with this byte
// will be assumed to be a HTTP connection.

// Config handles all of the customizable values for Service.
type Config struct {
	DataDir           string   // where to store raft data.
	BindAddr          string   // serf addr.
	CommunicationPort int      // port for raft and client connections
	StartJoinAddrs    []string // addresses to join to
	Bootstrap         bool     // should bootstrap cluster?
	NodeName          string   // raft server id
	LeaderStartAddr   string   // mostly for testing

	ServerTLS *tls.Config
	PeerTLS   *tls.Config
}

// HTTPAddr returns the HTTP address of the service's http handler.
func (c *Config) HTTPAddr() (string, error) {
	host, _, err := net.SplitHostPort(c.BindAddr)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("http://%s:%d", host, c.CommunicationPort), nil
}

// RaftAddr returns the raft connection address
func (c *Config) RaftAddr() (string, error) {
	host, _, err := net.SplitHostPort(c.BindAddr)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%d", host, c.CommunicationPort), nil
}

// Coordinator handles connecting all of the independent components of the system.
// Mainly connecting the HTTP service and raft store is the main its main job.
type Coordinator struct {
	Config       Config
	mux          cmux.CMux
	manager      *manager.Manager
	raft         *consensus.Consensus
	shutdown     bool
	shutdowns    chan struct{}
	shutdownlock sync.Mutex
}

// New sets up all of the fields in the Coordinator and makes the service ready for
// running.
func New(conf Config) (*Coordinator, error) {
	c := &Coordinator{
		Config:    conf,
		shutdowns: make(chan struct{}),
	}

	// run setups
	if err := c.setupMux(); err != nil {
		return nil, err
	}

	setupFns := []func() error{
		c.setupRaft,
		c.setupHTTP,
		c.setupManager,
	}

	for _, fn := range setupFns {
		if err := fn(); err != nil {
			return nil, err
		}
	}

	if err := c.joinToLeader(); err != nil {
		return nil, err
	}

	// start connection multiplexer
	go c.serve()

	return c, nil
}

// setupMux sets up the terminal multiplexer that handles all of the communication
// between nodes.
func (c *Coordinator) setupMux() error {
	host, _, err := net.SplitHostPort(c.Config.BindAddr)
	if err != nil {
		return err
	}

	// create the listener
	addr := fmt.Sprintf("%s:%d", host, c.Config.CommunicationPort)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	c.mux = cmux.New(l)
	return nil
}

// setupRaft sets up a filtered listener to the connection multiplexer and starts
// raft.
func (c *Coordinator) setupRaft() error {
	// create filtered listener
	raftListener := c.mux.Match(func(reader io.Reader) bool {
		b := make([]byte, 1)
		if _, err := reader.Read(b); err != nil {
			return false
		}
		return b[0] == 1
	})

	// create config
	conf := &consensus.Config{}
	conf.Transport = consensus.NewTLSTransport(
		raftListener,
		c.Config.ServerTLS,
		c.Config.PeerTLS,
	)
	conf.Raft.LocalID = raft.ServerID(c.Config.NodeName)
	conf.Raft.Bootstrap = c.Config.Bootstrap

	// create raft instance and bootstrap the cluster if neccesary
	var err error
	c.raft, err = consensus.NewDB(c.Config.DataDir, conf)
	if err != nil {
		return err
	}

	if c.Config.Bootstrap {
		err = c.raft.WaitForLeader(3 * time.Second)
	}

	return err
}

// Close shuts down the  components and leaves the registry cluster.
func (c *Coordinator) Close() error {
	c.shutdownlock.Lock()
	defer c.shutdownlock.Unlock()

	// check that the service hasn't already shutdown
	if c.shutdown {
		return nil
	}
	c.shutdown = true
	close(c.shutdowns)

	// stop running different submodules.
	closeFns := []func() error{
		c.manager.Leave,
		c.raft.Close,
	}

	for _, fn := range closeFns {
		if err := fn(); err != nil {
			return err
		}
	}

	return nil
}

// serve starts the connection multiplexer
func (c *Coordinator) serve() error {
	if err := c.mux.Serve(); err != nil {
		// don't leave serf and other things running.
		c.Close()
		return err
	}

	return nil
}

func (c *Coordinator) joinToLeader() error {
	// don't connect to anything this is fine.
	if c.Config.LeaderStartAddr == "" {
		return nil
	}

	addr, err := c.Config.RaftAddr()
	if err != nil {
		return err
	}

	body := map[string]string{
		"id":   c.Config.NodeName,
		"addr": addr,
	}

	reqBytes, err := json.Marshal(body)
	if err != nil {
		return err
	}

	fmt.Println("GOT ADDR", c.Config.LeaderStartAddr)
	r, err := http.NewRequest("POST", c.Config.LeaderStartAddr+"/join", bytes.NewBuffer(reqBytes))
	if err != nil {
		return err
	}
	client := &http.Client{}
	res, err := client.Do(r)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	bod, _ := io.ReadAll(res.Body)
	fmt.Println(string(bod))
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to join leader cluster got code: %d", res.StatusCode)
	}
	return nil
}

// setupManager sets up the service discovery and management module.
func (c *Coordinator) setupManager() error {
	addr, err := c.Config.HTTPAddr()
	if err != nil {
		return err
	}

	c.manager, err = manager.New(c.raft, manager.Config{
		NodeName: c.Config.NodeName,
		BindAddr: c.Config.BindAddr,
		Tags: map[string]string{
			"addr": addr,
		},
		StartJoinAddrs: c.Config.StartJoinAddrs,
	})

	return err
}

// setupHTTP starts up the http server
func (c *Coordinator) setupHTTP() error {
	httpaddr, err := c.Config.HTTPAddr()
	if err != nil {
		return err
	}

	conf := service.Config{
		EnablePPROF: true,
	}

	httpserv, err := service.NewService(httpaddr, c.raft, conf)
	if err != nil {
		return err
	}

	// match any connection not matched with the raft byte to be a HTTP
	// connection.
	ln := c.mux.Match(cmux.Any())
	if err = httpserv.StartWithListener(ln); err != nil {
		return err
	}

	return nil
}
