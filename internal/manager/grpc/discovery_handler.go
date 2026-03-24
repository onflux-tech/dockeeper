package grpc

import (
	"context"
	"log/slog"

	"github.com/pocketbase/pocketbase/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/onflux-tech/dockeeper/internal/manager/hooks"
	"github.com/onflux-tech/dockeeper/internal/manager/store"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// DiscoveryHandler implements AgentDiscoveryService, receiving resource reports
// from agents and upserting them in PocketBase.
type DiscoveryHandler struct {
	agentpb.UnimplementedAgentDiscoveryServiceServer
	app       core.App
	resources *store.ResourceStore
}

// NewDiscoveryHandler creates a new DiscoveryHandler.
func NewDiscoveryHandler(app core.App, resources *store.ResourceStore) *DiscoveryHandler {
	return &DiscoveryHandler{app: app, resources: resources}
}

// ReportContainers upserts container records and marks removed containers.
func (h *DiscoveryHandler) ReportContainers(ctx context.Context, req *agentpb.ReportContainersRequest) (*agentpb.ReportContainersResponse, error) {
	sid := h.serverID(ctx, req.GetServerId())
	if sid == "" {
		return nil, status.Error(codes.Unauthenticated, "unknown server")
	}

	if !h.serverExists(sid) {
		return nil, status.Error(codes.NotFound, "server not found")
	}

	count, err := h.resources.UpsertContainers(sid, req.GetContainers())
	if err != nil {
		slog.Error("discovery: upsert containers failed", "server_id", sid, "error", err)
		return nil, status.Error(codes.Internal, "failed to store containers")
	}

	hooks.UpdateHeartbeat(h.app, sid)
	return &agentpb.ReportContainersResponse{ReceivedCount: int32(count)}, nil
}

// ReportImages upserts image records for a server.
func (h *DiscoveryHandler) ReportImages(ctx context.Context, req *agentpb.ReportImagesRequest) (*agentpb.ReportImagesResponse, error) {
	sid := h.serverID(ctx, req.GetServerId())
	if sid == "" {
		return nil, status.Error(codes.Unauthenticated, "unknown server")
	}

	if !h.serverExists(sid) {
		return nil, status.Error(codes.NotFound, "server not found")
	}

	count, err := h.resources.UpsertImages(sid, req.GetImages())
	if err != nil {
		slog.Error("discovery: upsert images failed", "server_id", sid, "error", err)
		return nil, status.Error(codes.Internal, "failed to store images")
	}

	hooks.UpdateHeartbeat(h.app, sid)
	return &agentpb.ReportImagesResponse{ReceivedCount: int32(count)}, nil
}

// ReportNetworks upserts network records for a server.
func (h *DiscoveryHandler) ReportNetworks(ctx context.Context, req *agentpb.ReportNetworksRequest) (*agentpb.ReportNetworksResponse, error) {
	sid := h.serverID(ctx, req.GetServerId())
	if sid == "" {
		return nil, status.Error(codes.Unauthenticated, "unknown server")
	}

	if !h.serverExists(sid) {
		return nil, status.Error(codes.NotFound, "server not found")
	}

	count, err := h.resources.UpsertNetworks(sid, req.GetNetworks())
	if err != nil {
		slog.Error("discovery: upsert networks failed", "server_id", sid, "error", err)
		return nil, status.Error(codes.Internal, "failed to store networks")
	}

	hooks.UpdateHeartbeat(h.app, sid)
	return &agentpb.ReportNetworksResponse{ReceivedCount: int32(count)}, nil
}

// ReportVolumes upserts volume records for a server.
func (h *DiscoveryHandler) ReportVolumes(ctx context.Context, req *agentpb.ReportVolumesRequest) (*agentpb.ReportVolumesResponse, error) {
	sid := h.serverID(ctx, req.GetServerId())
	if sid == "" {
		return nil, status.Error(codes.Unauthenticated, "unknown server")
	}

	if !h.serverExists(sid) {
		return nil, status.Error(codes.NotFound, "server not found")
	}

	count, err := h.resources.UpsertVolumes(sid, req.GetVolumes())
	if err != nil {
		slog.Error("discovery: upsert volumes failed", "server_id", sid, "error", err)
		return nil, status.Error(codes.Internal, "failed to store volumes")
	}

	hooks.UpdateHeartbeat(h.app, sid)
	return &agentpb.ReportVolumesResponse{ReceivedCount: int32(count)}, nil
}

// serverID returns the server_id from mTLS context, falling back to the request field.
func (h *DiscoveryHandler) serverID(ctx context.Context, reqServerID string) string {
	if sid, ok := ServerIDFromContext(ctx); ok {
		return sid
	}
	return reqServerID
}

// serverExists checks whether the server record still exists in PocketBase.
func (h *DiscoveryHandler) serverExists(serverID string) bool {
	_, err := h.app.FindRecordById("servers", serverID)
	return err == nil
}
