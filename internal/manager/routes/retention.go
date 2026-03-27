package routes

import (
	"net/http"
	"strconv"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

var retentionKeys = []string{
	"retention_raw_hours",
	"retention_hourly_days",
	"retention_daily_days",
	"retention_events_days",
	"retention_audit_logs_days",
	"retention_notification_logs_days",
}

// RegisterRetentionRoutes adds retention configuration endpoints.
func RegisterRetentionRoutes(se *core.ServeEvent, app core.App, rbac *middleware.RBAC) {
	se.Router.GET("/api/dk/settings/retention", func(e *core.RequestEvent) error {
		return handleGetRetention(e, app)
	}).BindFunc(rbac.RequireCapability(constants.CapSettingsView))

	se.Router.PUT("/api/dk/settings/retention", func(e *core.RequestEvent) error {
		return handleUpdateRetention(e, app)
	}).BindFunc(rbac.RequireCapability(constants.CapSettingsManage))
}

func handleGetRetention(e *core.RequestEvent, app core.App) error {
	result := make(map[string]string)

	for _, key := range retentionKeys {
		rec, err := app.FindFirstRecordByFilter(constants.CollSettings, "key = {:key}", dbx.Params{"key": key})
		if err != nil || rec == nil {
			result[key] = ""
			continue
		}
		result[key] = rec.GetString("value")
	}

	return e.JSON(http.StatusOK, result)
}

func handleUpdateRetention(e *core.RequestEvent, app core.App) error {
	var body map[string]string
	if err := e.BindBody(&body); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}

	validKeys := make(map[string]bool)
	for _, k := range retentionKeys {
		validKeys[k] = true
	}

	for key, val := range body {
		if !validKeys[key] {
			continue
		}
		// Validate numeric value.
		if val != "" {
			n, err := strconv.Atoi(val)
			if err != nil || n <= 0 {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": key + " must be a positive number"})
			}
		}

		rec, err := app.FindFirstRecordByFilter(constants.CollSettings, "key = {:key}", dbx.Params{"key": key})
		if err != nil || rec == nil {
			// Create setting.
			col, colErr := app.FindCollectionByNameOrId(constants.CollSettings)
			if colErr != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": colErr.Error()})
			}
			rec = core.NewRecord(col)
			rec.Set("key", key)
		}
		rec.Set("value", val)
		if err := app.Save(rec); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	return e.JSON(http.StatusOK, map[string]string{"status": "updated"})
}
