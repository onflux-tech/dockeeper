package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/robfig/cron/v3"

	"github.com/onflux-tech/dockeeper/internal/manager/services"
	"github.com/onflux-tech/dockeeper/internal/manager/store"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// RegisterBackupJobs registers the backup scheduler and retention cleanup jobs.
func RegisterBackupJobs(app core.App, backupSvc *services.BackupService, backupStore *store.BackupStore) {
	register := func(name, cronExpr string, fn func()) {
		if err := app.Cron().Add(name, cronExpr, fn); err != nil {
			slog.Error("failed to register backup job", "name", name, "error", err)
		}
	}

	// Check enabled policies every 5 minutes and trigger due backups.
	register("backup_scheduler", "*/5 * * * *", func() {
		runScheduledBackups(app, backupSvc, backupStore)
	})

	// Retention cleanup daily at 04:00.
	register("backup_retention_cleanup", "0 4 * * *", func() {
		backupSvc.RunRetentionCleanup()
	})

	slog.Info("backup jobs registered", "scheduler", "*/5min", "retention", "04:00")
}

func runScheduledBackups(app core.App, backupSvc *services.BackupService, backupStore *store.BackupStore) {
	policies, err := backupStore.GetEnabledPolicies()
	if err != nil {
		slog.Error("backup scheduler: failed to list policies", "error", err)
		return
	}

	now := time.Now().UTC()
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

	for _, policy := range policies {
		schedule := policy.GetString("schedule")
		if schedule == "" {
			continue
		}

		sched, parseErr := parser.Parse(schedule)
		if parseErr != nil {
			slog.Warn("backup scheduler: invalid cron expression", "policy", policy.Id, "schedule", schedule, "error", parseErr)
			continue
		}

		// Check if this policy should have run since last backup.
		lastBackup, _ := backupStore.GetLastBackupForPolicy(policy.Id)
		var lastRun time.Time
		if lastBackup != nil {
			lastRun = lastBackup.GetDateTime("started_at").Time()
		}

		// If no previous backup, or the next scheduled time after the last run is before now.
		nextRun := sched.Next(lastRun)
		if nextRun.Before(now) || nextRun.Equal(now) {
			// Skip if a backup is already running.
			running, _ := app.FindRecordsByFilter(
				constants.CollBackupRecords,
				"policy = '"+policy.Id+"' && status = 'running'",
				"", 1, 0,
			)
			if len(running) > 0 {
				continue
			}

			slog.Info("backup scheduler: triggering backup", "policy", policy.Id, "schedule", schedule)
			if _, trigErr := backupSvc.TriggerBackup(context.Background(), policy.Id, "system", "scheduler"); trigErr != nil {
				slog.Error("backup scheduler: trigger failed", "policy", policy.Id, "error", trigErr)
			}
		}
	}
}
