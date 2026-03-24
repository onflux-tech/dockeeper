package hooks

import (
	"log/slog"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

const (
	offlineThreshold = 60 * time.Second
)

// RegisterOfflineChecker registers a cron job that marks agents as offline
// when they haven't sent a heartbeat within the threshold.
func RegisterOfflineChecker(app core.App) {
	if err := app.Cron().Add("agent_offline_check", "*/1 * * * *", func() {
		checkOfflineAgents(app)
	}); err != nil {
		slog.Error("failed to register offline checker cron", "error", err)
	}
}

func checkOfflineAgents(app core.App) {
	cutoff := time.Now().Add(-offlineThreshold).UTC().Format("2006-01-02 15:04:05.000Z")

	records, err := app.FindRecordsByFilter(
		constants.CollServers,
		"status = 'online' && last_seen < {:cutoff}",
		"", 0, 0,
		map[string]any{"cutoff": cutoff},
	)
	if err != nil {
		slog.Error("offline check query failed", "error", err)
		return
	}

	for _, r := range records {
		r.Set("status", "offline")
		if err := app.Save(r); err != nil {
			slog.Error("failed to mark agent offline", "server_id", r.Id, "error", err)
			continue
		}
		slog.Warn("agent marked offline", "server_id", r.Id, "hostname", r.GetString("hostname"))
	}
}

// UpdateHeartbeat sets the server's last_seen to now and status to online.
// It returns false if the server record does not exist (e.g. deleted).
func UpdateHeartbeat(app core.App, serverID string) bool {
	record, err := app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		slog.Error("heartbeat: server not found", "server_id", serverID, "error", err)
		return false
	}

	record.Set("last_seen", time.Now().UTC())
	record.Set("status", "online")
	if err := app.Save(record); err != nil {
		slog.Error("heartbeat: failed to update", "server_id", serverID, "error", err)
	}
	return true
}
