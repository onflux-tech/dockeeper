package proxies

import (
	"context"
	"strings"

	"github.com/onflux-tech/dockeeper/internal/agent/collector"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

type traefikRouter struct {
	Name     string         `json:"name"`
	Rule     string         `json:"rule"`
	Service  string         `json:"service"`
	Provider string         `json:"provider"`
	Status   string         `json:"status"`
	TLS      map[string]any `json:"tls"`
}

type traefikService struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Servers  []struct {
		URL string `json:"url"`
	} `json:"servers"`
}

func (c *Collector) collectTraefik(ctx context.Context, app *collector.HostApp, apiURL string) *agentpb.ProxyInstanceInfo {
	inst := instance("traefik", "docker", "traefik")
	if app != nil {
		inst.Source = app.Source
		inst.App = app.Name
	}
	inst.AdminUrl = apiURL

	base := strings.TrimSuffix(apiURL, "/")
	var routers []traefikRouter
	apiErr := c.getJSON(ctx, base+"/api/http/routers", &routers)
	if apiErr == nil && len(routers) > 0 {
		var services []traefikService
		if err := c.getJSON(ctx, base+"/api/http/services", &services); err != nil {
			c.logger.Debug("proxies: traefik services failed", "error", err)
		}
		inst.Routes = parseTraefik(routers, services)
		return inst
	}

	if c.labelRoutes != nil {
		if routes := c.labelRoutes(ctx); len(routes) > 0 {
			inst.Routes = convertTraefikRoutes(routes)
			return inst
		}
	}

	inst.Status = "unknown"
	if apiErr != nil {
		c.logger.Debug("proxies: traefik API failed", "error", apiErr)
	}
	return inst
}

func convertTraefikRoutes(routes []*agentpb.TraefikRouteInfo) []*agentpb.ProxyRouteInfo {
	converted := make([]*agentpb.ProxyRouteInfo, 0, len(routes))
	for _, route := range routes {
		hosts := ruleValues(route.GetRule(), "Host")
		paths := append(ruleValues(route.GetRule(), "PathPrefix"), ruleValues(route.GetRule(), "Path")...)
		if len(paths) == 0 {
			paths = []string{"/"}
		}
		status := traefikRouteStatus(route.GetStatus())
		converted = append(converted, &agentpb.ProxyRouteInfo{
			Name:        route.GetName(),
			Hosts:       hosts,
			Paths:       paths,
			Tls:         route.GetTls(),
			Middlewares: route.GetMiddlewares(),
			Status:      status,
			Provider:    route.GetProvider(),
		})
	}
	return converted
}

func parseTraefik(routers []traefikRouter, services []traefikService) []*agentpb.ProxyRouteInfo {
	serviceURLs := map[string][]string{}
	for _, service := range services {
		for _, server := range service.Servers {
			if server.URL != "" {
				serviceURLs[service.Name] = append(serviceURLs[service.Name], redactUpstream(server.URL))
			}
		}
	}

	routes := make([]*agentpb.ProxyRouteInfo, 0, len(routers))
	for _, router := range routers {
		hosts := ruleValues(router.Rule, "Host")
		paths := append(ruleValues(router.Rule, "PathPrefix"), ruleValues(router.Rule, "Path")...)
		if len(paths) == 0 {
			paths = []string{"/"}
		}
		status := traefikRouteStatus(router.Status)
		routes = append(routes, &agentpb.ProxyRouteInfo{
			Name:      router.Name,
			Hosts:     hosts,
			Paths:     paths,
			Upstreams: serviceURLs[router.Service],
			Tls:       router.TLS != nil,
			Status:    status,
			Provider:  router.Provider,
		})
	}
	return routes
}

func traefikRouteStatus(status string) string {
	switch status {
	case "", "enabled":
		return "healthy"
	case "disabled":
		return "degraded"
	default:
		return "unknown"
	}
}

func ruleValues(rule, function string) []string {
	marker := function + "("
	index := strings.Index(rule, marker)
	if index < 0 {
		return nil
	}
	rest := rule[index+len(marker):]
	end := strings.Index(rest, ")")
	if end < 0 {
		return nil
	}

	parts := strings.Split(rest[:end], ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.Trim(strings.TrimSpace(part), "`\"'")
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}
