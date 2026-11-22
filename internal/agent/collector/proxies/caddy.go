package proxies

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/onflux-tech/dockeeper/internal/agent/collector"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

const caddyDefaultConfig = "/etc/caddy/Caddyfile"

func (c *Collector) collectCaddy(ctx context.Context, app collector.HostApp) *agentpb.ProxyInstanceInfo {
	inst := instance("caddy", app.Source, app.Name)
	inst.ConfigPath = caddyDefaultConfig

	output, err := c.runCommand(ctx, "caddy", "adapt", "--config", caddyDefaultConfig, "--adapter", "caddyfile")
	if err != nil {
		inst.Status = "unknown"
		c.logger.Debug("proxies: caddy adapt failed", "error", err)
		return inst
	}
	inst.Routes = parseCaddyJSON([]byte(output))
	if len(inst.Routes) == 0 {
		inst.Status = "unknown"
	}
	return inst
}

func parseCaddyJSON(data []byte) []*agentpb.ProxyRouteInfo {
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil
	}

	apps, _ := root["apps"].(map[string]any)
	httpApp, _ := apps["http"].(map[string]any)
	servers, _ := httpApp["servers"].(map[string]any)

	var routes []*agentpb.ProxyRouteInfo
	for name, rawServer := range servers {
		server, _ := rawServer.(map[string]any)
		tls := caddyTLS(server["listen"])
		rawRoutes, _ := server["routes"].([]any)
		for _, rawRoute := range rawRoutes {
			route, _ := rawRoute.(map[string]any)
			hosts, paths := caddyMatch(route["match"])
			upstreams := caddyUpstreams(route["handle"])
			if len(hosts) == 0 && len(upstreams) == 0 {
				continue
			}
			if len(paths) == 0 {
				paths = []string{"/"}
			}
			routes = append(routes, &agentpb.ProxyRouteInfo{
				Name:      name,
				Hosts:     hosts,
				Paths:     paths,
				Upstreams: upstreams,
				Tls:       tls,
				Status:    "healthy",
				Provider:  "caddy",
			})
		}
	}
	return routes
}

func caddyTLS(listen any) bool {
	for _, value := range stringSlice(listen) {
		if strings.Contains(value, ":443") || strings.HasPrefix(value, "https://") {
			return true
		}
	}
	return false
}

func caddyMatch(match any) ([]string, []string) {
	var hosts, paths []string
	entries, _ := match.([]any)
	for _, rawEntry := range entries {
		entry, _ := rawEntry.(map[string]any)
		hosts = append(hosts, stringSlice(entry["host"])...)
		paths = append(paths, stringSlice(entry["path"])...)
	}
	return hosts, paths
}

func caddyUpstreams(handle any) []string {
	var upstreams []string
	entries, _ := handle.([]any)
	for _, rawEntry := range entries {
		entry, _ := rawEntry.(map[string]any)
		if entry["handler"] != "reverse_proxy" {
			continue
		}
		servers, _ := entry["upstreams"].([]any)
		for _, rawServer := range servers {
			server, _ := rawServer.(map[string]any)
			if dial, ok := server["dial"].(string); ok && dial != "" {
				upstreams = append(upstreams, redactUpstream(dial))
			}
		}
	}
	return upstreams
}

func stringSlice(value any) []string {
	entries, _ := value.([]any)
	result := make([]string, 0, len(entries))
	for _, entry := range entries {
		if text, ok := entry.(string); ok {
			result = append(result, text)
		}
	}
	return result
}
