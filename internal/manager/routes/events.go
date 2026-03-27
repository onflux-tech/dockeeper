package routes

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// RegisterEventRoutes registers the events API endpoints.
func RegisterEventRoutes(se *core.ServeEvent, app core.App, rbac *middleware.RBAC) {
	group := se.Router.Group("/api/dk/events")

	group.GET("", handleListEvents(app)).BindFunc(rbac.RequireCapability(constants.CapEventsView))
}

func handleListEvents(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		q := e.Request.URL.Query()

		// Build filter.
		var filters []string
		params := map[string]any{}

		if t := q.Get("type"); t != "" {
			filters = append(filters, "event_type = {:event_type}")
			params["event_type"] = t
		}
		if sev := q.Get("severity"); sev != "" {
			filters = append(filters, "severity = {:severity}")
			params["severity"] = sev
		}
		if sid := q.Get("server_id"); sid != "" {
			filters = append(filters, "server_id = {:server_id}")
			params["server_id"] = sid
		}
		if from := q.Get("from"); from != "" {
			if t, err := time.Parse(time.RFC3339, from); err == nil {
				filters = append(filters, "timestamp >= {:from}")
				params["from"] = t.UTC()
			}
		}
		if to := q.Get("to"); to != "" {
			if t, err := time.Parse(time.RFC3339, to); err == nil {
				filters = append(filters, "timestamp <= {:to}")
				params["to"] = t.UTC()
			}
		}

		filter := ""
		if len(filters) > 0 {
			filter = strings.Join(filters, " && ")
		}

		limit := 50
		if l := q.Get("limit"); l != "" {
			if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 200 {
				limit = v
			}
		}
		offset := 0
		if o := q.Get("offset"); o != "" {
			if v, err := strconv.Atoi(o); err == nil && v >= 0 {
				offset = v
			}
		}

		var paramSlice []dbx.Params
		if len(params) > 0 {
			paramSlice = append(paramSlice, dbx.Params(params))
		}

		records, err := app.FindRecordsByFilter(
			constants.CollEvents,
			filter,
			"-timestamp",
			limit, offset,
			paramSlice...,
		)
		if err != nil {
			return e.JSON(http.StatusOK, []any{})
		}

		// Serialize records.
		result := make([]map[string]any, 0, len(records))
		for _, r := range records {
			result = append(result, map[string]any{
				"id":             r.Id,
				"event_type":     r.GetString("event_type"),
				"severity":       r.GetString("severity"),
				"server_id":      r.GetString("server_id"),
				"server_name":    r.GetString("server_name"),
				"container_id":   r.GetString("container_id"),
				"container_name": r.GetString("container_name"),
				"details":        r.Get("details"),
				"timestamp":      r.GetString("timestamp"),
				"created":        r.GetString("created"),
			})
		}

		return e.JSON(http.StatusOK, result)
	}
}
