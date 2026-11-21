package collector

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/connectivity"
)

func TestCollectContainersSkipsDockerWhenDisconnected(t *testing.T) {
	gate := &ConnectionGate{conn: &fakeGRPCConn{state: connectivity.TransientFailure}}
	gate.Failed()

	orch := &fakeOrchestrator{stats: []ContainerMetrics{{ContainerID: "c1"}}}
	client := &fakeMetricsClient{stream: &fakeContainerMetricsStream{}}
	m := NewMetricsCollector(orch, client, "srv1", gate, discardLogger())

	m.collectContainers(context.Background())

	assert.Equal(t, 0, orch.listStats)
	assert.Equal(t, 0, client.opens)
}

func TestCollectContainersOpensStreamBeforeStats(t *testing.T) {
	gate := &ConnectionGate{conn: &fakeGRPCConn{state: connectivity.Ready}}
	client := &fakeMetricsClient{stream: &fakeContainerMetricsStream{}}
	orch := &fakeOrchestrator{stats: []ContainerMetrics{{ContainerID: "c1"}}}
	orch.onStats = func() {
		assert.Equal(t, 1, client.opens)
	}
	m := NewMetricsCollector(orch, client, "srv1", gate, discardLogger())

	m.collectContainers(context.Background())

	assert.Equal(t, 1, orch.listStats)
	assert.Len(t, client.stream.sent, 1)
	assert.Equal(t, 0, gate.attempt)
}

func TestCollectContainersSkipsStatsWhenStreamFails(t *testing.T) {
	gate := &ConnectionGate{conn: &fakeGRPCConn{state: connectivity.Ready}}
	client := &fakeMetricsClient{openErr: errors.New("manager unavailable")}
	orch := &fakeOrchestrator{stats: []ContainerMetrics{{ContainerID: "c1"}}}
	m := NewMetricsCollector(orch, client, "srv1", gate, discardLogger())

	m.collectContainers(context.Background())

	assert.Equal(t, 1, client.opens)
	assert.Equal(t, 0, orch.listStats)
	assert.Equal(t, 1, gate.attempt)
}

func TestCollectContainersDoesNotFailGateOnDockerError(t *testing.T) {
	gate := &ConnectionGate{conn: &fakeGRPCConn{state: connectivity.Ready}}
	client := &fakeMetricsClient{stream: &fakeContainerMetricsStream{}}
	orch := &fakeOrchestrator{statsErr: errors.New("docker unavailable")}
	m := NewMetricsCollector(orch, client, "srv1", gate, discardLogger())

	m.collectContainers(context.Background())

	assert.Equal(t, 1, orch.listStats)
	assert.Equal(t, 1, client.opens)
	assert.Empty(t, client.stream.sent)
	assert.Equal(t, 0, gate.attempt)
}

func TestCollectContainersResetsGateOnSuccess(t *testing.T) {
	gate := &ConnectionGate{conn: &fakeGRPCConn{state: connectivity.Ready}}
	gate.Failed()

	orch := &fakeOrchestrator{stats: []ContainerMetrics{{ContainerID: "c1"}}}
	client := &fakeMetricsClient{stream: &fakeContainerMetricsStream{}}
	m := NewMetricsCollector(orch, client, "srv1", gate, discardLogger())

	m.collectContainers(context.Background())

	assert.Equal(t, 0, orch.listStats)
	assert.Equal(t, 0, client.opens)

	gate.retryAt = gate.retryAt.Add(-2 * gateBackoffMax)
	m.collectContainers(context.Background())

	assert.Equal(t, 1, orch.listStats)
	assert.Equal(t, 0, gate.attempt)
}
