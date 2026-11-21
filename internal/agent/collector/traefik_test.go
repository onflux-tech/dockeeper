package collector

import (
	"context"
	"errors"
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
	assert.Equal(t, 0, orch.listSummaries)
	assert.Equal(t, 0, orch.listContainers)
	assert.Empty(t, orch.inspectCalls)

	tc.collect(context.Background())
	assert.Equal(t, 0, orch.listSummaries)
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

	assert.Equal(t, 0, orch.listSummaries)
	assert.Equal(t, 0, orch.listContainers)
	assert.Equal(t, 0, client.reports)
}

func TestTraefikDetectsOnFirstCycle(t *testing.T) {
	gate := &ConnectionGate{conn: &fakeGRPCConn{state: connectivity.Idle}}

	orch := &fakeOrchestrator{summaries: []*agentpb.ContainerInfo{
		{
			ContainerId: "traefik-id",
			Image:       "traefik:v3",
			Name:        "traefik",
			PortsJson:   `[{"IP":"0.0.0.0","PrivatePort":8080,"PublicPort":8088,"Type":"tcp"}]`,
		},
	}}
	client := &fakeTraefikClient{}
	tc := NewTraefikCollector(orch, client, "srv1", gate, discardLogger())

	tc.detectTraefik(context.Background())

	assert.True(t, tc.detected)
	assert.Equal(t, 1, orch.listSummaries)
	assert.Equal(t, 0, orch.listContainers)
	assert.Empty(t, orch.inspectCalls)
	assert.Equal(t, "http://127.0.0.1:8088", tc.apiURL)
}

func TestTraefikDetectsImageCaseInsensitive(t *testing.T) {
	gate := &ConnectionGate{conn: &fakeGRPCConn{state: connectivity.Idle}}

	orch := &fakeOrchestrator{summaries: []*agentpb.ContainerInfo{
		{
			ContainerId: "traefik-id",
			Image:       "ghcr.io/acme/TRAEFIK:v3",
			Name:        "traefik",
			PortsJson:   `[{"IP":"0.0.0.0","PrivatePort":8080,"PublicPort":8088,"Type":"tcp"}]`,
		},
	}}
	tc := NewTraefikCollector(orch, &fakeTraefikClient{}, "srv1", gate, discardLogger())

	tc.detectTraefik(context.Background())

	assert.True(t, tc.detected)
	assert.Equal(t, "http://127.0.0.1:8088", tc.apiURL)
}

func TestTraefikInspectsOnlyFoundContainer(t *testing.T) {
	gate := &ConnectionGate{conn: &fakeGRPCConn{state: connectivity.Idle}}

	orch := &fakeOrchestrator{
		summaries: []*agentpb.ContainerInfo{
			{ContainerId: "nginx-id", Image: "nginx:latest", Name: "web"},
			{ContainerId: "traefik-id", Image: "traefik:v3", Name: "traefik"},
		},
		inspectResult: &agentpb.ContainerInfo{
			ContainerId:  "traefik-id",
			NetworksJson: `{"eth0":{"IPAddress":"10.0.0.5"}}`,
		},
	}
	tc := NewTraefikCollector(orch, &fakeTraefikClient{}, "srv1", gate, discardLogger())

	tc.detectTraefik(context.Background())

	assert.True(t, tc.detected)
	assert.Equal(t, 1, orch.listSummaries)
	assert.Equal(t, 0, orch.listContainers)
	assert.Equal(t, []string{"traefik-id"}, orch.inspectCalls)
	assert.Equal(t, "http://10.0.0.5:8080", tc.apiURL)
}

func TestTraefikFallsBackToDefaultWhenInspectFails(t *testing.T) {
	gate := &ConnectionGate{conn: &fakeGRPCConn{state: connectivity.Idle}}

	orch := &fakeOrchestrator{
		summaries: []*agentpb.ContainerInfo{
			{ContainerId: "traefik-id", Image: "traefik:v3", Name: "traefik"},
		},
		inspectErr: errors.New("inspect failed"),
	}
	tc := NewTraefikCollector(orch, &fakeTraefikClient{}, "srv1", gate, discardLogger())

	tc.detectTraefik(context.Background())

	assert.True(t, tc.detected)
	assert.Equal(t, []string{"traefik-id"}, orch.inspectCalls)
	assert.Equal(t, "http://127.0.0.1:8080", tc.apiURL)
}

func TestTraefikDoesNotInspectWhenNoTraefik(t *testing.T) {
	gate := &ConnectionGate{conn: &fakeGRPCConn{state: connectivity.Idle}}

	orch := &fakeOrchestrator{summaries: []*agentpb.ContainerInfo{
		{ContainerId: "nginx-id", Image: "nginx:latest", Name: "web"},
	}}
	tc := NewTraefikCollector(orch, &fakeTraefikClient{}, "srv1", gate, discardLogger())

	tc.detectTraefik(context.Background())

	assert.False(t, tc.detected)
	assert.Equal(t, 1, orch.listSummaries)
	assert.Equal(t, 0, orch.listContainers)
	assert.Empty(t, orch.inspectCalls)
	assert.Empty(t, tc.apiURL)
}
