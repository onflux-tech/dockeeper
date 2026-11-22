package routes

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/pocketbase/dbx"
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

	filter := strings.Join(filters, " && ")
	if filter == "" {
		filter = "1=1"
	}
	start := (page - 1) * perPage

	records, err := app.FindRecordsByFilter(collection, filter, sort, perPage, start, params)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	totalItems, err := app.CountRecords(collection, dbx.NewExp(filter, dbx.Params(params)))
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	totalPages := (int(totalItems) + perPage - 1) / perPage

	items := make([]any, 0, len(records))
	for _, record := range records {
		item := record.PublicExport()
		if collection == constants.CollProxyInstances {
			count, err := app.CountRecords(constants.CollProxyRoutes, dbx.NewExp("instance = {:instance}", dbx.Params{"instance": record.Id}))
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}
			item["routes_count"] = count
		}
		items = append(items, item)
	}

	return e.JSON(http.StatusOK, map[string]any{
		"items":      items,
		"page":       page,
		"perPage":    perPage,
		"totalItems": totalItems,
		"totalPages": totalPages,
	})
}
