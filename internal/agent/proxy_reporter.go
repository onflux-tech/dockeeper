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

type ProxyReporter struct {
	client       agentpb.AgentHostServiceClient
	serverID     string
	logger       *slog.Logger
	onServerGone func()
}

func NewProxyReporter(client agentpb.AgentHostServiceClient, serverID string, logger *slog.Logger) *ProxyReporter {
	return &ProxyReporter{client: client, serverID: serverID, logger: logger}
}

func (r *ProxyReporter) SetServerGoneCallback(fn func()) {
	r.onServerGone = fn
}

func (r *ProxyReporter) Report(ctx context.Context, instances []*agentpb.ProxyInstanceInfo) error {
	req := &agentpb.ReportProxyInventoryRequest{
		ServerId:    r.serverID,
		Instances:   instances,
		CollectedAt: timestamppb.Now(),
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	if _, err := r.client.ReportProxyInventory(ctx, req); err != nil {
		if collector.IsAuthError(err) && r.onServerGone != nil {
			r.logger.Warn("proxies: mTLS authentication failed — triggering re-enrollment", "error", err)
			r.onServerGone()
			return nil
		}
		return fmt.Errorf("report proxy inventory: %w", err)
	}

	return nil
}
