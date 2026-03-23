package store

import (
	"log/slog"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/jkdesenvolvedor/dockeeper/internal/shared/constants"
)

// AuditStore persists entries to the audit_logs collection.
type AuditStore struct {
	app core.App
}

// NewAuditStore creates an AuditStore.
func NewAuditStore(app core.App) *AuditStore {
	return &AuditStore{app: app}
}

// Log creates a new audit log entry.
func (s *AuditStore) Log(action, userID, targetType, targetID, ip string, details map[string]any) {
	col, err := s.app.FindCachedCollectionByNameOrId(constants.CollAuditLogs)
	if err != nil {
		slog.Error("audit_logs collection not found", "error", err)
		return
	}

	record := core.NewRecord(col)
	record.Set("action", action)
	record.Set("user", userID)
	record.Set("target_type", targetType)
	record.Set("target_id", targetID)
	record.Set("ip_address", ip)
	record.Set("details", details)
	record.Set("timestamp", time.Now().UTC().Format("2006-01-02 15:04:05.000Z"))

	if err := s.app.Save(record); err != nil {
		slog.Error("failed to save audit log", "action", action, "error", err)
	}
}
