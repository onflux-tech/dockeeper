package routes

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"

	"github.com/jkdesenvolvedor/dockeeper/internal/manager/middleware"
	"github.com/jkdesenvolvedor/dockeeper/internal/manager/store"
	"github.com/jkdesenvolvedor/dockeeper/internal/shared/constants"
)

// RegisterAlertRoutes adds alert rule management endpoints.
func RegisterAlertRoutes(se *core.ServeEvent, app core.App, alertStore *store.AlertStore) {
	alerts := se.Router.Group("/api/dk/alerts")

	alerts.GET("", func(e *core.RequestEvent) error {
		return handleListAlertRules(e, app)
	}).BindFunc(middleware.RequireSuperuser)

	alerts.POST("", func(e *core.RequestEvent) error {
		return handleCreateAlertRule(e, alertStore)
	}).BindFunc(middleware.RequireSuperuser)

	alerts.PUT("/{id}", func(e *core.RequestEvent) error {
		return handleUpdateAlertRule(e, alertStore)
	}).BindFunc(middleware.RequireSuperuser)

	alerts.DELETE("/{id}", func(e *core.RequestEvent) error {
		return handleDeleteAlertRule(e, alertStore)
	}).BindFunc(middleware.RequireSuperuser)

	alerts.POST("/{id}/toggle", func(e *core.RequestEvent) error {
		return handleToggleAlertRule(e, app, alertStore)
	}).BindFunc(middleware.RequireSuperuser)
}

func handleListAlertRules(e *core.RequestEvent, app core.App) error {
	serverID := e.Request.URL.Query().Get("server_id")
	filter := ""
	if serverID != "" {
		filter = "target_type = 'server' && target_id = '" + serverID + "'"
	}

	records, err := app.FindRecordsByFilter(
		constants.CollAlertRules,
		filter,
		"-created",
		200, 0,
	)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return e.JSON(http.StatusOK, records)
}

func handleCreateAlertRule(e *core.RequestEvent, alertStore *store.AlertStore) error {
	var body struct {
		Name       string   `json:"name"`
		Metric     string   `json:"metric"`
		Operator   string   `json:"operator"`
		Threshold  float64  `json:"threshold"`
		Duration   int      `json:"duration"`
		TargetType string   `json:"target_type"`
		TargetID   string   `json:"target_id"`
		Channels   []string `json:"channels"`
		Enabled    bool     `json:"enabled"`
	}
	if err := e.BindBody(&body); err != nil || body.Name == "" || body.Metric == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "name, metric, operator, threshold, target_type, and target_id are required"})
	}

	validOps := map[string]bool{"gt": true, "lt": true, "gte": true, "lte": true, "eq": true}
	if !validOps[body.Operator] {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "operator must be gt, lt, gte, lte, or eq"})
	}

	if body.Duration <= 0 {
		body.Duration = 60
	}

	id, err := alertStore.CreateRule(map[string]any{
		"name":        body.Name,
		"metric":      body.Metric,
		"operator":    body.Operator,
		"threshold":   body.Threshold,
		"duration":    body.Duration,
		"target_type": body.TargetType,
		"target_id":   body.TargetID,
		"channels":    body.Channels,
		"enabled":     body.Enabled,
		"state":       "ok",
	})
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return e.JSON(http.StatusCreated, map[string]string{"id": id})
}

func handleUpdateAlertRule(e *core.RequestEvent, alertStore *store.AlertStore) error {
	ruleID := e.Request.PathValue("id")

	var body map[string]any
	if err := e.BindBody(&body); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}

	allowed := map[string]bool{
		"name": true, "metric": true, "operator": true, "threshold": true,
		"duration": true, "target_type": true, "target_id": true,
		"channels": true, "enabled": true,
	}
	fields := make(map[string]any)
	for k, v := range body {
		if allowed[k] {
			fields[k] = v
		}
	}

	if err := alertStore.UpdateRule(ruleID, fields); err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return e.JSON(http.StatusOK, map[string]string{"status": "updated"})
}

func handleDeleteAlertRule(e *core.RequestEvent, alertStore *store.AlertStore) error {
	ruleID := e.Request.PathValue("id")

	if err := alertStore.DeleteRule(ruleID); err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return e.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

func handleToggleAlertRule(e *core.RequestEvent, app core.App, alertStore *store.AlertStore) error {
	ruleID := e.Request.PathValue("id")

	rec, err := app.FindRecordById(constants.CollAlertRules, ruleID)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{"error": "rule not found"})
	}

	newEnabled := !rec.GetBool("enabled")
	if err := alertStore.UpdateRule(ruleID, map[string]any{"enabled": newEnabled}); err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return e.JSON(http.StatusOK, map[string]any{"enabled": newEnabled})
}
