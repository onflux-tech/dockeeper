package routes

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/manager/store"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// RegisterAlertRoutes adds alert rule management endpoints.
func RegisterAlertRoutes(se *core.ServeEvent, app core.App, alertStore *store.AlertStore, rbac *middleware.RBAC) {
	alerts := se.Router.Group("/api/dk/alerts")

	alerts.GET("", func(e *core.RequestEvent) error {
		return handleListAlertRules(e, app)
	}).BindFunc(rbac.RequireCapability(constants.CapAlertsView))

	alerts.POST("", func(e *core.RequestEvent) error {
		return handleCreateAlertRule(e, alertStore)
	}).BindFunc(rbac.RequireCapability(constants.CapAlertsManage))

	alerts.PUT("/{id}", func(e *core.RequestEvent) error {
		return handleUpdateAlertRule(e, alertStore)
	}).BindFunc(rbac.RequireCapability(constants.CapAlertsManage))

	alerts.DELETE("/{id}", func(e *core.RequestEvent) error {
		return handleDeleteAlertRule(e, alertStore)
	}).BindFunc(rbac.RequireCapability(constants.CapAlertsManage))

	alerts.POST("/{id}/toggle", func(e *core.RequestEvent) error {
		return handleToggleAlertRule(e, app, alertStore)
	}).BindFunc(rbac.RequireCapability(constants.CapAlertsManage))
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
		Name                  string   `json:"name"`
		RuleType              string   `json:"rule_type"`
		Metric                string   `json:"metric"`
		Operator              string   `json:"operator"`
		Threshold             float64  `json:"threshold"`
		Duration              int      `json:"duration"`
		TargetType            string   `json:"target_type"`
		TargetID              string   `json:"target_id"`
		EventType             string   `json:"event_type"`
		Scope                 string   `json:"scope"`
		ActiveDelay           int      `json:"active_delay"`
		ClearDelay            int      `json:"clear_delay"`
		NotificationCooldown  int      `json:"notification_cooldown"`
		SuppressDuringUpdates bool     `json:"suppress_during_updates"`
		Template              string   `json:"template"`
		Channels              []string `json:"channels"`
		Enabled               bool     `json:"enabled"`
	}
	if err := e.BindBody(&body); err != nil || body.Name == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
	}

	if body.RuleType == "" {
		body.RuleType = "metric"
	}

	if body.RuleType == "metric" {
		if body.Metric == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "metric is required for metric rules"})
		}
		validOps := map[string]bool{"gt": true, "lt": true, "gte": true, "lte": true, "eq": true}
		if !validOps[body.Operator] {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "operator must be gt, lt, gte, lte, or eq"})
		}
		if body.Duration <= 0 {
			body.Duration = 60
		}
	} else if body.RuleType == "event" {
		if body.EventType == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "event_type is required for event rules"})
		}
		if body.Scope == "" {
			body.Scope = "all"
		}
	}

	if body.NotificationCooldown <= 0 {
		body.NotificationCooldown = 300
	}

	id, err := alertStore.CreateRule(map[string]any{
		"name":                    body.Name,
		"rule_type":               body.RuleType,
		"metric":                  body.Metric,
		"operator":                body.Operator,
		"threshold":               body.Threshold,
		"duration":                body.Duration,
		"target_type":             body.TargetType,
		"target_id":               body.TargetID,
		"event_type":              body.EventType,
		"scope":                   body.Scope,
		"active_delay":            body.ActiveDelay,
		"clear_delay":             body.ClearDelay,
		"notification_cooldown":   body.NotificationCooldown,
		"suppress_during_updates": body.SuppressDuringUpdates,
		"template":                body.Template,
		"channels":                body.Channels,
		"enabled":                 body.Enabled,
		"state":                   "ok",
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
		"channels": true, "enabled": true, "rule_type": true, "event_type": true,
		"scope": true, "active_delay": true, "clear_delay": true,
		"notification_cooldown": true, "suppress_during_updates": true, "template": true,
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
