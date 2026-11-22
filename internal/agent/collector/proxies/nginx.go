package proxies

import (
	"context"
	"strings"

	"github.com/onflux-tech/dockeeper/internal/agent/collector"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

const nginxDefaultConfig = "/etc/nginx/nginx.conf"

func (c *Collector) collectNginx(ctx context.Context, app collector.HostApp) *agentpb.ProxyInstanceInfo {
	inst := instance("nginx", app.Source, app.Name)
	inst.ConfigPath = nginxDefaultConfig

	output, err := c.runCommand(ctx, "nginx", "-T")
	if err != nil {
		inst.Status = "unknown"
		c.logger.Debug("proxies: nginx -T failed", "error", err)
		return inst
	}
	inst.Routes = parseNginxDump(output)
	if len(inst.Routes) == 0 {
		inst.Status = "unknown"
	}
	return inst
}

func parseNginxDump(dump string) []*agentpb.ProxyRouteInfo {
	upstreams := map[string][]string{}
	var routes []*agentpb.ProxyRouteInfo

	depth := 0
	pendingUpstream := ""
	upstreamDepth := -1
	serverDepth := -1
	locationDepth := -1
	locationPath := ""
	var hosts []string
	var paths []string
	var serverUpstreams []string
	var locationUpstreams []string
	var serverRoutes []*agentpb.ProxyRouteInfo
	tls := false

	emitServerRoute := func() {
		if len(hosts) == 0 && len(paths) == 0 && len(serverUpstreams) == 0 {
			return
		}
		if len(paths) == 0 {
			paths = []string{"/"}
		}
		serverRoutes = append(serverRoutes, &agentpb.ProxyRouteInfo{
			Name:      routeName(hosts, "nginx"),
			Hosts:     hosts,
			Paths:     paths,
			Upstreams: serverUpstreams,
			Tls:       tls,
			Status:    "healthy",
			Provider:  "nginx",
		})
	}

	for _, rawLine := range strings.Split(dump, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		switch {
		case strings.HasPrefix(line, "upstream ") && strings.HasSuffix(line, "{"):
			pendingUpstream = strings.TrimSuffix(strings.TrimPrefix(line, "upstream "), "{")
			pendingUpstream = strings.TrimSpace(pendingUpstream)
			upstreams[pendingUpstream] = nil
			upstreamDepth = depth
		case upstreamDepth >= 0 && strings.HasPrefix(line, "server "):
			addr := strings.TrimSuffix(strings.TrimPrefix(line, "server "), ";")
			upstreams[pendingUpstream] = append(upstreams[pendingUpstream], strings.Fields(addr)[0])
		case line == "server {":
			serverDepth = depth
			hosts = nil
			paths = nil
			serverUpstreams = nil
			serverRoutes = nil
			tls = false
		case strings.HasPrefix(line, "server_name ") && serverDepth >= 0:
			value := strings.TrimSuffix(strings.TrimPrefix(line, "server_name "), ";")
			for _, host := range strings.Fields(value) {
				if host != "_" {
					hosts = append(hosts, host)
				}
			}
		case strings.HasPrefix(line, "listen ") && serverDepth >= 0:
			if strings.Contains(line, "ssl") {
				tls = true
			}
		case line == "ssl on;":
			tls = true
		case strings.HasPrefix(line, "location ") && serverDepth >= 0 && strings.HasSuffix(line, "{"):
			locationDepth = depth
			locationPath = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "location "), "{"))
			locationUpstreams = nil
		case strings.HasPrefix(line, "proxy_pass ") && locationDepth >= 0:
			target := strings.TrimSuffix(strings.TrimPrefix(line, "proxy_pass "), ";")
			locationUpstreams = append(locationUpstreams, redactUpstream(resolveNginxUpstream(target, upstreams)))
		case line == "}":
			closingDepth := depth - 1
			switch {
			case locationDepth >= 0 && closingDepth == locationDepth:
				if len(locationUpstreams) > 0 {
					routes = append(routes, &agentpb.ProxyRouteInfo{
						Name:      routeName(hosts, "nginx"),
						Hosts:     hosts,
						Paths:     []string{locationPath},
						Upstreams: locationUpstreams,
						Tls:       tls,
						Status:    "healthy",
						Provider:  "nginx",
					})
				}
				locationDepth = -1
				locationUpstreams = nil
			case serverDepth >= 0 && closingDepth == serverDepth:
				if len(routes) == 0 {
					emitServerRoute()
				}
				routes = append(routes, serverRoutes...)
				serverDepth = -1
			case upstreamDepth >= 0 && closingDepth == upstreamDepth:
				pendingUpstream = ""
				upstreamDepth = -1
			}
			depth--
			continue
		}

		if strings.HasSuffix(line, "{") {
			depth++
		}
	}

	return routes
}

func resolveNginxUpstream(target string, upstreams map[string][]string) string {
	name := strings.TrimPrefix(target, "http://")
	name = strings.TrimPrefix(name, "https://")
	name = strings.TrimSuffix(name, "/")
	if servers, ok := upstreams[name]; ok && len(servers) > 0 {
		return strings.Join(servers, ",")
	}
	return target
}

func routeName(hosts []string, fallback string) string {
	if len(hosts) > 0 {
		return strings.Join(hosts, ",")
	}
	return fallback
}
