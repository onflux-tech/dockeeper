package collector

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

const metricsInterval = 10 * time.Second

// MetricsCollector periodically collects system and container metrics and
// streams them to the manager via gRPC client-streaming.
type MetricsCollector struct {
	orch         OrchestratorClient
	client       agentpb.AgentMetricsServiceClient
	serverID     string
	logger       *slog.Logger
	system       *SystemCollector
	onServerGone func() // called on mTLS/auth failures to trigger re-enrollment
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

// SetServerGoneCallback sets the function called when a metrics RPC returns
// an authentication error, indicating stale certificates.
func (m *MetricsCollector) SetServerGoneCallback(fn func()) {
	m.onServerGone = fn
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
		if IsAuthError(err) && m.onServerGone != nil {
			m.logger.Warn("metrics: mTLS authentication failed — triggering re-enrollment", "error", err)
			m.onServerGone()
			return
		}
		m.logger.Error("metrics: open system stream failed", "error", err)
		return
	}

	req := &agentpb.StreamMetricsRequest{
		ServerId:      m.serverID,
		Timestamp:     timestamppb.Now(),
		CpuPercent:    sm.CPUPercent,
		MemPercent:    sm.MemPercent,
		MemUsed:       sm.MemUsed,
		MemTotal:      sm.MemTotal,
		DiskUsed:      sm.DiskUsed,
		DiskTotal:     sm.DiskTotal,
		NetRx:         sm.NetRx,
		NetTx:         sm.NetTx,
		Load_1M:       sm.Load1m,
		Load_5M:       sm.Load5m,
		Load_15M:      sm.Load15m,
		DiskIoRead:    sm.DiskIORead,
		DiskIoWrite:   sm.DiskIOWrite,
		SwapUsed:      sm.SwapUsed,
		SwapTotal:     sm.SwapTotal,
		MemAvailable:  sm.MemAvailable,
		CpuCores:      sm.CPUCores,
		UptimeSeconds: sm.UptimeSeconds,
	}

	// Map temperature sensors.
	for _, t := range sm.Temperatures {
		req.Temperatures = append(req.Temperatures, &agentpb.TemperatureSensor{
			Name:    t.Name,
			Celsius: t.Celsius,
		})
	}

	// Map disk partitions.
	for _, p := range sm.Partitions {
		req.Partitions = append(req.Partitions, &agentpb.DiskPartition{
			Mountpoint: p.Mountpoint,
			Device:     p.Device,
			Fstype:     p.FSType,
			Total:      p.Total,
			Used:       p.Used,
			Available:  p.Available,
			Percent:    p.Percent,
		})
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
		if IsAuthError(err) && m.onServerGone != nil {
			m.logger.Warn("metrics: mTLS authentication failed — triggering re-enrollment", "error", err)
			m.onServerGone()
			return
		}
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
