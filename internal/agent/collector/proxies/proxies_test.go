package proxies

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflux-tech/dockeeper/internal/agent/collector"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

const nginxFixture = `# configuration file /etc/nginx/nginx.conf:
http {
    upstream api_backend {
        server 127.0.0.1:3000;
        server 127.0.0.1:3001;
    }
    server {
        listen 443 ssl;
        server_name app.example.com www.example.com;
        location / {
            proxy_pass http://api_backend;
        }
        location /static {
            proxy_pass http://127.0.0.1:8080;
        }
    }
}
`

const caddyFixture = `{
  "apps": {
    "http": {
      "servers": {
        "srv0": {
          "listen": [":443"],
          "routes": [
            {
              "match": [{"host": ["app.example.com"]}],
              "handle": [{"handler": "reverse_proxy", "upstreams": [{"dial": "127.0.0.1:3000"}]}]
            }
          ]
        }
      }
    }
  }
}`

const haproxyFixture = `global
    daemon

frontend web
    bind *:443 ssl crt /etc/ssl/site.pem
    acl host_app hdr(host) app.example.com
    use_backend app_servers

backend app_servers
    server app1 127.0.0.1:3000 check
`

const envoyFixture = `{
  "configs": [
    {
      "dynamic_route_configs": [
        {
          "route_config": {
            "virtual_hosts": [
              {
                "domains": ["app.example.com"],
                "routes": [
                  {"match": {"prefix": "/api"}, "route": {"cluster": "api_cluster"}}
                ]
              }
            ]
          }
        }
      ]
    }
  ]
}`

func TestParseNginxDump(t *testing.T) {
	routes := parseNginxDump(nginxFixture)
	require.Len(t, routes, 2)

	require.Equal(t, []string{"app.example.com", "www.example.com"}, routes[0].Hosts)
	require.Equal(t, []string{"/"}, routes[0].Paths)
	require.Equal(t, []string{"127.0.0.1:3000,127.0.0.1:3001"}, routes[0].Upstreams)
	require.True(t, routes[0].Tls)

	require.Equal(t, []string{"/static"}, routes[1].Paths)
	require.Equal(t, []string{"http://127.0.0.1:8080"}, routes[1].Upstreams)
}

func TestParseNginxDumpRedactsCredentials(t *testing.T) {
	routes := parseNginxDump(`http {
	server {
		server_name secure.example.com;
		location / {
			proxy_pass http://user:secret@10.0.0.5:8080;
		}
	}
}`)
	require.Len(t, routes, 1)
	require.Equal(t, []string{"http://10.0.0.5:8080"}, routes[0].Upstreams)
}

func TestRedactUpstream(t *testing.T) {
	require.Equal(t, "http://10.0.0.5:8080", redactUpstream("http://user:pass@10.0.0.5:8080"))
	require.Equal(t, "http://10.0.0.5:8080/path?token=redacted", redactUpstream("http://10.0.0.5:8080/path?token=abc"))
	require.Equal(t, "10.0.0.5:8080", redactUpstream("user:pass@10.0.0.5:8080"))
	require.Equal(t, "127.0.0.1:3000", redactUpstream("127.0.0.1:3000"))
	require.Equal(t, "api_cluster", redactUpstream("api_cluster"))
}

func TestParseCaddyJSON(t *testing.T) {
	routes := parseCaddyJSON([]byte(caddyFixture))
	require.Len(t, routes, 1)
	require.Equal(t, "caddy", routes[0].Provider)
	require.Equal(t, []string{"app.example.com"}, routes[0].Hosts)
	require.Equal(t, []string{"127.0.0.1:3000"}, routes[0].Upstreams)
	require.True(t, routes[0].Tls)
}

func TestParseHAProxyConfig(t *testing.T) {
	routes := parseHAProxyConfig(haproxyFixture)
	require.Len(t, routes, 1)
	require.Equal(t, "web", routes[0].Name)
	require.Equal(t, []string{"app.example.com"}, routes[0].Hosts)
	require.Equal(t, []string{"127.0.0.1:3000"}, routes[0].Upstreams)
	require.True(t, routes[0].Tls)
}

