package jobs

import (
	"log/slog"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const deleteBatchSize = 1000

// Retention periods.
const (
	rawRetention    = 48 * time.Hour
	hourlyRetention = 30 * 24 * time.Hour  // 30 days
	dailyRetention  = 365 * 24 * time.Hour // 365 days
)

// CleanupOldMetrics removes expired metrics data according to retention policies:
// - raw: > 48 hours
// - hourly: > 30 days
// - daily: > 365 days
func CleanupOldMetrics(app core.App) {
	slog.Info("job: metrics cleanup started")

	tables := []struct {
		name      string
		retention time.Duration
	}{
		{"server_metrics_raw", rawRetention},
		{"container_metrics_raw", rawRetention},
		{"server_metrics_hourly", hourlyRetention},
		{"container_metrics_hourly", hourlyRetention},
		{"server_metrics_daily", dailyRetention},
		{"container_metrics_daily", dailyRetention},
	}

	for _, t := range tables {
		cutoff := time.Now().UTC().Add(-t.retention).Format("2006-01-02 15:04:05.000Z")
		total := deleteInBatches(app, t.name, cutoff)
		if total > 0 {
			slog.Info("job: cleanup deleted records", "table", t.name, "count", total)
		}
	}

	slog.Info("job: metrics cleanup finished")
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
