package jobs

import (
	"log/slog"
	"strconv"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const deleteBatchSize = 1000

// Default retention periods (used as fallback if settings are not configured).
const (
	defaultRawHours      = 168 // 7 days
	defaultHourlyDays    = 30
	defaultDailyDays     = 365
	defaultEventsDays    = 30
	defaultAuditLogsDays = 90
	defaultNotifLogsDays = 30
)

// CleanupOldMetrics removes expired data according to configurable retention policies.
func CleanupOldMetrics(app core.App) {
	slog.Info("job: data cleanup started")

	rawHours := getRetentionSetting(app, "retention_raw_hours", defaultRawHours)
	hourlyDays := getRetentionSetting(app, "retention_hourly_days", defaultHourlyDays)
	dailyDays := getRetentionSetting(app, "retention_daily_days", defaultDailyDays)
	eventsDays := getRetentionSetting(app, "retention_events_days", defaultEventsDays)
	auditDays := getRetentionSetting(app, "retention_audit_logs_days", defaultAuditLogsDays)
	notifDays := getRetentionSetting(app, "retention_notification_logs_days", defaultNotifLogsDays)

	tables := []struct {
		name      string
		retention time.Duration
	}{
		{"server_metrics_raw", time.Duration(rawHours) * time.Hour},
		{"container_metrics_raw", time.Duration(rawHours) * time.Hour},
		{"host_app_metrics_raw", time.Duration(rawHours) * time.Hour},
		{"server_connectivity_raw", time.Duration(rawHours) * time.Hour},
		{"server_metrics_hourly", time.Duration(hourlyDays) * 24 * time.Hour},
		{"container_metrics_hourly", time.Duration(hourlyDays) * 24 * time.Hour},
		{"server_metrics_daily", time.Duration(dailyDays) * 24 * time.Hour},
		{"container_metrics_daily", time.Duration(dailyDays) * 24 * time.Hour},
		{"events", time.Duration(eventsDays) * 24 * time.Hour},
		{"audit_logs", time.Duration(auditDays) * 24 * time.Hour},
		{"notification_logs", time.Duration(notifDays) * 24 * time.Hour},
	}

	for _, t := range tables {
		cutoff := time.Now().UTC().Add(-t.retention).Format("2006-01-02 15:04:05.000Z")
		total := deleteInBatches(app, t.name, cutoff)
		if total > 0 {
			slog.Info("job: cleanup deleted records", "table", t.name, "count", total)
		}
	}

	// Clean up expired action tokens.
	actionTokensDeleted := cleanupActionTokens(app)
	if actionTokensDeleted > 0 {
		slog.Info("job: cleanup deleted action tokens", "count", actionTokensDeleted)
	}

	slog.Info("job: data cleanup finished")
}

// getRetentionSetting reads a retention setting from the settings collection.
func getRetentionSetting(app core.App, key string, defaultVal int) int {
	rec, err := app.FindFirstRecordByFilter("settings", "key = {:key}", dbx.Params{"key": key})
	if err != nil || rec == nil {
		return defaultVal
	}

	val := rec.GetString("value")
	if val == "" {
		return defaultVal
	}

	n, err := strconv.Atoi(val)
	if err != nil || n <= 0 {
		return defaultVal
	}

	return n
}

// deleteInBatches deletes records older than cutoff in batches. Returns total
// number of deleted records.
func deleteInBatches(app core.App, table, cutoff string) int64 {
	var total int64

	for {
		//nolint:gosec // table name is an internal constant, not user input
		sql := "DELETE FROM " + table +
			" WHERE id IN (SELECT id FROM " + table +
			" WHERE created < {:cutoff} LIMIT {:limit})"

		res, err := app.DB().NewQuery(sql).Bind(dbx.Params{
			"cutoff": cutoff,
			"limit":  deleteBatchSize,
		}).Execute()
		if err != nil {
			slog.Error("job: cleanup batch delete failed", "table", table, "error", err)
			break
		}

		affected, _ := res.RowsAffected()
		total += affected

		if affected < deleteBatchSize {
			break
		}
	}

	return total
}

// cleanupActionTokens removes expired and used action tokens.
func cleanupActionTokens(app core.App) int64 {
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000Z")
	old := time.Now().UTC().Add(-24 * time.Hour).Format("2006-01-02 15:04:05.000Z")

	sql := `DELETE FROM action_tokens WHERE id IN (
		SELECT id FROM action_tokens
		WHERE (used = true AND expires_at < {:now})
		   OR (expires_at < {:old})
		LIMIT {:limit}
	)`

	var total int64
	for {
		res, err := app.DB().NewQuery(sql).Bind(dbx.Params{
			"now":   now,
			"old":   old,
			"limit": deleteBatchSize,
		}).Execute()
		if err != nil {
			slog.Error("job: action token cleanup failed", "error", err)
			break
		}

		affected, _ := res.RowsAffected()
		total += affected
		if affected < deleteBatchSize {
			break
		}
	}

	return total
}
