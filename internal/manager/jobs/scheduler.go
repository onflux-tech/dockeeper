package jobs

import (
	"log/slog"

	"github.com/pocketbase/pocketbase/core"
)

// RegisterJobs registers all periodic background jobs (aggregation + cleanup)
// on the PocketBase scheduler.
func RegisterJobs(app core.App) {
	register := func(name, cron string, fn func()) {
		if err := app.Cron().Add(name, cron, fn); err != nil {
			slog.Error("failed to register job", "name", name, "error", err)
		}
	}

	// Server metrics: hourly aggregation at :05 every hour.
	register("metrics_server_hourly", "5 * * * *", func() {
		AggregateServerMetricsHourly(app)
	})

	// Container metrics: hourly aggregation at :10 every hour.
	register("metrics_container_hourly", "10 * * * *", func() {
		AggregateContainerMetricsHourly(app)
	})

	// Server metrics: daily aggregation at 00:15.
	register("metrics_server_daily", "15 0 * * *", func() {
		AggregateServerMetricsDaily(app)
	})

	// Container metrics: daily aggregation at 00:20.
	register("metrics_container_daily", "20 0 * * *", func() {
		AggregateContainerMetricsDaily(app)
	})

	// Cleanup expired metrics at 03:30.
	register("metrics_cleanup", "30 3 * * *", func() {
		CleanupOldMetrics(app)
	})

	slog.Info("metrics jobs registered",
		"hourly", "server@:05 container@:10",
		"daily", "server@00:15 container@00:20",
		"cleanup", "03:30",
	)
}
