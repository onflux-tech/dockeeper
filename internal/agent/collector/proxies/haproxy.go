package proxies

import (
	"context"
	"os"
	"strings"

	"github.com/onflux-tech/dockeeper/internal/agent/collector"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

const haproxyDefaultConfig = "/etc/haproxy/haproxy.cfg"

type haproxyFrontend struct {
	name    string
	hosts   []string
	bind    string
	backend string
	tls     bool
}

func (c *Collector) collectHAProxy(ctx context.Context, app collector.HostApp) *agentpb.ProxyInstanceInfo {
	inst := instance("haproxy", app.Source, app.Name)
	inst.ConfigPath = haproxyDefaultConfig

	if _, err := c.runCommand(ctx, "haproxy", "-c", "-f", haproxyDefaultConfig); err != nil {
		inst.Status = "unknown"
		c.logger.Debug("proxies: haproxy validation failed", "error", err)
		return inst
	}

	data, err := os.ReadFile(haproxyDefaultConfig)
	if err != nil {
		inst.Status = "unknown"
		return inst
	}
	inst.Routes = parseHAProxyConfig(string(data))
	if len(inst.Routes) == 0 {
		inst.Status = "unknown"
	}
	return inst
}

func parseHAProxyConfig(config string) []*agentpb.ProxyRouteInfo {
	backends := map[string][]string{}
	var frontends []*haproxyFrontend
	var current *haproxyFrontend
	currentBackend := ""
	section := ""

	for _, rawLine := range strings.Split(config, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		switch fields[0] {
		case "backend":
			section = "backend"
			current = nil
			currentBackend = fields[1]
			backends[currentBackend] = nil
		case "frontend", "listen":
			section = fields[0]
			current = &haproxyFrontend{name: fields[1]}
			frontends = append(frontends, current)
			if section == "listen" {
				backends[current.name] = nil
			}
		case "bind":
			if current != nil {
				current.bind = fields[1]
				current.tls = strings.Contains(line, "ssl")
			}
		case "acl":
			if current != nil && len(fields) >= 4 && strings.Contains(fields[2], "hdr(host)") {
				for _, host := range fields[3:] {
					if strings.HasPrefix(host, "-") {
						continue
					}
					current.hosts = append(current.hosts, strings.Trim(host, "\"'"))
				}
			}
		case "use_backend", "default_backend":
			if current != nil {
				current.backend = fields[1]
			}
		case "server":
			if len(fields) < 3 {
				continue
			}
			address := strings.Trim(fields[2], "\"'")
			switch section {
			case "backend":
				backends[currentBackend] = append(backends[currentBackend], address)
			case "listen":
				backends[current.name] = append(backends[current.name], address)
			}
		}
	}

	routes := make([]*agentpb.ProxyRouteInfo, 0, len(frontends))
	for _, frontend := range frontends {
		backend := frontend.backend
		if backend == "" {
			backend = frontend.name
		}
		upstreams := backends[backend]
		if len(upstreams) == 0 && len(frontend.hosts) == 0 {
			continue
		}
		routes = append(routes, &agentpb.ProxyRouteInfo{
			Name:      frontend.name,
			Hosts:     frontend.hosts,
			Paths:     []string{"/"},
			Upstreams: upstreams,
			Tls:       frontend.tls,
			Status:    "healthy",
			Provider:  "haproxy",
		})
	}
	return routes
}
