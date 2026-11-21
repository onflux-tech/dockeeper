package collector

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/connectivity"

	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

func newIncrementalDiscovery(orch *fakeOrchestrator, client *fakeDiscoveryClient) *DiscoveryCollector {
	gate := &ConnectionGate{conn: &fakeGRPCConn{state: connectivity.Ready}}
	return NewDiscoveryCollector(orch, client, "srv1", gate, discardLogger())
}

func seedDiscoveryContainers(d *DiscoveryCollector, orch *fakeOrchestrator, client *fakeDiscoveryClient, containers []*agentpb.ContainerInfo) {
	orch.containers = containers
	d.collect(context.Background())
	orch.listContainers = 0
	orch.inspectCalls = nil
	client.containers = 0
	client.lastContainers = nil
}

func TestDiscoveryFullSyncOnFirstCollect(t *testing.T) {
	orch := &fakeOrchestrator{containers: []*agentpb.ContainerInfo{
		{ContainerId: "c1"},
		{ContainerId: "c2"},
	}}
	client := &fakeDiscoveryClient{}
	d := newIncrementalDiscovery(orch, client)

	d.collect(context.Background())

	assert.Equal(t, 1, orch.listContainers)
	assert.Empty(t, orch.inspectCalls)
	require.NotNil(t, client.lastContainers)
	assert.Len(t, client.lastContainers.GetContainers(), 2)
}

func TestDiscoveryIncrementalInspectsOnlyChangedContainers(t *testing.T) {
	orch := &fakeOrchestrator{containers: []*agentpb.ContainerInfo{
		{ContainerId: "c1"},
		{ContainerId: "c2", Image: "old"},
	}}
	client := &fakeDiscoveryClient{}
	d := newIncrementalDiscovery(orch, client)
	seedDiscoveryContainers(d, orch, client, orch.containers)

	orch.inspectResult = &agentpb.ContainerInfo{ContainerId: "c2", Image: "updated"}
	d.handleContainerEvent(ContainerEvent{ContainerID: "c2", Action: "start"})
	d.collect(context.Background())

	assert.Equal(t, 0, orch.listContainers)
	assert.Equal(t, []string{"c2"}, orch.inspectCalls)
	require.NotNil(t, client.lastContainers)
	require.Len(t, client.lastContainers.GetContainers(), 2)
	assert.Equal(t, "updated", client.lastContainers.GetContainers()[1].GetImage())
}

func TestDiscoveryIncrementalWithoutDirtyDoesNotInspect(t *testing.T) {
	orch := &fakeOrchestrator{containers: []*agentpb.ContainerInfo{
		{ContainerId: "c1"},
		{ContainerId: "c2"},
	}}
	client := &fakeDiscoveryClient{}
	d := newIncrementalDiscovery(orch, client)
	seedDiscoveryContainers(d, orch, client, orch.containers)

	d.collect(context.Background())

	assert.Equal(t, 0, orch.listContainers)
	assert.Empty(t, orch.inspectCalls)
	assert.Equal(t, 0, client.containers)
}

func TestDiscoveryDestroyRemovesContainer(t *testing.T) {
	orch := &fakeOrchestrator{containers: []*agentpb.ContainerInfo{
		{ContainerId: "c1"},
		{ContainerId: "c2"},
	}}
	client := &fakeDiscoveryClient{}
	d := newIncrementalDiscovery(orch, client)
	seedDiscoveryContainers(d, orch, client, orch.containers)

	d.handleContainerEvent(ContainerEvent{ContainerID: "c1", Action: "destroy"})
	d.collect(context.Background())

	assert.Equal(t, 0, orch.listContainers)
	assert.Empty(t, orch.inspectCalls)
	require.NotNil(t, client.lastContainers)
	require.Len(t, client.lastContainers.GetContainers(), 1)
	assert.Equal(t, "c2", client.lastContainers.GetContainers()[0].GetContainerId())
}

func TestDiscoveryFullSyncAfterInterval(t *testing.T) {
	orch := &fakeOrchestrator{containers: []*agentpb.ContainerInfo{{ContainerId: "c1"}}}
	client := &fakeDiscoveryClient{}
	d := newIncrementalDiscovery(orch, client)
	seedDiscoveryContainers(d, orch, client, orch.containers)

	d.lastFullSync = time.Now().Add(-discoveryFullSyncInterval - time.Second)
	d.collect(context.Background())

	assert.Equal(t, 1, orch.listContainers)
	assert.Empty(t, orch.inspectCalls)
}

