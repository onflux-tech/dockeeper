package grpc_handlers

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflux-tech/dockeeper/internal/agent/collector"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

func TestHostHealthHTTPStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	parsed, err := url.Parse(server.URL)
	require.NoError(t, err)
	port, err := strconv.Atoi(parsed.Port())
	require.NoError(t, err)

	h := NewHostHealthHandler(slog.Default())
	result := h.Execute(context.Background(), &agentpb.HostHealthCheckCommand{
		CheckType:      "http",
		Host:           "127.0.0.1",
		Port:           uint32(port),
		Scheme:         "http",
		Method:         "GET",
		Path:           "/",
		TimeoutMs:      1000,
		ExpectedStatus: http.StatusNoContent,
	})

	require.True(t, result.GetSuccess(), result.GetError())
	require.Equal(t, int32(http.StatusNoContent), result.GetStatusCode())
}

func TestHostHealthHTTPExpectedBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("all good"))
	}))
	defer server.Close()

	parsed, err := url.Parse(server.URL)
	require.NoError(t, err)
	port, err := strconv.Atoi(parsed.Port())
	require.NoError(t, err)

	h := NewHostHealthHandler(slog.Default())
	ok := h.Execute(context.Background(), &agentpb.HostHealthCheckCommand{
		CheckType: "http", Host: "127.0.0.1", Port: uint32(port), Scheme: "http",
		Method: "GET", Path: "/", TimeoutMs: 1000, ExpectedStatus: 200, ExpectedBody: "all good",
	})
	require.True(t, ok.GetSuccess(), ok.GetError())

	missing := h.Execute(context.Background(), &agentpb.HostHealthCheckCommand{
		CheckType: "http", Host: "127.0.0.1", Port: uint32(port), Scheme: "http",
		Method: "GET", Path: "/", TimeoutMs: 1000, ExpectedStatus: 200, ExpectedBody: "missing",
	})
	require.False(t, missing.GetSuccess())
	require.Contains(t, missing.GetError(), "expected body")
}

func TestHostHealthTCPClosed(t *testing.T) {
	h := NewHostHealthHandler(slog.Default())

	result := h.Execute(context.Background(), &agentpb.HostHealthCheckCommand{
		CheckType: "tcp",
		Host:      "127.0.0.1",
		Port:      1,
		TimeoutMs: 100,
	})

	require.False(t, result.GetSuccess())
}

func TestHostHealthProcessUsesInventory(t *testing.T) {
	h := NewHostHealthHandler(slog.Default(),
		WithHostInventory(func() []collector.HostApp {
			return []collector.HostApp{{Source: "pm2", Name: "api-takeflow", Status: "running"}}
		}),
	)

	result := h.Execute(context.Background(), &agentpb.HostHealthCheckCommand{
		CheckType: "process", Source: "pm2", Name: "api-takeflow",
	})

	require.True(t, result.GetSuccess(), result.GetError())

	missing := h.Execute(context.Background(), &agentpb.HostHealthCheckCommand{
		CheckType: "process", Source: "pm2", Name: "gone",
	})
	require.False(t, missing.GetSuccess())
}

func TestHostHealthRejectsURLCredentials(t *testing.T) {
	h := NewHostHealthHandler(slog.Default())

	result := h.Execute(context.Background(), &agentpb.HostHealthCheckCommand{
		CheckType: "http",
		Host:      "user:pass@127.0.0.1",
		Port:      80,
		Scheme:    "http",
		TimeoutMs: 1000,
	})

	require.False(t, result.GetSuccess())
	require.Contains(t, result.GetError(), "invalid host")
}
