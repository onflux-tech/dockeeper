package routes

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/manager/services"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// RegisterBlackoutRoutes adds blackout window CRUD endpoints.
func RegisterBlackoutRoutes(se *core.ServeEvent, blackoutSvc *services.BlackoutService, rbac *middleware.RBAC) {
	g := se.Router.Group("/api/dk/blackout-windows")

	g.GET("", func(e *core.RequestEvent) error {
		windows, err := blackoutSvc.ListWindows()
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return e.JSON(http.StatusOK, windows)
	}).BindFunc(rbac.RequireCapability(constants.CapAlertsView))

	g.POST("", func(e *core.RequestEvent) error {
		var body struct {
			Name            string   `json:"name"`
			StartTime       string   `json:"start_time"`
			EndTime         string   `json:"end_time"`
			Recurring       bool     `json:"recurring"`
			RecurrenceRule  string   `json:"recurrence_rule"`
			DurationMinutes int      `json:"duration_minutes"`
			ServerIDs       []string `json:"server_ids"`
			Enabled         bool     `json:"enabled"`
		}
		if err := e.BindBody(&body); err != nil || body.Name == "" || body.StartTime == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "name and start_time are required"})
		}

		id, err := blackoutSvc.CreateWindow(map[string]any{
			"name":             body.Name,
			"start_time":       body.StartTime,
			"end_time":         body.EndTime,
			"recurring":        body.Recurring,
			"recurrence_rule":  body.RecurrenceRule,
			"duration_minutes": body.DurationMinutes,
			"server_ids":       body.ServerIDs,
			"enabled":          body.Enabled,
		})
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return e.JSON(http.StatusCreated, map[string]string{"id": id})
	}).BindFunc(rbac.RequireCapability(constants.CapAlertsManage))

	g.PUT("/{id}", func(e *core.RequestEvent) error {
		windowID := e.Request.PathValue("id")
		var body map[string]any
		if err := e.BindBody(&body); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
		}

		allowed := map[string]bool{
			"name": true, "start_time": true, "end_time": true,
			"recurring": true, "recurrence_rule": true, "duration_minutes": true,
			"server_ids": true, "enabled": true,
		}
		fields := make(map[string]any)
		for k, v := range body {
			if allowed[k] {
				fields[k] = v
			}
		}

		if err := blackoutSvc.UpdateWindow(windowID, fields); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return e.JSON(http.StatusOK, map[string]string{"status": "updated"})
	}).BindFunc(rbac.RequireCapability(constants.CapAlertsManage))

	g.DELETE("/{id}", func(e *core.RequestEvent) error {
		windowID := e.Request.PathValue("id")
		if err := blackoutSvc.DeleteWindow(windowID); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return e.JSON(http.StatusOK, map[string]string{"status": "deleted"})
	}).BindFunc(rbac.RequireCapability(constants.CapAlertsManage))
}
