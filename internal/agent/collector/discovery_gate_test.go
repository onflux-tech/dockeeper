package collector

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/connectivity"
)

func TestDiscoverySkipsDockerWhenDisconnected(t *testing.T) {
	gate := &ConnectionGate{conn: &fakeGRPCConn{state: connectivity.TransientFailure}}
	gate.Failed()

	orch := &fakeOrchestrator{}
	client := &fakeDiscoveryClient{}
	d := NewDiscoveryCollector(orch, client, "srv1", gate, discardLogger())

	d.collect(context.Background())

	assert.Equal(t, 0, orch.listContainers)
	assert.Equal(t, 0, orch.listImages)
	assert.Equal(t, 0, orch.listNetworks)
	assert.Equal(t, 0, orch.listVolumes)
	assert.Equal(t, 0, client.containers)
	assert.Equal(t, 0, client.images)
	assert.Equal(t, 0, client.networks)
	assert.Equal(t, 0, client.volumes)
}

func TestDiscoveryCollectsOnFirstCycle(t *testing.T) {
	gate := &ConnectionGate{conn: &fakeGRPCConn{state: connectivity.Idle}}

	orch := &fakeOrchestrator{}
	client := &fakeDiscoveryClient{}
	d := NewDiscoveryCollector(orch, client, "srv1", gate, discardLogger())

	d.collect(context.Background())

	assert.Equal(t, 1, orch.listContainers)
	assert.Equal(t, 1, orch.listImages)
	assert.Equal(t, 1, orch.listNetworks)
	assert.Equal(t, 1, orch.listVolumes)
	assert.Equal(t, 1, client.containers)
	assert.Equal(t, 1, client.images)
	assert.Equal(t, 1, client.networks)
	assert.Equal(t, 1, client.volumes)
	assert.Equal(t, 0, gate.attempt)
}
