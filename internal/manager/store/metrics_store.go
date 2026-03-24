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

	if err := s.app.Save(rec); err != nil {
		return fmt.Errorf("save server metric: %w", err)
	}
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
	return nil
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
