package routes

import (
	"encoding/json"
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// Widget represents a single dashboard widget layout entry.
type Widget struct {
	ID string `json:"id"`
	X  int    `json:"x"`
	Y  int    `json:"y"`
	W  int    `json:"w"`
	H  int    `json:"h"`
}

// DashboardLayout holds a user's widget arrangement.
type DashboardLayout struct {
	Widgets []Widget `json:"widgets"`
}

// DefaultLayout is returned when a user has no saved layout.
var DefaultLayout = DashboardLayout{
	Widgets: []Widget{
		{ID: "health-status", X: 0, Y: 0, W: 4, H: 5},
		{ID: "recent-events", X: 4, Y: 0, W: 4, H: 5},
		{ID: "alerts-summary", X: 8, Y: 0, W: 4, H: 5},
	},
}

// validWidgetIDs lists all known widget IDs.
var validWidgetIDs = map[string]bool{
	"stats-overview": true,
	"server-cards":   true,
	"recent-events":  true,
	"health-status":  true,
	"alerts-summary": true,
	"cpu-chart":      true,
	"memory-chart":   true,
	"quick-actions":  true,
}

// RegisterDashboardRoutes adds layout persistence endpoints.
func RegisterDashboardRoutes(se *core.ServeEvent, app core.App, rbac *middleware.RBAC) {
	g := se.Router.Group("/api/dk/dashboard")

	// GET /api/dk/dashboard/layout — get user's layout or default.
	g.GET("/layout", func(e *core.RequestEvent) error {
		userID := getUserID(e)
		if userID == "" {
			return e.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		}

		key := "dashboard_layout:" + userID
		rec, err := app.FindFirstRecordByFilter(constants.CollSettings, "key = {:key}", dbx.Params{"key": key})
		if err != nil || rec == nil {
			return e.JSON(http.StatusOK, DefaultLayout)
		}

		raw := rec.GetString("value")
		var layout DashboardLayout
		if err := json.Unmarshal([]byte(raw), &layout); err != nil {
			return e.JSON(http.StatusOK, DefaultLayout)
		}

		return e.JSON(http.StatusOK, layout)
	}).BindFunc(rbac.RequireCapability(constants.CapContainersView))

	// PUT /api/dk/dashboard/layout — save user's layout.
	g.PUT("/layout", func(e *core.RequestEvent) error {
		userID := getUserID(e)
		if userID == "" {
			return e.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		}

		var layout DashboardLayout
		if err := e.BindBody(&layout); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid layout"})
		}

		// Validate widgets.
		for _, w := range layout.Widgets {
			if !validWidgetIDs[w.ID] {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "unknown widget: " + w.ID})
			}
			if w.W < 1 || w.W > 12 || w.H < 1 || w.H > 12 {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "widget dimensions out of range"})
			}
		}

		key := "dashboard_layout:" + userID
		data, _ := json.Marshal(layout)

		rec, err := app.FindFirstRecordByFilter(constants.CollSettings, "key = {:key}", dbx.Params{"key": key})
		if err != nil || rec == nil {
			col, err := app.FindCachedCollectionByNameOrId(constants.CollSettings)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}
			rec = core.NewRecord(col)
			rec.Set("key", key)
		}
		rec.Set("value", string(data))

		if err := app.Save(rec); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return e.JSON(http.StatusOK, map[string]string{"status": "saved"})
	}).BindFunc(rbac.RequireCapability(constants.CapContainersView))

	// DELETE /api/dk/dashboard/layout — reset to default.
	g.DELETE("/layout", func(e *core.RequestEvent) error {
		userID := getUserID(e)
		if userID == "" {
			return e.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		}

		key := "dashboard_layout:" + userID
		rec, err := app.FindFirstRecordByFilter(constants.CollSettings, "key = {:key}", dbx.Params{"key": key})
		if err == nil && rec != nil {
			_ = app.Delete(rec)
		}

		return e.JSON(http.StatusOK, DefaultLayout)
	}).BindFunc(rbac.RequireCapability(constants.CapContainersView))
}

// getUserID extracts the authenticated user ID from the request.
func getUserID(e *core.RequestEvent) string {
	if e.Auth != nil {
		return e.Auth.Id
	}
	return ""
}
