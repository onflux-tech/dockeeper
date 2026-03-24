package grpc

import (
	"context"
	"log/slog"
	"net"

	"github.com/pocketbase/pocketbase/core"
	"google.golang.org/grpc/peer"

	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

func peerHostFromContext(ctx context.Context) (string, bool) {
	p, ok := peer.FromContext(ctx)
	if !ok || p.Addr == nil {
		return "", false
	}

	host, _, err := net.SplitHostPort(p.Addr.String())
	if err == nil && host != "" {
		return host, true
	}

	if ip := net.ParseIP(p.Addr.String()); ip != nil {
		return ip.String(), true
	}

	return "", false
}

func syncServerAddressFromContext(app core.App, ctx context.Context, serverID string) {
	if serverID == "" {
		return
	}

	host, ok := peerHostFromContext(ctx)
	if !ok || host == "" {
		return
	}

	record, err := app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		slog.Warn("failed to load server for address sync", "server_id", serverID, "error", err)
		return
	}

	if record.GetString("ip_address") == host {
		return
	}

	record.Set("ip_address", host)
	if err := app.Save(record); err != nil {
		slog.Warn("failed to sync server address", "server_id", serverID, "ip_address", host, "error", err)
		return
	}

	slog.Info("server address synced from gRPC peer", "server_id", serverID, "ip_address", host)
}
