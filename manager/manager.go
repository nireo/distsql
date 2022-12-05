package manager

import (
	"io"
	"net"

	"github.com/hashicorp/serf/serf"
	"go.uber.org/zap"
)

// Config has all the configurable fields for Registry.
type Config struct {
	NodeName       string
	BindAddr       string
	Tags           map[string]string
	StartJoinAddrs []string
}

// Handler represents a interface to a internal handler that also needs information about
// any possible joins or leaves. In actual use the store.Store is passed here because Raft
// needs to know of any possible changes to the cluster.
type Handler interface {
	Join(id, addr string) error
	Leave(id string) error
}

// Manager handles service discovery and keeping track of the nodes currently in the cluster.
type Manager struct {
	Config
	handler Handler
	serf    *serf.Serf
	events  chan serf.Event
	logger  *zap.Logger
}

// New creates a registry instance and sets up serf for service discovery. This function
// starts up the whole registry functionality by running an event handler, connection to
// existing nodes and managing possible joins/leaves.
func New(handler Handler, config Config) (*Manager, error) {
	r := &Manager{
		Config:  config,
		handler: handler,
		logger:  zap.L().Named("registry"),
	}

	if err := r.setupSerf(); err != nil {
		return nil, err
	}

	return r, nil
}

// setupSerf sets up a listener, configuration and event channel for serf to use. It
// also attemps to connect to any nodes listed in StartJoinAddrs. It also starts the
// eventHandler in a goroutine to handle any incoming serf events.
func (r *Manager) setupSerf() error {
	addr, err := net.ResolveTCPAddr("tcp", r.BindAddr)
	if err != nil {
		return err
	}

	config := serf.DefaultConfig()
	config.Init() // allocate subdata structures

	config.LogOutput = io.Discard
	config.MemberlistConfig.BindAddr = addr.IP.String()
	config.MemberlistConfig.BindPort = addr.Port

	r.events = make(chan serf.Event)
	config.EventCh = r.events
	config.Tags = r.Tags
	config.NodeName = r.NodeName

	r.serf, err = serf.Create(config)
	if err != nil {
		return err
	}

	go r.eventHandler()
	if r.StartJoinAddrs != nil {
		if _, err := r.serf.Join(r.StartJoinAddrs, true); err != nil {
			return err
		}
	}

	return nil
}

// eventHandler is run concurrently and it listens for items in the event channel.
// Then events that arrive in the event channel are handled. The only events that
// matter here are serf.EventMemberJoin and serv.EventMemberLeave
func (r *Manager) eventHandler() {
	for e := range r.events {
		switch e.EventType() {
		case serf.EventMemberJoin:
			for _, member := range e.(serf.MemberEvent).Members {
				if r.isLocal(member) {
					continue
				}
				r.handleJoin(member)
			}
		case serf.EventMemberLeave:
			for _, member := range e.(serf.MemberEvent).Members {
				if r.isLocal(member) {
					continue
				}
				r.handleLeave(member)
			}
		}
	}
}

// handleJoin sends information to the internal handler to add given node to the cluster.
func (r *Manager) handleJoin(member serf.Member) {
	if err := r.handler.Join(member.Name, member.Tags["rpc_addr"]); err != nil {
		r.logError(err, "failed to join", member)
	}
}

// handleLeave sends information to the internal handler to remove given node.
func (r *Manager) handleLeave(member serf.Member) {
	if err := r.handler.Leave(member.Name); err != nil {
		r.logError(err, "failed to leave", member)
	}
}

// isLocal checks wheter the given Serf member is the local member
// by checking the members' names
func (r *Manager) isLocal(member serf.Member) bool {
	return r.serf.LocalMember().Name == member.Name
}

// Members returns a point-in-time snapshot of the cluster's members.
func (r *Manager) Members() []serf.Member {
	return r.serf.Members()
}

// Leave tells this member to leave the cluster.
func (r *Manager) Leave() error {
	return r.serf.Leave()
}

func (r *Manager) logError(err error, msg string, member serf.Member) {
	r.logger.Error(
		msg,
		zap.Error(err),
		zap.String("name", member.Name),
		zap.String("rcp_addr", member.Tags["rcp_addr"]),
	)
}
