package routes

import (
	"fmt"
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

func proxyCountFilter(filter string) string {
	return strings.NewReplacer("&&", "AND", "||", "OR").Replace(filter)
}

func proxyRouteInstanceFilter(instanceIDs []string) (string, map[string]any) {
	clauses := make([]string, 0, len(instanceIDs))
	params := make(map[string]any, len(instanceIDs))
	for index, instanceID := range instanceIDs {
		key := fmt.Sprintf("instance%d", index)
		clauses = append(clauses, fmt.Sprintf("instance = {:%s}", key))
		params[key] = instanceID
	}
	return "(" + strings.Join(clauses, " || ") + ")", params
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

	server := query.Get("server")
	var filters []string
	params := map[string]any{}
	if server != "" {
		filters = append(filters, "server = {:server}")
		params["server"] = server
	}
	if kind != "" {
		if collection == constants.CollProxyInstances {
			filters = append(filters, "kind = {:kind}")
			params["kind"] = kind
		} else {
			instanceFilter := "kind = {:kind}"
			instanceParams := map[string]any{"kind": kind}
			if server != "" {
				instanceFilter += " && server = {:server}"
				instanceParams["server"] = server
			}
			instances, err := app.FindRecordsByFilter(constants.CollProxyInstances, instanceFilter, "", 500, 0, instanceParams)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}
			if len(instances) == 0 {
				return e.JSON(http.StatusOK, map[string]any{
					"items":      []any{},
					"page":       page,
					"perPage":    perPage,
					"totalItems": 0,
					"totalPages": 0,
				})
			}
			instanceIDs := make([]string, 0, len(instances))
			for _, instance := range instances {
				instanceIDs = append(instanceIDs, instance.Id)
			}
			filter, filterParams := proxyRouteInstanceFilter(instanceIDs)
			filters = append(filters, filter)
			for key, value := range filterParams {
				params[key] = value
			}
		}
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

	totalItems, err := app.CountRecords(collection, dbx.NewExp(proxyCountFilter(filter), dbx.Params(params)))
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
