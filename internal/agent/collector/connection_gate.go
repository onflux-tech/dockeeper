package collector

import (
	"math/rand/v2"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

const (
	gateBackoffBase = 30 * time.Second
	gateBackoffMax  = 5 * time.Minute
	gateBackoffMul  = 2.0
	gateJitterFrac  = 0.25
)

type grpcConnState interface {
	GetState() connectivity.State
	Connect()
}

type ConnectionGate struct {
	conn    grpcConnState
	mu      sync.Mutex
	attempt int
	retryAt time.Time
}

func NewConnectionGate(conn *grpc.ClientConn) *ConnectionGate {
	return &ConnectionGate{conn: conn}
}

func (g *ConnectionGate) Allow() bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if time.Now().Before(g.retryAt) {
		return false
	}
	if g.attempt == 0 || g.conn.GetState() == connectivity.Ready {
		return true
	}
	g.conn.Connect()
	return false
}

func (g *ConnectionGate) Succeeded() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.attempt = 0
	g.retryAt = time.Time{}
}

func (g *ConnectionGate) Failed() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.attempt++
	g.retryAt = time.Now().Add(gateBackoff(g.attempt))
}

func gateBackoff(attempt int) time.Duration {
	d := float64(gateBackoffBase)
	for range attempt - 1 {
		d *= gateBackoffMul
		if d > float64(gateBackoffMax) {
			d = float64(gateBackoffMax)
			break
		}
	}
	jitter := d * gateJitterFrac * (2*rand.Float64() - 1)
	d += jitter
	if d > float64(gateBackoffMax) {
		d = float64(gateBackoffMax)
	}
	return time.Duration(d)
}
