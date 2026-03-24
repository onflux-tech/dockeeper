package store

import (
	"log/slog"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/shared/constants"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// TraefikStore provides storage for Traefik route records.
type TraefikStore struct {
	app core.App
}

// NewTraefikStore creates a TraefikStore.
func NewTraefikStore(app core.App) *TraefikStore {
	return &TraefikStore{app: app}
}

// UpsertRoutes replaces all traefik_routes for a given server with the provided data.
func (s *TraefikStore) UpsertRoutes(serverID string, routes []*agentpb.TraefikRouteInfo) (int, error) {
	// Delete existing routes for this server (full replace).
	existing, err := s.app.FindRecordsByFilter(
		constants.CollTraefikRoutes,
		"server = '"+serverID+"'",
		"", 0, 0,
	)
	if err == nil {
		for _, rec := range existing {
			if delErr := s.app.Delete(rec); delErr != nil {
				slog.Warn("traefik: delete old route failed", "id", rec.Id, "error", delErr)
			}
		}
	}

	// Create new records.
	coll, err := s.app.FindCollectionByNameOrId(constants.CollTraefikRoutes)
	if err != nil {
		return 0, err
	}

	for _, r := range routes {
		rec := core.NewRecord(coll)
		rec.Set("server", serverID)
		rec.Set("name", r.GetName())
		rec.Set("rule", r.GetRule())
		rec.Set("service", r.GetService())
		rec.Set("entrypoints", r.GetEntrypoints())
		rec.Set("tls", r.GetTls())
		rec.Set("middlewares", r.GetMiddlewares())
		rec.Set("provider", r.GetProvider())
		rec.Set("status", r.GetStatus())

		if err := s.app.Save(rec); err != nil {
			slog.Error("traefik: save route failed", "name", r.GetName(), "error", err)
		}
	}

	return len(routes), nil
}
