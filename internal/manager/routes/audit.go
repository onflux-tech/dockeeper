package routes

import (
	"net/http"
	"strconv"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// RegisterAuditRoutes adds audit log query endpoints.
func RegisterAuditRoutes(se *core.ServeEvent, app core.App) {
	audit := se.Router.Group("/api/dk/audit-logs")

	audit.GET("", func(e *core.RequestEvent) error {
		return handleListAuditLogs(e, app)
	}).BindFunc(middleware.RequireSuperuser)
}

func handleListAuditLogs(e *core.RequestEvent, app core.App) error {
	q := e.Request.URL.Query()

	// Pagination.
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(q.Get("perPage"))
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}

	// Build filter.
	filter := ""
	params := map[string]any{}

	if action := q.Get("action"); action != "" {
		filter = appendFilter(filter, "action ~ {:action}")
		params["action"] = action
	}
	if targetType := q.Get("target_type"); targetType != "" {
		filter = appendFilter(filter, "target_type = {:target_type}")
		params["target_type"] = targetType
	}
	if user := q.Get("user"); user != "" {
		filter = appendFilter(filter, "user = {:user}")
		params["user"] = user
	}
	if from := q.Get("from"); from != "" {
		filter = appendFilter(filter, "timestamp >= {:from}")
		params["from"] = from
	}
	if to := q.Get("to"); to != "" {
		filter = appendFilter(filter, "timestamp <= {:to}")
		params["to"] = to
	}
	if search := q.Get("search"); search != "" {
		filter = appendFilter(filter, "(action ~ {:search} || target_type ~ {:search} || target_id ~ {:search})")
		params["search"] = search
	}

	// Total count for pagination.
	allRecords, err := app.FindRecordsByFilter(
		constants.CollAuditLogs,
		filter,
		"",
		0, 0,
		params,
	)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	totalItems := len(allRecords)
	totalPages := (totalItems + perPage - 1) / perPage

	// Fetch page.
	offset := (page - 1) * perPage
	records, err := app.FindRecordsByFilter(
		constants.CollAuditLogs,
		filter,
		"-timestamp",
		perPage, offset,
		params,
	)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return e.JSON(http.StatusOK, map[string]any{
		"page":       page,
		"perPage":    perPage,
		"totalItems": totalItems,
		"totalPages": totalPages,
		"items":      records,
	})
}

func appendFilter(existing, clause string) string {
	if existing == "" {
		return clause
	}
	return existing + " && " + clause
}
