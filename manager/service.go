package manager

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

const (
	MxRaftHeader    = 1
	MxClusterHeader = 2
)

type Transport interface {
	net.Listener
	Dial(address string, timeout time.Duration) (net.Conn, error)
}

type Service struct {
	transport Transport
	addr      net.Addr
	timeout   time.Duration
	mu        sync.RWMutex
	apiAddr   string
	logger    *zap.Logger
}

func NewService(transport Transport) *Service {
	return &Service{
		transport: transport,
		addr:      transport.Addr(),
		timeout:   10 * time.Second,
		logger:    zap.L().Named("manager"),
	}
}

func (s *Service) Addr() string {
	return s.addr.String()
}

func (s *Service) SetAPIAddr(addr string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.apiAddr = addr
}

func (s *Service) GetAPIAddr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.apiAddr
}

func (s *Service) Close() error {
	return s.transport.Close()
}

func (s *Service) serve() error {
	for {
		conn, err := s.transport.Accept()
		if err != nil {
			return err
		}

		s.handleConn(conn)
	}
}

func (s *Service) handleConn(conn net.Conn) {
	defer conn.Close()
	var err error

	b := make([]byte, 4)
	if _, err = io.ReadFull(conn, b); err != nil {
		return
	}

	sz := binary.LittleEndian.Uint16(b[0:])
	b = make([]byte, sz)

	if _, err = io.ReadFull(conn, b); err != nil {
		return
	}

	c := &Action{}
	if err = proto.Unmarshal(b, c); err != nil {
		conn.Close()
		return
	}

	switch c.Type {
	case Action_ACTION_GET_NODE_URL:
		s.mu.RLock()
		defer s.mu.RUnlock()

		a := &Addr{}
		a.Url = fmt.Sprintf("%s://%s", "http", s.apiAddr)

		b, err := proto.Marshal(a)
		if err != nil {
			conn.Close()
			return
		}
		conn.Write(b)
	}
}

func (s *Service) Open() error {
	go s.serve()
	s.logger.Info("service listening", zap.String("addr", s.transport.Addr().String()))

	return nil
}

func (s *Service) GetNodeAPIAddr(nodeAddr string) (string, error) {
	conn, err := s.transport.Dial(nodeAddr, s.timeout)
	if err != nil {
		return "", fmt.Errorf("dial connection: %s", err)
	}
	defer conn.Close()

	c := &Action{
		Type: Action_ACTION_GET_NODE_URL,
	}

	p, err := proto.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("command marshal: %s", err)
	}

	b := make([]byte, 4)
	binary.LittleEndian.PutUint16(b[0:], uint16(len(p)))
	_, err = conn.Write(b)
	if err != nil {
		return "", fmt.Errorf("write protobuf length: %s", err)
	}
	_, err = conn.Write(p)
	if err != nil {
		return "", fmt.Errorf("write protobuf: %s", err)
	}

	b, err = ioutil.ReadAll(conn)
	if err != nil {
		return "", fmt.Errorf("read protobuf bytes: %s", err)
	}

	a := &Addr{}
	err = proto.Unmarshal(b, a)
	if err != nil {
		return "", fmt.Errorf("protobuf unmarshal: %s", err)
	}

	return a.Url, nil
}
