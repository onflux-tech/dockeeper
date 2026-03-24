package routes

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"

	"github.com/jkdesenvolvedor/dockeeper/internal/manager/middleware"
	"github.com/jkdesenvolvedor/dockeeper/internal/manager/store"
	"github.com/jkdesenvolvedor/dockeeper/internal/shared/constants"
)

// RegisterNotificationRoutes adds notification channel and log endpoints.
func RegisterNotificationRoutes(se *core.ServeEvent, app core.App, alertStore *store.AlertStore) {
	notif := se.Router.Group("/api/dk/notifications")

	// ── Channels ─────────────────────────────────────
	notif.GET("/channels", func(e *core.RequestEvent) error {
		return handleListChannels(e, app)
	}).BindFunc(middleware.RequireSuperuser)

	notif.POST("/channels", func(e *core.RequestEvent) error {
		return handleCreateChannel(e, alertStore)
	}).BindFunc(middleware.RequireSuperuser)

	notif.PUT("/channels/{id}", func(e *core.RequestEvent) error {
		return handleUpdateChannel(e, alertStore)
	}).BindFunc(middleware.RequireSuperuser)

	notif.DELETE("/channels/{id}", func(e *core.RequestEvent) error {
		return handleDeleteChannel(e, alertStore)
	}).BindFunc(middleware.RequireSuperuser)

	// ── Logs ─────────────────────────────────────────
	notif.GET("/logs", func(e *core.RequestEvent) error {
		return handleListNotificationLogs(e, app)
	}).BindFunc(middleware.RequireSuperuser)
}

func handleListChannels(e *core.RequestEvent, app core.App) error {
	records, err := app.FindRecordsByFilter(
		constants.CollNotificationChannels,
		"",
		"-created",
		200, 0,
	)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return e.JSON(http.StatusOK, records)
}

func handleCreateChannel(e *core.RequestEvent, alertStore *store.AlertStore) error {
	var body struct {
		Name    string         `json:"name"`
		Type    string         `json:"type"`
		Config  map[string]any `json:"config"`
		Enabled bool           `json:"enabled"`
	}
	if err := e.BindBody(&body); err != nil || body.Name == "" || body.Type == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "name and type are required"})
	}

	validTypes := map[string]bool{"whatsapp": true, "telegram": true, "webhook": true}
	if !validTypes[body.Type] {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "type must be whatsapp, telegram, or webhook"})
	}

	id, err := alertStore.CreateChannel(map[string]any{
		"name":    body.Name,
		"type":    body.Type,
		"config":  body.Config,
		"enabled": body.Enabled,
	})
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return e.JSON(http.StatusCreated, map[string]string{"id": id})
}

func handleUpdateChannel(e *core.RequestEvent, alertStore *store.AlertStore) error {
	channelID := e.Request.PathValue("id")

	var body map[string]any
	if err := e.BindBody(&body); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}

	allowed := map[string]bool{"name": true, "type": true, "config": true, "enabled": true}
	fields := make(map[string]any)
	for k, v := range body {
		if allowed[k] {
			fields[k] = v
		}
	}

	if err := alertStore.UpdateChannel(channelID, fields); err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return e.JSON(http.StatusOK, map[string]string{"status": "updated"})
}

func handleDeleteChannel(e *core.RequestEvent, alertStore *store.AlertStore) error {
	channelID := e.Request.PathValue("id")

	if err := alertStore.DeleteChannel(channelID); err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return e.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

func handleListNotificationLogs(e *core.RequestEvent, app core.App) error {
	alertRuleID := e.Request.URL.Query().Get("alert_rule_id")
	channelID := e.Request.URL.Query().Get("channel_id")

	filter := ""
	if alertRuleID != "" {
		filter = "alert_rule = '" + alertRuleID + "'"
	} else if channelID != "" {
		filter = "channel = '" + channelID + "'"
	}

	records, err := app.FindRecordsByFilter(
		constants.CollNotificationLogs,
		filter,
		"-sent_at",
		200, 0,
	)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return e.JSON(http.StatusOK, records)
}
