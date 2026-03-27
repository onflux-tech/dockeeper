package grpc

import (
	"context"
	"log/slog"

	"github.com/pocketbase/pocketbase/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/onflux-tech/dockeeper/internal/manager/events"
	"github.com/onflux-tech/dockeeper/internal/manager/hooks"
	"github.com/onflux-tech/dockeeper/internal/manager/store"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// TraefikHandler implements AgentTraefikService, receiving Traefik route
// reports from agents and storing them in PocketBase.
type TraefikHandler struct {
	agentpb.UnimplementedAgentTraefikServiceServer
	app   core.App
	store *store.TraefikStore
	bus   *events.Bus
}

// NewTraefikHandler creates a new TraefikHandler.
func NewTraefikHandler(app core.App, traefikStore *store.TraefikStore, bus *events.Bus) *TraefikHandler {
	return &TraefikHandler{app: app, store: traefikStore, bus: bus}
}

// ReportTraefikRoutes upserts Traefik route records for a server.
func (h *TraefikHandler) ReportTraefikRoutes(ctx context.Context, req *agentpb.ReportTraefikRoutesRequest) (*agentpb.ReportTraefikRoutesResponse, error) {
	sid := h.serverID(ctx, req.GetServerId())
	if sid == "" {
		return nil, status.Error(codes.Unauthenticated, "unknown server")
	}
	syncServerAddressFromContext(h.app, ctx, sid)

	count, err := h.store.UpsertRoutes(sid, req.GetRoutes())
	if err != nil {
		slog.Error("traefik: upsert routes failed", "server_id", sid, "error", err)
		return nil, status.Error(codes.Internal, "failed to store traefik routes")
	}

	hooks.UpdateHeartbeat(h.app, h.bus, sid)
	return &agentpb.ReportTraefikRoutesResponse{ReceivedCount: int32(count)}, nil
}

// serverID returns the server_id from mTLS context, falling back to the request field.
func (h *TraefikHandler) serverID(ctx context.Context, reqServerID string) string {
	if sid, ok := ServerIDFromContext(ctx); ok {
		return sid
	}
	return reqServerID
}
