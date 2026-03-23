package grpc

import (
	"io"
	"log/slog"

	"github.com/pocketbase/pocketbase/core"

	"github.com/jkdesenvolvedor/dockeeper/internal/manager/hooks"
	"github.com/jkdesenvolvedor/dockeeper/internal/manager/store"
	agentpb "github.com/jkdesenvolvedor/dockeeper/proto/gen/dockeeper/agent/v1"
)

// MetricsHandler implements AgentMetricsService, receiving streamed metrics
// from agents and persisting them in PocketBase.
type MetricsHandler struct {
	agentpb.UnimplementedAgentMetricsServiceServer
	app     core.App
	metrics *store.MetricsStore
}

// NewMetricsHandler creates a new MetricsHandler.
func NewMetricsHandler(app core.App, metrics *store.MetricsStore) *MetricsHandler {
	return &MetricsHandler{app: app, metrics: metrics}
}

// StreamMetrics receives a client-stream of system metrics and stores each one.
func (h *MetricsHandler) StreamMetrics(stream agentpb.AgentMetricsService_StreamMetricsServer) error {
	serverID, ok := ServerIDFromContext(stream.Context())
	if !ok {
		slog.Warn("metrics: no server_id in context")
	}

	var count int32
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&agentpb.StreamMetricsResponse{
				ReceivedCount: count,
			})
		}
		if err != nil {
			return err
		}

		// Use mTLS identity; fall back to request field.
		sid := serverID
		if sid == "" {
			sid = req.GetServerId()
		}
		if sid == "" {
			continue
		}

		if err := h.metrics.SaveServerMetrics(sid, req); err != nil {
			slog.Error("metrics: save server metrics failed", "server_id", sid, "error", err)
			continue
		}
		count++

		hooks.UpdateHeartbeat(h.app, sid)
	}
}

// StreamContainerMetrics receives a client-stream of per-container metrics.
func (h *MetricsHandler) StreamContainerMetrics(stream agentpb.AgentMetricsService_StreamContainerMetricsServer) error {
	serverID, ok := ServerIDFromContext(stream.Context())
	if !ok {
		slog.Warn("container metrics: no server_id in context")
	}

	var count int32
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&agentpb.StreamContainerMetricsResponse{
				ReceivedCount: count,
			})
		}
		if err != nil {
			return err
		}

		sid := serverID
		if sid == "" {
			sid = req.GetServerId()
		}
		if sid == "" {
			continue
		}

		if err := h.metrics.SaveContainerMetrics(sid, req); err != nil {
			slog.Error("metrics: save container metrics failed",
				"server_id", sid, "container_id", req.GetContainerId(), "error", err)
			continue
		}
		count++
	}
}
