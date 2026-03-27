package events

import (
	"log/slog"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// DBHandler persists every event to the PocketBase events collection.
type DBHandler struct {
	app core.App
}

// NewDBHandler creates a handler that writes events to the database.
func NewDBHandler(app core.App) *DBHandler {
	return &DBHandler{app: app}
}

// Handle persists the event as a PB record.
func (h *DBHandler) Handle(e Event) {
	col, err := h.app.FindCachedCollectionByNameOrId(constants.CollEvents)
	if err != nil {
		slog.Error("events collection not found", "error", err)
		return
	}

	rec := core.NewRecord(col)
	rec.Set("event_type", string(e.Type))
	rec.Set("severity", string(e.Severity))
	rec.Set("server_id", e.ServerID)
	rec.Set("server_name", e.ServerName)
	rec.Set("container_id", e.ContainerID)
	rec.Set("container_name", e.ContainerName)
	rec.Set("details", e.Details)
	rec.Set("timestamp", e.Timestamp.UTC())

	if err := h.app.Save(rec); err != nil {
		slog.Error("failed to persist event", "type", string(e.Type), "error", err)
	}
}
