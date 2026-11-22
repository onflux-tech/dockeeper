package services

import (
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func newAlertEngineTestApp(t *testing.T) (*tests.TestApp, string, string) {
	t.Helper()

	app, err := tests.NewTestApp()
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)

	servers := core.NewBaseCollection("servers")
	servers.Fields.Add(&core.TextField{Name: "name"}, &core.TextField{Name: "hostname"})
	require.NoError(t, app.Save(servers))

	hostApps := core.NewBaseCollection("host_apps")
	hostApps.Fields.Add(
		&core.RelationField{Name: "server", CollectionId: servers.Id, MaxSelect: 1},
		&core.TextField{Name: "source"},
		&core.TextField{Name: "name"},
	)
	require.NoError(t, app.Save(hostApps))

	metrics := core.NewBaseCollection("host_app_metrics_raw")
	metrics.Fields.Add(
		&core.RelationField{Name: "server", CollectionId: servers.Id, MaxSelect: 1},
		&core.TextField{Name: "app"},
		&core.TextField{Name: "source"},
		&core.DateField{Name: "timestamp"},
		&core.NumberField{Name: "cpu_percent"},
		&core.NumberField{Name: "mem_bytes"},
	)
	require.NoError(t, app.Save(metrics))

	connectivity := core.NewBaseCollection("server_connectivity_raw")
	connectivity.Fields.Add(
		&core.RelationField{Name: "server", CollectionId: servers.Id, MaxSelect: 1},
		&core.DateField{Name: "timestamp"},
		&core.BoolField{Name: "internet_ok"},
		&core.NumberField{Name: "internet_ms"},
		&core.BoolField{Name: "dns_ok"},
		&core.BoolField{Name: "gateway_ok"},
	)
	require.NoError(t, app.Save(connectivity))

	server := core.NewRecord(servers)
	server.Set("name", "host-a")
	server.Set("hostname", "host-a")
	require.NoError(t, app.Save(server))

	hostApp := core.NewRecord(hostApps)
	hostApp.Set("server", server.Id)
	hostApp.Set("source", "systemd")
	hostApp.Set("name", "nginx.service")
	require.NoError(t, app.Save(hostApp))

	now := time.Now().UTC()
	for index, cpu := range []float64{42.5, 10} {
		record := core.NewRecord(metrics)
		record.Set("server", server.Id)
		record.Set("app", "nginx.service")
		record.Set("source", "systemd")
		record.Set("timestamp", now.Add(time.Duration(index)*time.Second).Format(time.RFC3339))
		record.Set("cpu_percent", cpu)
		require.NoError(t, app.Save(record))
	}

	for index, ok := range []bool{true, false} {
		record := core.NewRecord(connectivity)
		record.Set("server", server.Id)
		record.Set("timestamp", now.Add(time.Duration(index)*time.Second).Format(time.RFC3339))
		record.Set("internet_ok", ok)
		record.Set("internet_ms", 20)
		record.Set("dns_ok", ok)
		record.Set("gateway_ok", ok)
		require.NoError(t, app.Save(record))
	}

	return app, server.Id, hostApp.Id
}

func TestAlertEngineReadsHostAppAndConnectivityMetrics(t *testing.T) {
	app, serverID, hostAppID := newAlertEngineTestApp(t)
	engine := NewAlertEngine(app, nil, nil)

	hostValues, err := engine.queryMetrics("host_app", hostAppID, "cpu_percent", 300)
	require.NoError(t, err)
	require.Equal(t, []float64{10, 42.5}, hostValues)

	connectivityValues, err := engine.queryMetrics("server", serverID, "connectivity_internet_ok", 300)
	require.NoError(t, err)
	require.Equal(t, []float64{0, 1}, connectivityValues)

	latencyValues, err := engine.queryMetrics("server", serverID, "connectivity_internet_ms", 300)
	require.NoError(t, err)
	require.Equal(t, []float64{20, 20}, latencyValues)
}

func TestAlertEngineResolvesHostAppName(t *testing.T) {
	app, _, hostAppID := newAlertEngineTestApp(t)
	engine := NewAlertEngine(app, nil, nil)

	require.Equal(t, "nginx.service", engine.resolveTargetName("host_app", hostAppID))
}

func TestExtractConnectivityMetricKeepsUnknownFieldsOut(t *testing.T) {
	collection := core.NewBaseCollection("connectivity-fixture")
	collection.Fields.Add(&core.BoolField{Name: "internet_ok"})
	record := core.NewRecord(collection)
	record.Set("internet_ok", true)

	value, ok := extractConnectivityMetric(record, "connectivity_internet_ok")
	require.True(t, ok)
	require.Equal(t, float64(1), value)

	_, ok = extractConnectivityMetric(record, "cpu_percent")
	require.False(t, ok)
}
