package store

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"

	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

func newProxyStoreTestApp(t *testing.T) (*tests.TestApp, string) {
	t.Helper()

	app, err := tests.NewTestApp()
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)

	servers := core.NewBaseCollection("servers")
	servers.Fields.Add(&core.TextField{Name: "name"})
	require.NoError(t, app.Save(servers))

	instances := core.NewBaseCollection("proxy_instances")
	instances.Fields.Add(
		&core.RelationField{Name: "server", CollectionId: servers.Id, MaxSelect: 1, Required: true, CascadeDelete: true},
		&core.TextField{Name: "kind"},
		&core.TextField{Name: "source"},
		&core.TextField{Name: "app"},
		&core.TextField{Name: "version"},
		&core.TextField{Name: "config_path"},
		&core.TextField{Name: "admin_url"},
		&core.TextField{Name: "status"},
		&core.DateField{Name: "last_seen"},
	)
	require.NoError(t, app.Save(instances))

	routes := core.NewBaseCollection("proxy_routes")
	routes.Fields.Add(
		&core.RelationField{Name: "instance", CollectionId: instances.Id, MaxSelect: 1, Required: true, CascadeDelete: true},
		&core.RelationField{Name: "server", CollectionId: servers.Id, MaxSelect: 1, Required: true, CascadeDelete: true},
		&core.TextField{Name: "name"},
		&core.JSONField{Name: "hosts"},
		&core.JSONField{Name: "paths"},
		&core.JSONField{Name: "upstreams"},
		&core.BoolField{Name: "tls"},
		&core.JSONField{Name: "middlewares"},
		&core.TextField{Name: "status"},
		&core.TextField{Name: "provider"},
	)
	require.NoError(t, app.Save(routes))

	server := core.NewRecord(servers)
	server.Set("name", "srv1")
	require.NoError(t, app.Save(server))

	return app, server.Id
}

func TestProxyStoreReplacesPerServer(t *testing.T) {
	app, serverID := newProxyStoreTestApp(t)
	s := NewProxyStore(app)

	instances, routes, err := s.Replace(serverID, []*agentpb.ProxyInstanceInfo{
		{
			Kind:   "nginx",
			Source: "systemd",
			App:    "nginx.service",
			Status: "healthy",
			Routes: []*agentpb.ProxyRouteInfo{
				{Name: "app", Hosts: []string{"app.example.com"}, Paths: []string{"/"}, Upstreams: []string{"127.0.0.1:3000"}},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, 1, instances)
	require.Equal(t, 1, routes)

	instances, routes, err = s.Replace(serverID, []*agentpb.ProxyInstanceInfo{
		{
			Kind:   "caddy",
			Source: "systemd",
			App:    "caddy.service",
			Status: "healthy",
			Routes: []*agentpb.ProxyRouteInfo{
				{Name: "app", Hosts: []string{"app.example.com"}, Paths: []string{"/"}, Upstreams: []string{"127.0.0.1:3000"}},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, 1, instances)
	require.Equal(t, 1, routes)

	rows, err := app.FindRecordsByFilter("proxy_instances", "server = {:server}", "", 0, 0, map[string]any{"server": serverID})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "caddy", rows[0].GetString("kind"))

	routeRows, err := app.FindRecordsByFilter("proxy_routes", "server = {:server}", "", 0, 0, map[string]any{"server": serverID})
	require.NoError(t, err)
	require.Len(t, routeRows, 1)
}

func TestProxyStoreIgnoresUnknownKinds(t *testing.T) {
	app, serverID := newProxyStoreTestApp(t)
	s := NewProxyStore(app)

	instances, routes, err := s.Replace(serverID, []*agentpb.ProxyInstanceInfo{
		{Kind: "apache", Source: "systemd", App: "apache.service"},
		{Kind: "nginx", Source: "unknown", App: "nginx.service"},
	})
	require.NoError(t, err)
	require.Zero(t, instances)
	require.Zero(t, routes)
}
