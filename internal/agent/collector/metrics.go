package collector

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	agentpb "github.com/jkdesenvolvedor/dockeeper/proto/gen/dockeeper/agent/v1"
)

const metricsInterval = 10 * time.Second

// MetricsCollector periodically collects system and container metrics and
// streams them to the manager via gRPC client-streaming.
type MetricsCollector struct {
	orch     OrchestratorClient
	client   agentpb.AgentMetricsServiceClient
	serverID string
	logger   *slog.Logger
	system   *SystemCollector
}

// NewMetricsCollector creates a new MetricsCollector.
func NewMetricsCollector(
	orch OrchestratorClient,
	client agentpb.AgentMetricsServiceClient,
	serverID string,
	logger *slog.Logger,
) *MetricsCollector {
	return &MetricsCollector{
		orch:     orch,
		client:   client,
		serverID: serverID,
		logger:   logger,
		system:   NewSystemCollector(),
	}
}

// Run starts the periodic metrics collection loop. It blocks until ctx is
// cancelled. System and container metrics are streamed over separate gRPC
// client-streaming RPCs that are reopened on error.
func (m *MetricsCollector) Run(ctx context.Context) {
	// First collection to prime CPU/net deltas (results will have zero deltas).
	m.collectSystem(ctx)

	ticker := time.NewTicker(metricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.collectSystem(ctx)
			m.collectContainers(ctx)
		}
	}
}

func (m *MetricsCollector) collectSystem(ctx context.Context) {
	sm, err := m.system.Collect(ctx)
	if err != nil {
		m.logger.Error("metrics: collect system failed", "error", err)
		return
	}

	stream, err := m.client.StreamMetrics(ctx)
	if err != nil {
		m.logger.Error("metrics: open system stream failed", "error", err)
		return
	}

	req := &agentpb.StreamMetricsRequest{
		ServerId:   m.serverID,
		Timestamp:  timestamppb.Now(),
		CpuPercent: sm.CPUPercent,
		MemPercent: sm.MemPercent,
		MemUsed:    sm.MemUsed,
		MemTotal:   sm.MemTotal,
		DiskUsed:   sm.DiskUsed,
		DiskTotal:  sm.DiskTotal,
		NetRx:      sm.NetRx,
		NetTx:      sm.NetTx,
		Load_1M:    sm.Load1m,
		Load_5M:    sm.Load5m,
		Load_15M:   sm.Load15m,
	}

	if err := stream.Send(req); err != nil {
		m.logger.Error("metrics: send system metrics failed", "error", err)
		return
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		m.logger.Error("metrics: close system stream failed", "error", err)
		return
	}
	m.logger.Debug("metrics: system metrics streamed", "received_count", resp.GetReceivedCount())
}

func (m *MetricsCollector) collectContainers(ctx context.Context) {
	stats, err := m.orch.ListContainerStats(ctx)
	if err != nil {
		m.logger.Error("metrics: collect container stats failed", "error", err)
		return
	}

	if len(stats) == 0 {
		return
	}

	stream, err := m.client.StreamContainerMetrics(ctx)
	if err != nil {
		m.logger.Error("metrics: open container stream failed", "error", err)
		return
	}

	now := timestamppb.Now()
	for _, s := range stats {
		req := &agentpb.StreamContainerMetricsRequest{
			ServerId:    m.serverID,
			ContainerId: s.ContainerID,
			Timestamp:   now,
			CpuPercent:  s.CPUPercent,
			MemUsed:     s.MemUsed,
			MemLimit:    s.MemLimit,
			NetRx:       s.NetRx,
			NetTx:       s.NetTx,
			BlockRead:   s.BlockRead,
			BlockWrite:  s.BlockWrite,
			Pids:        s.PIDs,
		}
		if err := stream.Send(req); err != nil {
			m.logger.Error("metrics: send container metrics failed", "container", s.ContainerID, "error", err)
			return
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		m.logger.Error("metrics: close container stream failed", "error", err)
		return
	}
	m.logger.Debug("metrics: container metrics streamed", "received_count", resp.GetReceivedCount())
}
