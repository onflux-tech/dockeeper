package routes

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

var validProxyKinds = map[string]bool{
	"traefik": true,
	"nginx":   true,
	"caddy":   true,
	"haproxy": true,
	"envoy":   true,
}

func RegisterProxyRoutes(se *core.ServeEvent, app core.App, rbac *middleware.RBAC) {
	se.Router.GET("/api/dk/proxies/instances", func(e *core.RequestEvent) error {
		return listProxyRecords(e, app, constants.CollProxyInstances)
	}).BindFunc(rbac.RequireCapability(constants.CapProxiesView))

	se.Router.GET("/api/dk/proxies/routes", func(e *core.RequestEvent) error {
		return listProxyRecords(e, app, constants.CollProxyRoutes)
	}).BindFunc(rbac.RequireCapability(constants.CapProxiesView))
}

func listProxyRecords(e *core.RequestEvent, app core.App, collection string) error {
	query := e.Request.URL.Query()

	page, _ := strconv.Atoi(query.Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(query.Get("perPage"))
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}

	kind := query.Get("kind")
	if kind != "" && !validProxyKinds[kind] {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid proxy kind"})
	}

	var filters []string
	params := map[string]any{}
	if server := query.Get("server"); server != "" {
		filters = append(filters, "server = {:server}")
		params["server"] = server
	}
	if kind != "" {
		if collection == constants.CollProxyInstances {
			filters = append(filters, "kind = {:kind}")
		} else {
			filters = append(filters, "instance.kind = {:kind}")
		}
		params["kind"] = kind
	}
	if search := strings.TrimSpace(query.Get("search")); search != "" {
		if collection == constants.CollProxyInstances {
			filters = append(filters, "(app ~ {:search} || kind ~ {:search})")
		} else {
			filters = append(filters, "(name ~ {:search} || provider ~ {:search})")
		}
		params["search"] = search
	}

	sort := "-last_seen"
	if collection == constants.CollProxyRoutes {
		sort = "-created"
	}

	records, err := app.FindRecordsByFilter(collection, strings.Join(filters, " && "), sort, 0, 0, params)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	totalItems := len(records)
	start := (page - 1) * perPage
	if start > totalItems {
		start = totalItems
	}
	end := start + perPage
	if end > totalItems {
		end = totalItems
	}

	totalPages := (totalItems + perPage - 1) / perPage
	return e.JSON(http.StatusOK, map[string]any{
		"items":      records[start:end],
		"page":       page,
		"perPage":    perPage,
		"totalItems": totalItems,
		"totalPages": totalPages,
	})
}
