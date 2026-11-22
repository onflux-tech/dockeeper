package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/onflux-tech/dockeeper/internal/manager/hooks"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

func (h *HostHandler) ReportProxyInventory(ctx context.Context, req *agentpb.ReportProxyInventoryRequest) (*agentpb.ReportProxyInventoryResponse, error) {
	serverID, ok := ServerIDFromContext(ctx)
	if !ok || serverID == "" {
		serverID = req.GetServerId()
	}
	if serverID == "" {
		return nil, status.Error(codes.Unauthenticated, "mTLS identity required")
	}

	instances, routes, err := h.proxies.Replace(serverID, req.GetInstances())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to persist proxy inventory")
	}

	if !hooks.UpdateHeartbeat(h.app, h.bus, serverID) {
		return nil, status.Error(codes.NotFound, "server not found")
	}

	return &agentpb.ReportProxyInventoryResponse{
		ReceivedInstances: int32(instances),
		ReceivedRoutes:    int32(routes),
	}, nil
}
