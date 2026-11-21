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

type HostReporter struct {
	client       agentpb.AgentHostServiceClient
	serverID     string
	logger       *slog.Logger
	onServerGone func()
}

func NewHostReporter(client agentpb.AgentHostServiceClient, serverID string, logger *slog.Logger) *HostReporter {
	return &HostReporter{client: client, serverID: serverID, logger: logger}
}

func (r *HostReporter) SetServerGoneCallback(fn func()) {
	r.onServerGone = fn
}

func (r *HostReporter) Report(ctx context.Context, inventory *collector.HostInventory) error {
	if inventory == nil {
		return nil
	}

	req := &agentpb.ReportHostInventoryRequest{
		ServerId:      r.serverID,
		UptimeSeconds: inventory.UptimeSeconds,
	}
	if !inventory.CollectedAt.IsZero() {
		req.CollectedAt = timestamppb.New(inventory.CollectedAt)
	}

	for _, app := range inventory.Apps {
		req.Apps = append(req.Apps, &agentpb.HostApp{
			Source:         app.Source,
			Name:           app.Name,
			Unit:           app.Unit,
			PmId:           app.PMID,
			Status:         app.Status,
			Pids:           app.PIDs,
			Ports:          app.Ports,
			CpuPercent:     app.CPUPercent,
			MemBytes:       app.MemBytes,
			DiskReadBytes:  app.DiskReadBytes,
			DiskWriteBytes: app.DiskWriteBytes,
			Restarts:       app.Restarts,
			UptimeSeconds:  app.UptimeSeconds,
			User:           app.User,
			Cwd:            app.Cwd,
			ExecPath:       app.ExecPath,
			LogPath:        app.LogPath,
		})
	}

	for _, process := range inventory.Processes {
		req.Processes = append(req.Processes, &agentpb.HostProcess{
			Pid:        process.PID,
			Name:       process.Name,
			App:        process.App,
			CpuPercent: process.CPUPercent,
			MemBytes:   process.MemBytes,
			Threads:    process.Threads,
		})
	}

	for _, port := range inventory.Ports {
		req.Ports = append(req.Ports, &agentpb.HostPort{
			Port:     port.Port,
			Protocol: port.Protocol,
			Address:  port.Address,
			Pid:      port.PID,
			App:      port.App,
		})
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	if _, err := r.client.ReportHostInventory(ctx, req); err != nil {
		if collector.IsAuthError(err) && r.onServerGone != nil {
			r.logger.Warn("host inventory: mTLS authentication failed — triggering re-enrollment", "error", err)
			r.onServerGone()
			return nil
		}
		return fmt.Errorf("report host inventory: %w", err)
	}

	return nil
}
