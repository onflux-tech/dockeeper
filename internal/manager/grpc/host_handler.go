package grpc

import (
	"context"
	"log/slog"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/onflux-tech/dockeeper/internal/manager/events"
	"github.com/onflux-tech/dockeeper/internal/manager/hooks"
	"github.com/onflux-tech/dockeeper/internal/manager/store"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

type HostHandler struct {
	agentpb.UnimplementedAgentHostServiceServer
	app   core.App
	store *store.HostStore
	bus   *events.Bus
}

func NewHostHandler(app core.App, hostStore *store.HostStore, bus *events.Bus) *HostHandler {
	return &HostHandler{app: app, store: hostStore, bus: bus}
}

func (h *HostHandler) ReportHostInventory(ctx context.Context, req *agentpb.ReportHostInventoryRequest) (*agentpb.ReportHostInventoryResponse, error) {
	serverID, ok := ServerIDFromContext(ctx)
	if !ok || serverID == "" {
		serverID = req.GetServerId()
	}
	if serverID == "" {
		return nil, status.Error(codes.Unauthenticated, "mTLS identity required")
	}

	count, err := h.store.UpsertInventory(serverID, req.GetApps(), req.GetProcesses(), req.GetPorts())
	if err != nil {
		slog.Error("host inventory: persist failed", "server_id", serverID, "error", err)
		return nil, status.Error(codes.Internal, "failed to persist host inventory")
	}

	collectedAt := time.Now().UTC()
	if req.GetCollectedAt() != nil {
		collectedAt = req.GetCollectedAt().AsTime().UTC()
	}
	if err := h.store.SaveAppMetrics(serverID, req.GetApps(), collectedAt); err != nil {
		slog.Error("host inventory: save app metrics failed", "server_id", serverID, "error", err)
	}

	if !hooks.UpdateHeartbeat(h.app, h.bus, serverID) {
		return nil, status.Error(codes.NotFound, "server not found")
	}

	return &agentpb.ReportHostInventoryResponse{
		ReceivedApps:      int32(count),
		ReceivedProcesses: int32(len(req.GetProcesses())),
	}, nil
}