func TestParseEnvoyConfigDump(t *testing.T) {
	var dump map[string]any
	require.NoError(t, json.Unmarshal([]byte(envoyFixture), &dump))

	routes := parseEnvoyConfigDump(dump)
	require.Len(t, routes, 1)
	require.Equal(t, []string{"app.example.com"}, routes[0].Hosts)
	require.Equal(t, []string{"/api"}, routes[0].Paths)
	require.Equal(t, []string{"api_cluster"}, routes[0].Upstreams)
}

func TestParseTraefik(t *testing.T) {
	routers := []traefikRouter{
		{
			Name:     "api@docker",
			Rule:     "Host(`app.example.com`) && PathPrefix(`/api`)",
			Service:  "api@docker",
			Provider: "docker",
			Status:   "enabled",
			TLS:      map[string]any{},
		},
	}
	services := []traefikService{
		{Name: "api@docker", Servers: []struct {
			URL string `json:"url"`
		}{{URL: "http://127.0.0.1:3000"}}},
	}

	routes := parseTraefik(routers, services)
	require.Len(t, routes, 1)
	require.Equal(t, []string{"app.example.com"}, routes[0].Hosts)
	require.Equal(t, []string{"/api"}, routes[0].Paths)
	require.Equal(t, []string{"http://127.0.0.1:3000"}, routes[0].Upstreams)
	require.True(t, routes[0].Tls)
	require.Equal(t, "healthy", routes[0].Status)
}

func TestCollectorDetectsNginx(t *testing.T) {
	apps := []collector.HostApp{
		{Name: "nginx.service", Source: "systemd", Unit: "nginx.service"},
	}
	c := New(
		slog.Default(),
		func() []collector.HostApp { return apps },
		WithCommandRunner(func(context.Context, string, ...string) (string, error) {
			return nginxFixture, nil
		}),
	)

	instances, err := c.Collect(context.Background())
	require.NoError(t, err)
	require.Len(t, instances, 1)
	require.Equal(t, "nginx", instances[0].Kind)
	require.Equal(t, "systemd", instances[0].Source)
	require.Len(t, instances[0].Routes, 2)
}

func TestCollectorTraefikFallsBackToLabels(t *testing.T) {
	c := New(
		slog.Default(),
		func() []collector.HostApp { return nil },
		WithTraefikAPIURL(func(context.Context) string { return "http://127.0.0.1:1" }),
		WithTraefikLabelRoutes(func(context.Context) []*agentpb.TraefikRouteInfo {
			return []*agentpb.TraefikRouteInfo{{
				Name:     "app",
				Rule:     "Host(`app.example.com`) && PathPrefix(`/api`)",
				Provider: "docker",
				Status:   "enabled",
				Tls:      true,
			}}
		}),
	)

	instances, err := c.Collect(context.Background())
	require.NoError(t, err)
	require.Len(t, instances, 1)
	require.Equal(t, "traefik", instances[0].Kind)
	require.Equal(t, "healthy", instances[0].Status)
	require.Len(t, instances[0].Routes, 1)
	require.Equal(t, []string{"app.example.com"}, instances[0].Routes[0].Hosts)
	require.Equal(t, []string{"/api"}, instances[0].Routes[0].Paths)
	require.True(t, instances[0].Routes[0].Tls)
}

func TestCollectorTraefikWithoutAPIOrLabelsIsUnknown(t *testing.T) {
	c := New(
		slog.Default(),
		func() []collector.HostApp { return nil },
		WithTraefikAPIURL(func(context.Context) string { return "http://127.0.0.1:1" }),
	)

	instances, err := c.Collect(context.Background())
	require.NoError(t, err)
	require.Len(t, instances, 1)
	require.Equal(t, "unknown", instances[0].Status)
	require.Empty(t, instances[0].Routes)
}