func TestDiscoveryKeepsCacheWhenDisconnected(t *testing.T) {
	orch := &fakeOrchestrator{containers: []*agentpb.ContainerInfo{
		{ContainerId: "c1"},
		{ContainerId: "c2"},
	}}
	client := &fakeDiscoveryClient{}
	d := newIncrementalDiscovery(orch, client)
	seedDiscoveryContainers(d, orch, client, orch.containers)

	d.gate.Failed()
	d.collect(context.Background())

	assert.Equal(t, 0, orch.listContainers)
	assert.Empty(t, orch.inspectCalls)
	assert.Len(t, d.containerSnapshot(), 2)

	d.gate.retryAt = time.Now().Add(-time.Second)
	d.collect(context.Background())

	assert.Equal(t, 0, orch.listContainers)
	assert.Len(t, d.containerSnapshot(), 2)
}

func TestDiscoveryConsumesContainerEvents(t *testing.T) {
	orch := &fakeOrchestrator{
		events:    make(chan ContainerEvent, 1),
		eventsErr: make(chan error),
	}
	client := &fakeDiscoveryClient{}
	d := newIncrementalDiscovery(orch, client)

	orch.events <- ContainerEvent{ContainerID: "c2", Action: "update"}
	close(orch.events)

	assert.NoError(t, d.consumeContainerEvents(context.Background()))

	d.mu.Lock()
	_, dirty := d.dirty["c2"]
	d.mu.Unlock()
	assert.True(t, dirty)
}

func TestDiscoveryMatchesContainerIDsFromEvents(t *testing.T) {
	shortID := "0123456789ab"
	fullID := shortID + strings.Repeat("0", 52)

	orch := &fakeOrchestrator{containers: []*agentpb.ContainerInfo{{ContainerId: shortID}}}
	client := &fakeDiscoveryClient{}
	d := newIncrementalDiscovery(orch, client)
	seedDiscoveryContainers(d, orch, client, orch.containers)

	orch.inspectResult = &agentpb.ContainerInfo{ContainerId: shortID, Image: "updated"}
	d.handleContainerEvent(ContainerEvent{ContainerID: fullID, Action: "update"})
	d.collect(context.Background())

	assert.Equal(t, []string{shortID}, orch.inspectCalls)
	require.NotNil(t, client.lastContainers)
	require.Len(t, client.lastContainers.GetContainers(), 1)
	assert.Equal(t, "updated", client.lastContainers.GetContainers()[0].GetImage())
}

func TestDiscoveryRetriesFailedInspect(t *testing.T) {
	orch := &fakeOrchestrator{containers: []*agentpb.ContainerInfo{
		{ContainerId: "c1"},
		{ContainerId: "c2", Image: "old"},
	}}
	client := &fakeDiscoveryClient{}
	d := newIncrementalDiscovery(orch, client)
	seedDiscoveryContainers(d, orch, client, orch.containers)

	orch.inspectErr = errors.New("docker unavailable")
	d.handleContainerEvent(ContainerEvent{ContainerID: "c2", Action: "start"})
	d.collect(context.Background())
	assert.Equal(t, []string{"c2"}, orch.inspectCalls)

	orch.inspectErr = nil
	orch.inspectResult = &agentpb.ContainerInfo{ContainerId: "c2", Image: "updated"}
	d.collect(context.Background())

	assert.Equal(t, []string{"c2", "c2"}, orch.inspectCalls)
	require.NotNil(t, client.lastContainers)
	assert.Equal(t, "updated", client.lastContainers.GetContainers()[1].GetImage())
}

func TestDiscoveryWatcherStopsOnCancel(t *testing.T) {
	orch := &fakeOrchestrator{
		events:    make(chan ContainerEvent),
		eventsErr: make(chan error),
	}
	client := &fakeDiscoveryClient{}
	d := newIncrementalDiscovery(orch, client)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		d.watchContainerEvents(ctx)
		close(done)
	}()
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("watchContainerEvents did not stop after cancel")
	}
}
