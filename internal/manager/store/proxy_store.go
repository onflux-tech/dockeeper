package store

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/shared/constants"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

var proxyKinds = map[string]bool{
	"traefik": true,
	"nginx":   true,
	"caddy":   true,
	"haproxy": true,
	"envoy":   true,
}

var proxySources = map[string]bool{
	"systemd": true,
	"docker":  true,
	"process": true,
}

var proxyStatuses = map[string]bool{
	"healthy":  true,
	"degraded": true,
	"unknown":  true,
}

type ProxyStore struct {
	app core.App
}

func NewProxyStore(app core.App) *ProxyStore {
	return &ProxyStore{app: app}
}

func (s *ProxyStore) Replace(serverID string, instances []*agentpb.ProxyInstanceInfo) (int, int, error) {
	instancesCount := 0
	routesCount := 0

	err := s.app.RunInTransaction(func(txApp core.App) error {
		oldRoutes, err := txApp.FindRecordsByFilter(
			constants.CollProxyRoutes, "server = {:server}", "", 0, 0,
			map[string]any{"server": serverID},
		)
		if err != nil {
			return fmt.Errorf("find proxy routes: %w", err)
		}
		for _, record := range oldRoutes {
			if err := txApp.Delete(record); err != nil {
				return fmt.Errorf("delete proxy route: %w", err)
			}
		}

		oldInstances, err := txApp.FindRecordsByFilter(
			constants.CollProxyInstances, "server = {:server}", "", 0, 0,
			map[string]any{"server": serverID},
		)
		if err != nil {
			return fmt.Errorf("find proxy instances: %w", err)
		}
		for _, record := range oldInstances {
			if err := txApp.Delete(record); err != nil {
				return fmt.Errorf("delete proxy instance: %w", err)
			}
		}

		instancesCol, err := txApp.FindCachedCollectionByNameOrId(constants.CollProxyInstances)
		if err != nil {
			return fmt.Errorf("find collection %s: %w", constants.CollProxyInstances, err)
		}
		routesCol, err := txApp.FindCachedCollectionByNameOrId(constants.CollProxyRoutes)
		if err != nil {
			return fmt.Errorf("find collection %s: %w", constants.CollProxyRoutes, err)
		}

		now := time.Now().UTC()
		seen := map[string]bool{}
		for _, info := range instances {
			kind := info.GetKind()
			source := info.GetSource()
			if !proxyKinds[kind] || !proxySources[source] {
				continue
			}
			key := kind + "\x00" + source + "\x00" + info.GetApp()
			if seen[key] {
				continue
			}
			seen[key] = true

			record := core.NewRecord(instancesCol)
			record.Set("server", serverID)
			record.Set("kind", kind)
			record.Set("source", source)
			record.Set("app", info.GetApp())
			record.Set("version", info.GetVersion())
			record.Set("config_path", info.GetConfigPath())
			record.Set("admin_url", info.GetAdminUrl())
			record.Set("status", defaultProxyStatus(info.GetStatus()))
			record.Set("last_seen", now)
			if err := txApp.Save(record); err != nil {
				return fmt.Errorf("save proxy instance: %w", err)
			}
			instancesCount++

			for _, route := range info.GetRoutes() {
				routeRecord := core.NewRecord(routesCol)
				routeRecord.Set("instance", record.Id)
				routeRecord.Set("server", serverID)
				routeRecord.Set("name", route.GetName())
				routeRecord.Set("hosts", route.GetHosts())
				routeRecord.Set("paths", route.GetPaths())
				routeRecord.Set("upstreams", route.GetUpstreams())
				routeRecord.Set("tls", route.GetTls())
				routeRecord.Set("middlewares", route.GetMiddlewares())
				routeRecord.Set("status", defaultProxyStatus(route.GetStatus()))
				routeRecord.Set("provider", route.GetProvider())
				if err := txApp.Save(routeRecord); err != nil {
					return fmt.Errorf("save proxy route: %w", err)
				}
				routesCount++
			}
		}
		return nil
	})
	if err != nil {
		return 0, 0, err
	}

	return instancesCount, routesCount, nil
}

func defaultProxyStatus(status string) string {
	if !proxyStatuses[status] {
		return "unknown"
	}
	return status
}
