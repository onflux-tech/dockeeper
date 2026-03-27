package routes

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// RegisterTraefikRoutes adds Traefik route query endpoints.
func RegisterTraefikRoutes(se *core.ServeEvent, app core.App, rbac *middleware.RBAC) {
	traefik := se.Router.Group("/api/dk/traefik")

	traefik.GET("/routes", func(e *core.RequestEvent) error {
		return handleListTraefikRoutes(e, app)
	}).BindFunc(rbac.RequireCapability(constants.CapTraefikView))
}

func handleListTraefikRoutes(e *core.RequestEvent, app core.App) error {
	q := e.Request.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(q.Get("perPage"))
	if perPage < 1 {
		perPage = 25
	}
	if perPage > 200 {
		perPage = 200
	}

	sort := q.Get("sort")
	if sort == "" {
		sort = "name"
	}

	// Build filter from query params.
	var parts []string
	if sid := q.Get("server_id"); sid != "" {
		parts = append(parts, "server = '"+sid+"'")
	}
	if search := strings.TrimSpace(q.Get("search")); search != "" {
		escaped := strings.ReplaceAll(search, "'", "\\'")
		parts = append(parts, "(name ~ '"+escaped+"' || rule ~ '"+escaped+"' || service ~ '"+escaped+"')")
	}
	filter := strings.Join(parts, " && ")

	// Count total matching records.
	all, err := app.FindRecordsByFilter(
		constants.CollTraefikRoutes,
		filter,
		sort,
		0, 0,
	)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	totalItems := len(all)
	totalPages := (totalItems + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}

	offset := (page - 1) * perPage
	records, err := app.FindRecordsByFilter(
		constants.CollTraefikRoutes,
		filter,
		sort,
		perPage,
		offset,
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
