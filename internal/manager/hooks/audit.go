package hooks

import (
	"log/slog"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/store"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// auditedCollections maps collection names to human-friendly target types.
var auditedCollections = map[string]string{
	constants.CollAlertRules:           "alert_rule",
	constants.CollNotificationChannels: "notification_channel",
	constants.CollBackupPolicies:       "backup_policy",
	constants.CollSettings:             "setting",
	constants.CollComposeStacks:        "compose_stack",
}

// RegisterAuditHooks registers PB record lifecycle hooks that automatically
// create audit log entries for collections not manually audited by services.
func RegisterAuditHooks(app core.App, auditStore *store.AuditStore) {
	app.OnRecordAfterCreateSuccess().BindFunc(func(e *core.RecordEvent) error {
		if targetType, ok := auditedCollections[e.Record.Collection().Name]; ok {
			auditStore.Log(
				targetType+".create",
				extractUser(e),
				targetType,
				e.Record.Id,
				"",
				map[string]any{"collection": e.Record.Collection().Name},
			)
		}
		return e.Next()
	})

	app.OnRecordAfterUpdateSuccess().BindFunc(func(e *core.RecordEvent) error {
		if targetType, ok := auditedCollections[e.Record.Collection().Name]; ok {
			auditStore.Log(
				targetType+".update",
				extractUser(e),
				targetType,
				e.Record.Id,
				"",
				map[string]any{"collection": e.Record.Collection().Name},
			)
		}
		return e.Next()
	})

	app.OnRecordAfterDeleteSuccess().BindFunc(func(e *core.RecordEvent) error {
		if targetType, ok := auditedCollections[e.Record.Collection().Name]; ok {
			auditStore.Log(
				targetType+".delete",
				extractUser(e),
				targetType,
				e.Record.Id,
				"",
				map[string]any{"collection": e.Record.Collection().Name},
			)
		}
		return e.Next()
	})

	slog.Info("audit hooks registered", "collections", len(auditedCollections))
}

// RegisterAuditCleanup registers a daily cron to delete audit logs older than 90 days.
func RegisterAuditCleanup(app core.App) {
	if err := app.Cron().Add("audit_logs_cleanup", "0 3 * * *", func() {
		cleanupOldAuditLogs(app)
	}); err != nil {
		slog.Error("failed to register audit cleanup cron", "error", err)
	}
	slog.Info("audit cleanup cron registered", "schedule", "daily@03:00", "retention", "90d")
}

func cleanupOldAuditLogs(app core.App) {
	cutoff := time.Now().AddDate(0, 0, -90).UTC().Format("2006-01-02 15:04:05.000Z")

	records, err := app.FindRecordsByFilter(
		constants.CollAuditLogs,
		"timestamp < {:cutoff}",
		"",
		0, 0,
		map[string]any{"cutoff": cutoff},
	)
	if err != nil {
		slog.Error("audit cleanup query failed", "error", err)
		return
	}

	deleted := 0
	for _, r := range records {
		if err := app.Delete(r); err != nil {
			slog.Error("failed to delete audit log", "id", r.Id, "error", err)
			continue
		}
		deleted++
	}

	if deleted > 0 {
		slog.Info("audit logs cleanup completed", "deleted", deleted)
	}
}

func extractUser(e *core.RecordEvent) string {
	if e.Record == nil {
		return ""
	}
	// The RecordEvent itself doesn't carry auth info directly.
	// For PB hook-based auditing, we log the record's collection-level action.
	return ""
}
