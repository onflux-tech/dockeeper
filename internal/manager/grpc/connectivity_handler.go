package grpc

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/onflux-tech/dockeeper/internal/manager/events"
	"github.com/onflux-tech/dockeeper/internal/manager/hooks"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

func (h *HostHandler) ReportConnectivity(ctx context.Context, req *agentpb.ReportConnectivityRequest) (*agentpb.ReportConnectivityResponse, error) {
	serverID, ok := ServerIDFromContext(ctx)
	if !ok || serverID == "" {
		serverID = req.GetServerId()
	}
	if serverID == "" {
		return nil, status.Error(codes.Unauthenticated, "mTLS identity required")
	}

	previous := ""
	serverName := ""
	if server, err := h.app.FindRecordById(constants.CollServers, serverID); err == nil {
		previous = server.GetString("connectivity_status")
		serverName = server.GetString("hostname")
	}

	connectivityStatus, err := h.connectivity.Save(serverID, req)
	if err != nil {
		slog.Error("connectivity: persist failed", "server_id", serverID, "error", err)
		return nil, status.Error(codes.Internal, "failed to persist connectivity")
	}

	if !hooks.UpdateHeartbeat(h.app, h.bus, serverID) {
		return nil, status.Error(codes.NotFound, "server not found")
	}

	if h.bus != nil {
		details := map[string]any{
			"internet_ok":    req.GetInternetOk(),
			"internet_ms":    req.GetInternetMs(),
			"dns_ok":         req.GetDnsOk(),
			"dns_ms":         req.GetDnsMs(),
			"gateway_ok":     req.GetGatewayOk(),
			"gateway_ms":     req.GetGatewayMs(),
			"gateway_method": req.GetGatewayMethod(),
			"error":          req.GetError(),
		}
		switch {
		case previous != "degraded" && connectivityStatus == "degraded":
			h.bus.Publish(events.Event{
				Type:       events.ServerConnectivityDegraded,
				Severity:   events.SeverityFor(events.ServerConnectivityDegraded),
				ServerID:   serverID,
				ServerName: serverName,
				Details:    details,
				Timestamp:  time.Now().UTC(),
			})
		case previous == "degraded" && connectivityStatus == "healthy":
			h.bus.Publish(events.Event{
				Type:       events.ServerConnectivityRecovered,
				Severity:   events.SeverityFor(events.ServerConnectivityRecovered),
				ServerID:   serverID,
				ServerName: serverName,
				Details:    details,
				Timestamp:  time.Now().UTC(),
			})
		}
	}

	return &agentpb.ReportConnectivityResponse{}, nil
}
