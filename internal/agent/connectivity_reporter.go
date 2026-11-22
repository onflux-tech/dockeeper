package agent

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/onflux-tech/dockeeper/internal/agent/collector"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

type ConnectivityReporter struct {
	client       agentpb.AgentHostServiceClient
	serverID     string
	logger       *slog.Logger
	onServerGone func()
}

func NewConnectivityReporter(client agentpb.AgentHostServiceClient, serverID string, logger *slog.Logger) *ConnectivityReporter {
	return &ConnectivityReporter{client: client, serverID: serverID, logger: logger}
}

func (r *ConnectivityReporter) SetServerGoneCallback(fn func()) {
	r.onServerGone = fn
}

func (r *ConnectivityReporter) Report(ctx context.Context, report *collector.ConnectivityReport) error {
	if report == nil {
		return nil
	}

	req := &agentpb.ReportConnectivityRequest{
		ServerId:      r.serverID,
		InternetOk:    report.InternetOK,
		InternetMs:    report.InternetMS,
		DnsOk:         report.DNSOK,
		DnsMs:         report.DNSMS,
		GatewayOk:     report.GatewayOK,
		GatewayMs:     report.GatewayMS,
		GatewayMethod: report.GatewayMethod,
		Error:         report.Error,
	}
	if !report.CollectedAt.IsZero() {
		req.CollectedAt = timestamppb.New(report.CollectedAt)
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	if _, err := r.client.ReportConnectivity(ctx, req); err != nil {
		if collector.IsAuthError(err) && r.onServerGone != nil {
			r.logger.Warn("connectivity: mTLS authentication failed — triggering re-enrollment", "error", err)
			r.onServerGone()
			return nil
		}
		return fmt.Errorf("report connectivity: %w", err)
	}

	return nil
}
