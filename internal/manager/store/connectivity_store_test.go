package store

import (
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

func newConnectivityTestApp(t *testing.T) (*tests.TestApp, string) {
	t.Helper()

	app, err := tests.NewTestApp()
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)

	servers := core.NewBaseCollection("servers")
	servers.Fields.Add(
		&core.TextField{Name: "name"},
		&core.TextField{Name: "hostname"},
		&core.TextField{Name: "status"},
		&core.TextField{Name: "connectivity_status"},
		&core.DateField{Name: "connectivity_checked_at"},
		&core.NumberField{Name: "connectivity_latency_ms"},
		&core.JSONField{Name: "connectivity_details"},
		&core.DateField{Name: "last_seen"},
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

func TestConnectivityStoreUpdatesCurrentState(t *testing.T) {
	app, serverID := newConnectivityTestApp(t)
	store := NewConnectivityStore(app)

	status, err := store.Save(serverID, &agentpb.ReportConnectivityRequest{
		InternetOk:    true,
		InternetMs:    24,
		DnsOk:         true,
		DnsMs:         9,
		GatewayOk:     false,
		GatewayMethod: "none",
		Error:         "probe failed: gateway",
		CollectedAt:   timestamppbNow(),
	})
	require.NoError(t, err)
	require.Equal(t, "degraded", status)

	server, err := app.FindRecordById("servers", serverID)
	require.NoError(t, err)
	require.Equal(t, "degraded", server.GetString("connectivity_status"))
	require.Equal(t, 33, server.GetInt("connectivity_latency_ms"))
	require.NotEmpty(t, server.GetString("connectivity_details"))

	rows, err := app.FindRecordsByFilter("server_connectivity_raw", "server = {:server}", "", 0, 0, map[string]any{"server": serverID})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.True(t, rows[0].GetBool("internet_ok"))
	require.False(t, rows[0].GetBool("gateway_ok"))
	require.Equal(t, "none", rows[0].GetString("gateway_method"))
}

func TestConnectivityStoreHealthy(t *testing.T) {
	app, serverID := newConnectivityTestApp(t)
	store := NewConnectivityStore(app)

	status, err := store.Save(serverID, &agentpb.ReportConnectivityRequest{
		InternetOk: true, InternetMs: 10,
		DnsOk: true, DnsMs: 5,
		GatewayOk: true, GatewayMs: 1, GatewayMethod: "icmp",
	})
	require.NoError(t, err)
	require.Equal(t, "healthy", status)

	server, err := app.FindRecordById("servers", serverID)
	require.NoError(t, err)
	require.Equal(t, "healthy", server.GetString("connectivity_status"))
	require.Equal(t, 16, server.GetInt("connectivity_latency_ms"))
}

func timestamppbNow() *timestamppb.Timestamp {
	return timestamppb.New(time.Now().UTC())
}
