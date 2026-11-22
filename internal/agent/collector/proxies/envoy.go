package proxies

import (
	"context"

	"github.com/onflux-tech/dockeeper/internal/agent/collector"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

const envoyAdminURL = "http://127.0.0.1:9901/config_dump"

func (c *Collector) collectEnvoy(ctx context.Context, app collector.HostApp) *agentpb.ProxyInstanceInfo {
	inst := instance("envoy", app.Source, app.Name)
	inst.AdminUrl = envoyAdminURL

	var dump map[string]any
	if err := c.getJSON(ctx, envoyAdminURL, &dump); err != nil {
		inst.Status = "unknown"
		c.logger.Debug("proxies: envoy admin API failed", "error", err)
		return inst
	}
	inst.Routes = parseEnvoyConfigDump(dump)
	if len(inst.Routes) == 0 {
		inst.Status = "unknown"
	}
	return inst
}

func parseEnvoyConfigDump(dump map[string]any) []*agentpb.ProxyRouteInfo {
	configs, _ := dump["configs"].([]any)
	var routes []*agentpb.ProxyRouteInfo

	for _, rawConfig := range configs {
		config, _ := rawConfig.(map[string]any)
		for _, field := range []string{"static_route_configs", "dynamic_route_configs"} {
			entries, _ := config[field].([]any)
			for _, rawEntry := range entries {
				entry, _ := rawEntry.(map[string]any)
				routeConfig, _ := entry["route_config"].(map[string]any)
				vhosts, _ := routeConfig["virtual_hosts"].([]any)
				for _, rawVHost := range vhosts {
					vhost, _ := rawVHost.(map[string]any)
					hosts := stringSlice(vhost["domains"])
					var paths, upstreams []string
					vhostRoutes, _ := vhost["routes"].([]any)
					for _, rawRoute := range vhostRoutes {
						route, _ := rawRoute.(map[string]any)
						match, _ := route["match"].(map[string]any)
						path := "/"
						if prefix, ok := match["prefix"].(string); ok && prefix != "" {
							path = prefix
						}
						if exact, ok := match["path"].(string); ok && exact != "" {
							path = exact
						}
						paths = append(paths, path)
						routeAction, _ := route["route"].(map[string]any)
						if cluster, ok := routeAction["cluster"].(string); ok && cluster != "" {
							upstreams = append(upstreams, redactUpstream(cluster))
						}
					}
					if len(hosts) == 0 && len(upstreams) == 0 {
						continue
					}
					if len(paths) == 0 {
						paths = []string{"/"}
					}
					routes = append(routes, &agentpb.ProxyRouteInfo{
						Name:      routeName(hosts, "envoy"),
						Hosts:     hosts,
						Paths:     paths,
						Upstreams: upstreams,
						Status:    "healthy",
						Provider:  "envoy",
					})
				}
			}
		}
	}
	return routes
}
