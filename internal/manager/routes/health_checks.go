package routes

import (
	"net/http"
	"strconv"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/manager/services"
	"github.com/onflux-tech/dockeeper/internal/manager/store"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// RegisterHealthCheckRoutes adds CRUD and trigger endpoints for health checks.
func RegisterHealthCheckRoutes(se *core.ServeEvent, app core.App, healthStore *store.HealthStore, scheduler *services.HealthScheduler, rbac *middleware.RBAC) {
	g := se.Router.Group("/api/dk/health-checks")

	// GET /api/dk/health-checks — list all checks.
	g.GET("", func(e *core.RequestEvent) error {
		serverID := e.Request.URL.Query().Get("server_id")
		records, err := healthStore.GetAllChecks(serverID)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return e.JSON(http.StatusOK, records)
	}).BindFunc(rbac.RequireCapability(constants.CapHealthView))

	// POST /api/dk/health-checks — create a new check.
	g.POST("", func(e *core.RequestEvent) error {
		var body struct {
			Name                 string `json:"name"`
			Container            string `json:"container"`
			Server               string `json:"server"`
			URLPath              string `json:"url_path"`
			Port                 int    `json:"port"`
			ExpectedStatus       int    `json:"expected_status"`
			IntervalSeconds      int    `json:"interval_seconds"`
			TimeoutSeconds       int    `json:"timeout_seconds"`
			FailureThreshold     int    `json:"failure_threshold"`
			SuccessThreshold     int    `json:"success_threshold"`
			AutoRestart          bool   `json:"auto_restart"`
			MaxRestarts          int    `json:"max_restarts"`
			RestartWindowMinutes int    `json:"restart_window_minutes"`
			Enabled              bool   `json:"enabled"`
		}
		if err := e.BindBody(&body); err != nil || body.Name == "" || body.Container == "" || body.Server == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "name, container, and server are required"})
		}
		if body.Port <= 0 {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "port is required"})
		}

		fields := map[string]any{
			"name":                   body.Name,
			"container":              body.Container,
			"server":                 body.Server,
			"url_path":               body.URLPath,
			"port":                   body.Port,
			"expected_status":        body.ExpectedStatus,
			"interval_seconds":       body.IntervalSeconds,
			"timeout_seconds":        body.TimeoutSeconds,
			"failure_threshold":      body.FailureThreshold,
			"success_threshold":      body.SuccessThreshold,
			"auto_restart":           body.AutoRestart,
			"max_restarts":           body.MaxRestarts,
			"restart_window_minutes": body.RestartWindowMinutes,
			"enabled":                body.Enabled,
		}

		id, err := healthStore.CreateCheck(fields)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		if body.Enabled {
			scheduler.AddCheck(id)
		}

		return e.JSON(http.StatusCreated, map[string]string{"id": id})
	}).BindFunc(rbac.RequireCapability(constants.CapHealthManage))

	// PUT /api/dk/health-checks/{id} — update a check.
	g.PUT("/{id}", func(e *core.RequestEvent) error {
		checkID := e.Request.PathValue("id")
		var body map[string]any
		if err := e.BindBody(&body); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
		}

		allowed := map[string]bool{
			"name": true, "url_path": true, "port": true,
			"expected_status": true, "interval_seconds": true, "timeout_seconds": true,
			"failure_threshold": true, "success_threshold": true,
			"auto_restart": true, "max_restarts": true, "restart_window_minutes": true,
			"enabled": true,
		}
		fields := make(map[string]any)
		for k, v := range body {
			if allowed[k] {
				fields[k] = v
			}
		}

		if err := healthStore.UpdateCheck(checkID, fields); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		// Reload scheduler to pick up changes.
		scheduler.RemoveCheck(checkID)
		if enabled, ok := fields["enabled"].(bool); ok && enabled {
			scheduler.AddCheck(checkID)
		} else if !ok {
			// If enabled wasn't changed, check current state.
			rec, err := healthStore.GetCheckByID(checkID)
			if err == nil && rec.GetBool("enabled") {
				scheduler.AddCheck(checkID)
			}
		}

		return e.JSON(http.StatusOK, map[string]string{"status": "updated"})
	}).BindFunc(rbac.RequireCapability(constants.CapHealthManage))

	// DELETE /api/dk/health-checks/{id} — delete a check.
	g.DELETE("/{id}", func(e *core.RequestEvent) error {
		checkID := e.Request.PathValue("id")
		scheduler.RemoveCheck(checkID)
		if err := healthStore.DeleteCheck(checkID); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return e.JSON(http.StatusOK, map[string]string{"status": "deleted"})
	}).BindFunc(rbac.RequireCapability(constants.CapHealthManage))

	// GET /api/dk/health-checks/{id}/logs — get check logs.
	g.GET("/{id}/logs", func(e *core.RequestEvent) error {
		checkID := e.Request.PathValue("id")
		page, _ := strconv.Atoi(e.Request.URL.Query().Get("page"))
		perPage, _ := strconv.Atoi(e.Request.URL.Query().Get("per_page"))
		if page < 1 {
			page = 1
		}
		if perPage < 1 {
			perPage = 50
		}

		records, total, err := healthStore.GetCheckLogs(checkID, page, perPage)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return e.JSON(http.StatusOK, map[string]any{
			"items":       records,
			"total_items": total,
			"page":        page,
			"per_page":    perPage,
		})
	}).BindFunc(rbac.RequireCapability(constants.CapHealthView))

	// POST /api/dk/health-checks/{id}/trigger — trigger immediate check.
	g.POST("/{id}/trigger", func(e *core.RequestEvent) error {
		checkID := e.Request.PathValue("id")
		scheduler.TriggerCheck(checkID)
		return e.JSON(http.StatusOK, map[string]string{"status": "triggered"})
	}).BindFunc(rbac.RequireCapability(constants.CapHealthManage))
}
