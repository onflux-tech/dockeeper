package collector

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/connectivity"
)

type fakeGRPCConn struct {
	state    connectivity.State
	connects int
}

func (c *fakeGRPCConn) GetState() connectivity.State { return c.state }
func (c *fakeGRPCConn) Connect()                     { c.connects++ }

func TestConnectionGateAllowsFirstCall(t *testing.T) {
	gate := &ConnectionGate{conn: &fakeGRPCConn{state: connectivity.Idle}}

	assert.True(t, gate.Allow())
}

func TestConnectionGateBlocksDuringBackoff(t *testing.T) {
	gate := &ConnectionGate{conn: &fakeGRPCConn{state: connectivity.TransientFailure}}

	gate.Failed()

	assert.False(t, gate.Allow())
}

func TestConnectionGateBacksOffExponentially(t *testing.T) {
	gate := &ConnectionGate{conn: &fakeGRPCConn{state: connectivity.TransientFailure}}

	gate.Failed()
	first := gate.retryAt
	gate.Failed()

	assert.True(t, gate.retryAt.After(first))
}

func TestConnectionGateBackoffCap(t *testing.T) {
	for attempt := 1; attempt <= 40; attempt++ {
		assert.LessOrEqual(t, gateBackoff(attempt), gateBackoffMax)
	}
}

func TestConnectionGateResetsOnSuccess(t *testing.T) {
	gate := &ConnectionGate{conn: &fakeGRPCConn{state: connectivity.TransientFailure}}

	gate.Failed()
	gate.Succeeded()

	assert.Equal(t, 0, gate.attempt)
	assert.True(t, gate.Allow())
}

func TestConnectionGateAllowsWhenReadyAfterBackoff(t *testing.T) {
	gate := &ConnectionGate{conn: &fakeGRPCConn{state: connectivity.Ready}}
	gate.Failed()
	gate.retryAt = time.Now().Add(-time.Millisecond)

	assert.True(t, gate.Allow())
}

func TestConnectionGateNudgesConnectWhenNotReady(t *testing.T) {
	conn := &fakeGRPCConn{state: connectivity.TransientFailure}
	gate := &ConnectionGate{conn: conn}
	gate.Failed()
	gate.retryAt = time.Now().Add(-time.Millisecond)

	assert.False(t, gate.Allow())
	assert.Equal(t, 1, conn.connects)
}
