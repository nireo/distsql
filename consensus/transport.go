package consensus

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/hashicorp/raft"
)

// Transport handles communications between different raft nodes.
type Transport struct {
	ln        net.Listener
	servertls *tls.Config
	peertls   *tls.Config
}

// NewTransport creates a new transport instance.
func NewTransport(ln net.Listener) *Transport {
	return &Transport{
		ln: ln,
	}
}

// NewTLSTransport creates a transport instance with the tls fields populated.
func NewTLSTransport(ln net.Listener, servertls *tls.Config, peertls *tls.Config) *Transport {
	return &Transport{
		ln:        ln,
		servertls: servertls,
		peertls:   peertls,
	}
}

// Dial creates a connection to a given address. This function appends the RaftRPC identifier
// (1) to the request's beginning such that raft requests can be properly identified.
func (tn *Transport) Dial(addr raft.ServerAddress, timeout time.Duration) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: timeout}

	conn, err := dialer.Dial("tcp", string(addr))
	if err != nil {
		return nil, err
	}

	if _, err = conn.Write([]byte{byte(1)}); err != nil {
		return nil, err
	}

	if tn.peertls != nil {
		return tls.Client(conn, tn.peertls), nil
	}

	return conn, nil
}

// Accept acceps a given dial and checks that the RaftRPC identifier is defined
// at the start; if not then just return an error.
func (tn *Transport) Accept() (net.Conn, error) {
	conn, err := tn.ln.Accept()
	if err != nil {
		return nil, err
	}

	b := make([]byte, 1)
	if _, err = conn.Read(b); err != nil {
		return nil, err
	}

	if b[0] != 1 {
		return nil, fmt.Errorf("not raft rpc connection")
	}

	if tn.servertls != nil {
		return tls.Server(conn, tn.servertls), nil
	}

	return conn, nil
}

// Close closes the listener
func (tn *Transport) Close() error {
	return tn.ln.Close()
}

// Addr returns a net.Addr representing the address Transport is listening on.
func (tn *Transport) Addr() net.Addr {
	return tn.ln.Addr()
}
