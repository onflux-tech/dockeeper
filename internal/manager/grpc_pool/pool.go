package grpc_pool

import (
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

const (
	// AgentPort is the port agents listen on for manager commands.
	AgentPort = "50052"
)

// Pool manages a set of reusable gRPC client connections to agents.
type Pool struct {
	mu    sync.RWMutex
	conns map[string]*grpc.ClientConn
}

// New creates a new connection pool.
func New() *Pool {
	return &Pool{
		conns: make(map[string]*grpc.ClientConn),
	}
}

// Get returns an existing healthy connection to the given agent address or
// creates a new one. The address should be the agent IP (without port).
func (p *Pool) Get(agentIP string) (*grpc.ClientConn, error) {
	addr := net.JoinHostPort(agentIP, AgentPort)

	// Fast path: read lock
	p.mu.RLock()
	conn, ok := p.conns[addr]
	p.mu.RUnlock()

	if ok && isHealthy(conn) {
		return conn, nil
	}

	// Slow path: write lock
	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if conn, ok := p.conns[addr]; ok && isHealthy(conn) {
		return conn, nil
	}

	// Remove stale connection
	if old, exists := p.conns[addr]; exists {
		old.Close()
		delete(p.conns, addr)
	}

	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("dial agent %s: %w", addr, err)
	}

	p.conns[addr] = conn
	slog.Debug("gRPC pool: new connection", "addr", addr)
	return conn, nil
}

// Close gracefully closes all pooled connections.
func (p *Pool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for addr, conn := range p.conns {
		if err := conn.Close(); err != nil {
			slog.Warn("gRPC pool: close error", "addr", addr, "error", err)
		}
	}
	p.conns = make(map[string]*grpc.ClientConn)
	slog.Info("gRPC pool closed")
}

// isHealthy returns true if the connection is usable.
func isHealthy(conn *grpc.ClientConn) bool {
	state := conn.GetState()
	return state != connectivity.Shutdown && state != connectivity.TransientFailure
}
