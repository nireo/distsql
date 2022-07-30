package manager

import (
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
)

type NodeConnection struct {
	ln     net.Listener
	header byte
	addr   net.Addr
}

type listener struct {
	c chan net.Conn
}

func (ln *listener) Accept() (net.Conn, error) {
	conn, ok := <-ln.c
	if !ok {
		return nil, errors.New("network connection failed")
	}
	return conn, nil
}

func (ln *listener) Close() error {
	return nil
}

func (ln *listener) Addr() net.Addr {
	return nil
}

func (c *NodeConnection) Dial(addr string, timeout time.Duration) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: timeout}

	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	if _, err = conn.Write([]byte{c.header}); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

func (c *NodeConnection) Accept() (net.Conn, error) {
	return c.ln.Accept()
}

func (c *NodeConnection) Close() error {
	return c.ln.Close()
}

func (c *NodeConnection) Addr() net.Addr {
	return c.addr
}

type Multiplexer struct {
	ln        net.Listener
	listeners map[byte]*listener
	addr      net.Addr
	wg        sync.WaitGroup
	Timeout   time.Duration
	logger    *zap.Logger
}

func NewMultiplexer(ln net.Listener, nAddr net.Addr) (*Multiplexer, error) {
	var addr net.Addr
	if nAddr == nil {
		addr = ln.Addr()
	}

	return &Multiplexer{
		ln:        ln,
		addr:      addr,
		listeners: make(map[byte]*listener),
		Timeout:   30 * time.Second,
		logger:    zap.L().Named("mux"),
	}, nil
}

func (mx *Multiplexer) Serve() error {
	mx.logger.Info("serving mux",
		zap.String("addr", mx.ln.Addr().String()),
		zap.String("adver", mx.addr.String()),
	)

	for {
		conn, err := mx.ln.Accept()
		if err, ok := err.(interface {
			Temporary() bool
		}); ok && err.Temporary() {
			continue
		}

		if err != nil {
			mx.wg.Wait()
			for _, ln := range mx.listeners {
				close(ln.c)
			}
			return err
		}

		mx.wg.Add(1)
		go mx.handleConn(conn)
	}
}

func (mx *Multiplexer) handleConn(conn net.Conn) {
	defer mx.wg.Done()

	if err := conn.SetReadDeadline(time.Now().Add(mx.Timeout)); err != nil {
		conn.Close()
		mx.logger.Error("cannot set read deadline", zap.Error(err))
		return
	}

	var typ [1]byte
	if _, err := io.ReadFull(conn, typ[:]); err != nil {
		conn.Close()
		mx.logger.Error("cannot read header type", zap.Error(err))
		return
	}

	if err := conn.SetReadDeadline(time.Time{}); err != nil {
		conn.Close()
		mx.logger.Error("cannot reset set read deadline", zap.Error(err))
		return
	}

	handler := mx.listeners[typ[0]]
	if handler == nil {
		conn.Close()
		mx.logger.Error("handler not registered", zap.Int("handler", int(typ[0])))
		return
	}

	handler.c <- conn
}

func (mx *Multiplexer) Listen(header byte) *NodeConnection {
	if _, ok := mx.listeners[header]; ok {
		panic("listener already registered under header byte")
	}

	ln := &listener{
		c: make(chan net.Conn),
	}
	mx.listeners[header] = ln

	conn := &NodeConnection{
		ln:     ln,
		header: header,
		addr:   mx.addr,
	}

	return conn
}
