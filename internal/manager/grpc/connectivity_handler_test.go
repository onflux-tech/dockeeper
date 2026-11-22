package grpc

import (
	"context"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"

	"github.com/onflux-tech/dockeeper/internal/manager/events"
	"github.com/onflux-tech/dockeeper/internal/manager/store"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

func newConnectivityHandlerTestApp(t *testing.T) (*tests.TestApp, string) {
	t.Helper()

	app, err := tests.NewTestApp()
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)

	servers := core.NewBaseCollection("servers")
	servers.Fields.Add(
		&core.TextField{Name: "name"},
		&core.TextField{Name: "hostname"},
		&core.TextField{Name: "status"},
		&core.DateField{Name: "last_seen"},
		&core.TextField{Name: "connectivity_status"},
		&core.DateField{Name: "connectivity_checked_at"},
		&core.NumberField{Name: "connectivity_latency_ms"},
		&core.JSONField{Name: "connectivity_details"},
	)
	require.NoError(t, app.Save(servers))

	raw := core.NewBaseCollection("server_connectivity_raw")
	raw.Fields.Add(
		&core.RelationField{Name: "server", CollectionId: servers.Id, MaxSelect: 1},
		&core.DateField{Name: "timestamp"},
		&core.BoolField{Name: "internet_ok"},
		&core.NumberField{Name: "internet_ms"},
		&core.BoolField{Name: "dns_ok"},
		&core.NumberField{Name: "dns_ms"},
		&core.BoolField{Name: "gateway_ok"},
		&core.NumberField{Name: "gateway_ms"},
		&core.TextField{Name: "gateway_method"},
		&core.TextField{Name: "error"},
	)
	require.NoError(t, app.Save(raw))

	server := core.NewRecord(servers)
	server.Set("name", "host-a")
	server.Set("hostname", "host-a")
	server.Set("status", "online")
	require.NoError(t, app.Save(server))

	return app, server.Id
}

func TestReportConnectivityPublishesOnlyOnChange(t *testing.T) {
	app, serverID := newConnectivityHandlerTestApp(t)
	bus := events.NewBus()
	defer bus.Close()

	var published []events.EventType
	bus.Subscribe(events.HandlerFunc(func(event events.Event) {
		switch event.Type {
		case events.ServerConnectivityDegraded, events.ServerConnectivityRecovered:
			published = append(published, event.Type)
		}
	}))

	handler := NewHostHandler(app, store.NewHostStore(app), bus)
	ctx := context.WithValue(context.Background(), serverIDKey, serverID)

	degraded := &agentpb.ReportConnectivityRequest{
		InternetOk: true, InternetMs: 20, DnsOk: true, DnsMs: 5,
		GatewayOk: false, GatewayMethod: "none", Error: "probe failed: gateway",
	}
	_, err := handler.ReportConnectivity(ctx, degraded)
	require.NoError(t, err)
	_, err = handler.ReportConnectivity(ctx, degraded)
	require.NoError(t, err)

	healthy := &agentpb.ReportConnectivityRequest{
		InternetOk: true, InternetMs: 18, DnsOk: true, DnsMs: 4,
		GatewayOk: true, GatewayMs: 1, GatewayMethod: "icmp",
	}
	_, err = handler.ReportConnectivity(ctx, healthy)
	require.NoError(t, err)

	bus.Close()

	require.Equal(t, []events.EventType{
		events.ServerConnectivityDegraded,
		events.ServerConnectivityRecovered,
	}, published)

	server, err := app.FindRecordById("servers", serverID)
	require.NoError(t, err)
	require.Equal(t, "healthy", server.GetString("connectivity_status"))
	require.Equal(t, 23, server.GetInt("connectivity_latency_ms"))
}
