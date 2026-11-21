package collector

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/connectivity"

	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

func TestTraefikSkipsDockerWhenDisconnected(t *testing.T) {
	gate := &ConnectionGate{conn: &fakeGRPCConn{state: connectivity.TransientFailure}}
	gate.Failed()

	orch := &fakeOrchestrator{}
	client := &fakeTraefikClient{}
	tc := NewTraefikCollector(orch, client, "srv1", gate, discardLogger())

	tc.detectTraefik(context.Background())
	assert.Equal(t, 0, orch.listContainers)

	tc.collect(context.Background())
	assert.Equal(t, 0, orch.listContainers)
	assert.Equal(t, 0, client.reports)
}

func TestTraefikSkipsRoutesWhenDisconnected(t *testing.T) {
	gate := &ConnectionGate{conn: &fakeGRPCConn{state: connectivity.TransientFailure}}
	gate.Failed()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("[]"))
	}))
	defer srv.Close()

	orch := &fakeOrchestrator{}
	client := &fakeTraefikClient{}
	tc := NewTraefikCollector(orch, client, "srv1", gate, discardLogger())
	tc.detected = true
	tc.checkedAt = time.Now()
	tc.apiURL = srv.URL

	tc.collect(context.Background())

	assert.Equal(t, 0, orch.listContainers)
	assert.Equal(t, 0, client.reports)
}

func TestTraefikDetectsOnFirstCycle(t *testing.T) {
	gate := &ConnectionGate{conn: &fakeGRPCConn{state: connectivity.Idle}}

	orch := &fakeOrchestrator{containers: []*agentpb.ContainerInfo{
		{
			ContainerId: "c1",
			Image:       "traefik:v3",
			Name:        "traefik",
			PortsJson:   `[{"container_port":8080,"host_port":8088}]`,
		},
	}}
	client := &fakeTraefikClient{}
	tc := NewTraefikCollector(orch, client, "srv1", gate, discardLogger())

	tc.detectTraefik(context.Background())

	assert.True(t, tc.detected)
	assert.Equal(t, 1, orch.listContainers)
}
