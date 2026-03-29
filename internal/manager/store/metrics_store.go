package store

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/shared/constants"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// MetricsStore persists raw metrics data into PocketBase collections.
type MetricsStore struct {
	app core.App
}

// NewMetricsStore creates a new MetricsStore.
func NewMetricsStore(app core.App) *MetricsStore {
	return &MetricsStore{app: app}
}

// SaveServerMetrics inserts a row into server_metrics_raw.
func (s *MetricsStore) SaveServerMetrics(serverID string, req *agentpb.StreamMetricsRequest) error {
	col, err := s.app.FindCachedCollectionByNameOrId(constants.CollServerMetricsRaw)
	if err != nil {
		return fmt.Errorf("find collection %s: %w", constants.CollServerMetricsRaw, err)
	}

	rec := core.NewRecord(col)
	rec.Set("server", serverID)
	rec.Set("timestamp", req.GetTimestamp().AsTime().UTC())
	rec.Set("cpu_percent", req.GetCpuPercent())
	rec.Set("mem_percent", req.GetMemPercent())
	rec.Set("mem_used", req.GetMemUsed())
	rec.Set("mem_total", req.GetMemTotal())
	rec.Set("disk_used", req.GetDiskUsed())
	rec.Set("disk_total", req.GetDiskTotal())
	rec.Set("net_rx", req.GetNetRx())
	rec.Set("net_tx", req.GetNetTx())
	rec.Set("load_1m", req.GetLoad_1M())
	rec.Set("load_5m", req.GetLoad_5M())
	rec.Set("load_15m", req.GetLoad_15M())
	rec.Set("disk_io_read", req.GetDiskIoRead())
	rec.Set("disk_io_write", req.GetDiskIoWrite())
	rec.Set("swap_used", req.GetSwapUsed())
	rec.Set("swap_total", req.GetSwapTotal())
	rec.Set("mem_available", req.GetMemAvailable())
	rec.Set("cpu_cores", req.GetCpuCores())
	rec.Set("uptime_seconds", req.GetUptimeSeconds())

	// Map partitions to JSON.
	if parts := req.GetPartitions(); len(parts) > 0 {
		type partJSON struct {
			Mountpoint string  `json:"mountpoint"`
			Device     string  `json:"device"`
			FSType     string  `json:"fstype"`
			Total      uint64  `json:"total"`
			Used       uint64  `json:"used"`
			Available  uint64  `json:"available"`
			Percent    float64 `json:"percent"`
		}
		plist := make([]partJSON, 0, len(parts))
		for _, p := range parts {
			plist = append(plist, partJSON{
				Mountpoint: p.GetMountpoint(),
				Device:     p.GetDevice(),
				FSType:     p.GetFstype(),
				Total:      p.GetTotal(),
				Used:       p.GetUsed(),
				Available:  p.GetAvailable(),
				Percent:    p.GetPercent(),
			})
		}
		rec.Set("partitions", plist)
	}

	// Map temperatures to JSON.
	if temps := req.GetTemperatures(); len(temps) > 0 {
		type tempJSON struct {
			Name    string  `json:"name"`
			Celsius float64 `json:"celsius"`
		}
		tlist := make([]tempJSON, 0, len(temps))
		for _, t := range temps {
			tlist = append(tlist, tempJSON{
				Name:    t.GetName(),
				Celsius: t.GetCelsius(),
			})
		}
		rec.Set("temperatures", tlist)
	}

	if err := s.app.Save(rec); err != nil {
		return fmt.Errorf("save server metric: %w", err)
	}

	// Denormalize current metrics to the server record for fast table queries.
	s.denormalizeServerMetrics(serverID, req)

	return nil
}

// SaveContainerMetrics inserts a row into container_metrics_raw.
// It resolves the container's PB record ID from (server, container_id).
// If the container is not yet known (no discovery report yet), the metric is
// silently dropped.
func (s *MetricsStore) SaveContainerMetrics(serverID string, req *agentpb.StreamContainerMetricsRequest) error {
	containerRecID, err := s.resolveContainerID(serverID, req.GetContainerId())
	if err != nil {
		slog.Debug("metrics: container not found, skipping metric",
			"server_id", serverID, "container_id", req.GetContainerId())
		return nil
	}

	col, err := s.app.FindCachedCollectionByNameOrId(constants.CollContainerMetricsRaw)
	if err != nil {
		return fmt.Errorf("find collection %s: %w", constants.CollContainerMetricsRaw, err)
	}

	ts := time.Now().UTC()
	if req.GetTimestamp() != nil {
		ts = req.GetTimestamp().AsTime().UTC()
	}

	rec := core.NewRecord(col)
	rec.Set("container", containerRecID)
	rec.Set("server", serverID)
	rec.Set("timestamp", ts)
	rec.Set("cpu_percent", req.GetCpuPercent())
	rec.Set("mem_used", req.GetMemUsed())
	rec.Set("mem_limit", req.GetMemLimit())
	rec.Set("net_rx", req.GetNetRx())
	rec.Set("net_tx", req.GetNetTx())
	rec.Set("block_read", req.GetBlockRead())
	rec.Set("block_write", req.GetBlockWrite())
	rec.Set("pids", req.GetPids())

	if err := s.app.Save(rec); err != nil {
		return fmt.Errorf("save container metric: %w", err)
	}

	// Denormalize current metrics to the container record for fast table queries.
	s.denormalizeContainerMetrics(containerRecID, req)

	return nil
}

// denormalizeServerMetrics updates the server record with current live metrics.
func (s *MetricsStore) denormalizeServerMetrics(serverID string, req *agentpb.StreamMetricsRequest) {
	server, err := s.app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		slog.Debug("metrics: denormalize server failed to find record", "server_id", serverID, "error", err)
		return
	}
	server.Set("current_cpu", req.GetCpuPercent())
	server.Set("current_ram", req.GetMemPercent())
	if req.GetDiskTotal() > 0 {
		server.Set("current_disk", float64(req.GetDiskUsed())/float64(req.GetDiskTotal())*100)
	}
	server.Set("uptime_seconds", req.GetUptimeSeconds())
	server.Set("last_seen", time.Now().UTC())
	if err := s.app.Save(server); err != nil {
		slog.Error("metrics: denormalize server metrics failed", "server_id", serverID, "error", err)
	}
}

// denormalizeContainerMetrics updates the container record with current live metrics.
func (s *MetricsStore) denormalizeContainerMetrics(containerRecID string, req *agentpb.StreamContainerMetricsRequest) {
	container, err := s.app.FindRecordById(constants.CollContainers, containerRecID)
	if err != nil {
		slog.Debug("metrics: denormalize container failed to find record", "container_id", containerRecID, "error", err)
		return
	}
	container.Set("current_cpu", req.GetCpuPercent())
	container.Set("current_ram", req.GetMemUsed())
	container.Set("current_ram_limit", req.GetMemLimit())
	if err := s.app.Save(container); err != nil {
		slog.Error("metrics: denormalize container metrics failed", "container_id", containerRecID, "error", err)
	}
}

// resolveContainerID looks up the PB record ID for a container by server + container_id.
func (s *MetricsStore) resolveContainerID(serverID, containerID string) (string, error) {
	records, err := s.app.FindRecordsByFilter(
		constants.CollContainers,
		"server = {:sid} && container_id = {:cid}",
		"", 1, 0,
		map[string]any{"sid": serverID, "cid": containerID},
	)
	if err != nil || len(records) == 0 {
		return "", fmt.Errorf("container not found")
	}
	return records[0].Id, nil
}
