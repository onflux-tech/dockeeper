package routes

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// RegisterSearchRoutes adds the global search endpoint.
func RegisterSearchRoutes(se *core.ServeEvent, app core.App) {
	se.Router.GET("/api/dk/search", func(e *core.RequestEvent) error {
		return handleGlobalSearch(e, app)
	}).BindFunc(middleware.RequireSuperuser)
}

// ── Types ────────────────────────────────────────────────────────────────────

type searchResultItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Subtitle string `json:"subtitle"`
	Href     string `json:"href"`
}

type searchResultGroup struct {
	Type        string             `json:"type"`
	Label       string             `json:"label"`
	Total       int                `json:"total"`
	ViewAllHref string             `json:"viewAllHref"`
	Items       []searchResultItem `json:"items"`
}

type searchResponse struct {
	Query  string              `json:"query"`
	Groups []searchResultGroup `json:"groups"`
}

// ── Handler ──────────────────────────────────────────────────────────────────

func handleGlobalSearch(e *core.RequestEvent, app core.App) error {
	q := strings.TrimSpace(e.Request.URL.Query().Get("q"))
	if len(q) < 2 {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "query parameter 'q' must be at least 2 characters",
		})
	}

	limit, _ := strconv.Atoi(e.Request.URL.Query().Get("limit"))
	if limit < 1 {
		limit = 5
	}
	if limit > 20 {
		limit = 20
	}

	// Define which collections to search and how.
	type collectionDef struct {
		collection   string
		label        string
		entityType   string
		searchFields []string
		nameField    string
		subtitleFn   func(r *core.Record) string
		hrefPrefix   string
		viewAllHref  string
	}

	defs := []collectionDef{
		{
			collection:   constants.CollServers,
			label:        "Servers",
			entityType:   "servers",
			searchFields: []string{"name", "hostname", "ip_address"},
			nameField:    "name",
			subtitleFn: func(r *core.Record) string {
				parts := []string{}
				if h := r.GetString("hostname"); h != "" {
					parts = append(parts, h)
				}
				if ip := r.GetString("ip_address"); ip != "" {
					parts = append(parts, ip)
				}
				return strings.Join(parts, " · ")
			},
			hrefPrefix:  "/servers/",
			viewAllHref: "/servers?search=",
		},
		{
			collection:   constants.CollContainers,
			label:        "Containers",
			entityType:   "containers",
			searchFields: []string{"name", "image"},
			nameField:    "name",
			subtitleFn: func(r *core.Record) string {
				parts := []string{}
				if s := r.GetString("status"); s != "" {
					parts = append(parts, s)
				}
				if img := r.GetString("image"); img != "" {
					parts = append(parts, img)
				}
				return strings.Join(parts, " · ")
			},
			hrefPrefix:  "/containers/",
			viewAllHref: "/containers?search=",
		},
		{
			collection:   constants.CollImages,
			label:        "Images",
			entityType:   "images",
			searchFields: []string{"tags", "image_id"},
			nameField:    "tags",
			subtitleFn: func(r *core.Record) string {
				id := r.GetString("image_id")
				if len(id) > 19 {
					id = id[:19]
				}
				return id
			},
			hrefPrefix:  "/images?search=",
			viewAllHref: "/images?search=",
		},
		{
			collection:   constants.CollVolumes,
			label:        "Volumes",
			entityType:   "volumes",
			searchFields: []string{"name", "driver"},
			nameField:    "name",
			subtitleFn: func(r *core.Record) string {
				return r.GetString("driver")
			},
			hrefPrefix:  "/volumes?search=",
			viewAllHref: "/volumes?search=",
		},
		{
			collection:   constants.CollNetworks,
			label:        "Networks",
			entityType:   "networks",
			searchFields: []string{"name", "network_id", "driver"},
			nameField:    "name",
			subtitleFn: func(r *core.Record) string {
				parts := []string{}
				if d := r.GetString("driver"); d != "" {
					parts = append(parts, d)
				}
				if s := r.GetString("scope"); s != "" {
					parts = append(parts, s)
				}
				return strings.Join(parts, " · ")
			},
			hrefPrefix:  "/networks?search=",
			viewAllHref: "/networks?search=",
		},
		{
			collection:   constants.CollComposeStacks,
			label:        "Compose Stacks",
			entityType:   "compose_stacks",
			searchFields: []string{"name"},
			nameField:    "name",
			subtitleFn: func(r *core.Record) string {
				return r.GetString("status")
			},
			hrefPrefix:  "/compose?search=",
			viewAllHref: "/compose?search=",
		},
		{
			collection:   constants.CollTraefikRoutes,
			label:        "Traefik Routes",
			entityType:   "traefik_routes",
			searchFields: []string{"name", "rule", "service"},
			nameField:    "name",
			subtitleFn: func(r *core.Record) string {
				parts := []string{}
				if rule := r.GetString("rule"); rule != "" {
					parts = append(parts, rule)
				}
				if svc := r.GetString("service"); svc != "" {
					parts = append(parts, svc)
				}
				return strings.Join(parts, " · ")
			},
			hrefPrefix:  "/traefik?search=",
			viewAllHref: "/traefik?search=",
		},
	}

	// Query all collections in parallel.
	var mu sync.Mutex
	var wg sync.WaitGroup
	groups := make([]searchResultGroup, len(defs))

	for i, def := range defs {
		wg.Add(1)
		go func(idx int, d collectionDef) {
			defer wg.Done()

			// Build filter: (field1 ~ 'escaped' || field2 ~ 'escaped' || ...)
			// Use direct string interpolation (same pattern as traefik.go)
			// to avoid issues with named params + LIKE operator.
			escaped := strings.ReplaceAll(q, "'", "\\'")
			filterParts := make([]string, len(d.searchFields))
			for j, field := range d.searchFields {
				filterParts[j] = field + " ~ '" + escaped + "'"
			}
			filter := "(" + strings.Join(filterParts, " || ") + ")"

			// Count total.
			allRecords, countErr := app.FindRecordsByFilter(
				d.collection,
				filter,
				"",
				0, 0,
			)
			total := 0
			if countErr != nil {
				slog.Warn("global search: count failed",
					"collection", d.collection,
					"error", countErr,
				)
			} else {
				total = len(allRecords)
			}

			// Fetch limited results.
			var items []searchResultItem
			if total > 0 {
				records, err := app.FindRecordsByFilter(
					d.collection,
					filter,
					"-created",
					limit, 0,
				)
				if err != nil {
					slog.Warn("global search: query failed",
						"collection", d.collection,
						"error", err,
					)
				} else {
					for _, r := range records {
						name := r.GetString(d.nameField)
						if name == "" {
							name = r.Id
						}
						hrefValue := r.Id
						if strings.Contains(d.hrefPrefix, "?search=") {
							hrefValue = name
						}
						items = append(items, searchResultItem{
							ID:       r.Id,
							Name:     name,
							Subtitle: d.subtitleFn(r),
							Href:     d.hrefPrefix + hrefValue,
						})
					}
				}
			}

			mu.Lock()
			groups[idx] = searchResultGroup{
				Type:        d.entityType,
				Label:       d.label,
				Total:       total,
				ViewAllHref: d.viewAllHref + q,
				Items:       items,
			}
			mu.Unlock()
		}(i, def)
	}

	wg.Wait()

	// Filter out groups with no results.
	var nonEmpty []searchResultGroup
	for _, g := range groups {
		if g.Total > 0 {
			nonEmpty = append(nonEmpty, g)
		}
	}

	return e.JSON(http.StatusOK, searchResponse{
		Query:  q,
		Groups: nonEmpty,
	})
}
