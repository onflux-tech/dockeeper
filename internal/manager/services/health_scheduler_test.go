package services

import (
	"context"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"

	"github.com/onflux-tech/dockeeper/internal/manager/events"
	"github.com/onflux-tech/dockeeper/internal/manager/store"
)

func newHealthSchedulerTestApp(t *testing.T) (*tests.TestApp, string) {
	t.Helper()

	app, err := tests.NewTestApp()
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)

	servers := core.NewBaseCollection("servers")
	servers.Fields.Add(
		&core.TextField{Name: "name"},
		&core.TextField{Name: "status"},
	)
	require.NoError(t, app.Save(servers))

	containers := core.NewBaseCollection("containers")
	containers.Fields.Add(
		&core.RelationField{Name: "server", CollectionId: servers.Id, MaxSelect: 1},
		&core.TextField{Name: "container_id"},
		&core.TextField{Name: "status"},
	)
	require.NoError(t, app.Save(containers))

	checks := core.NewBaseCollection("health_checks")
	checks.Fields.Add(
		&core.TextField{Name: "name"},
		&core.RelationField{Name: "server", CollectionId: servers.Id, MaxSelect: 1},
		&core.RelationField{Name: "container", CollectionId: containers.Id, MaxSelect: 1},
		&core.TextField{Name: "target_type"},
		&core.TextField{Name: "app_source"},
		&core.TextField{Name: "app_name"},
		&core.TextField{Name: "check_type"},
		&core.TextField{Name: "host"},
		&core.TextField{Name: "scheme"},
		&core.TextField{Name: "method"},
		&core.TextField{Name: "url_path"},
		&core.NumberField{Name: "port"},
		&core.NumberField{Name: "timeout_seconds"},
		&core.NumberField{Name: "expected_status"},
		&core.TextField{Name: "expected_body"},
		&core.JSONField{Name: "headers"},
		&core.BoolField{Name: "tls_skip_verify"},
		&core.NumberField{Name: "success_threshold"},
		&core.NumberField{Name: "failure_threshold"},
		&core.NumberField{Name: "consecutive_successes"},
		&core.NumberField{Name: "consecutive_failures"},
		&core.TextField{Name: "status"},
		&core.BoolField{Name: "enabled"},
		&core.BoolField{Name: "auto_restart"},
		&core.NumberField{Name: "interval_seconds"},
		&core.NumberField{Name: "max_restarts"},
		&core.NumberField{Name: "restart_window_minutes"},
		&core.DateField{Name: "last_check_at"},
		&core.DateField{Name: "next_check_at"},
	)
	require.NoError(t, app.Save(checks))

	logs := core.NewBaseCollection("health_check_logs")
	logs.Fields.Add(
		&core.RelationField{Name: "health_check", CollectionId: checks.Id, MaxSelect: 1},
		&core.NumberField{Name: "status_code"},
		&core.NumberField{Name: "response_time_ms"},
		&core.BoolField{Name: "success"},
		&core.TextField{Name: "error"},
		&core.DateField{Name: "checked_at"},
	)
	require.NoError(t, app.Save(logs))

	server := core.NewRecord(servers)
	server.Set("name", "host-a")
	server.Set("status", "online")
	require.NoError(t, app.Save(server))

	return app, server.Id
}

func newHostAppHealthCheck(t *testing.T, app *tests.TestApp, serverID string) *core.Record {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId("health_checks")
	require.NoError(t, err)
	record := core.NewRecord(collection)
	record.Set("name", "nginx http")
	record.Set("server", serverID)
	record.Set("target_type", "host_app")
	record.Set("app_source", "systemd")
	record.Set("app_name", "nginx.service")
	record.Set("check_type", "http")
	record.Set("host", "127.0.0.1")
	record.Set("scheme", "http")
	record.Set("method", "GET")
	record.Set("url_path", "/health")
	record.Set("port", 8080)
	record.Set("timeout_seconds", 5)
	record.Set("expected_status", 200)
	record.Set("expected_body", "ok")
	record.Set("headers", map[string]any{"X-Test": "1"})
	record.Set("tls_skip_verify", true)
	record.Set("success_threshold", 2)
	record.Set("failure_threshold", 2)
	record.Set("status", "unknown")
	record.Set("consecutive_successes", 0)
	record.Set("consecutive_failures", 0)
	record.Set("enabled", true)
	record.Set("interval_seconds", 30)
	require.NoError(t, app.Save(record))
	return record
}

func TestHealthSchedulerHostAppCommand(t *testing.T) {
	app, serverID := newHealthSchedulerTestApp(t)
	check := newHostAppHealthCheck(t, app, serverID)

	command := buildHostHealthCommand(check, 5)
	require.Nil(t, command.GetHealthCheck())

	host := command.GetHostHealthCheck()
	require.NotNil(t, host)
	require.Equal(t, "systemd", host.GetSource())
	require.Equal(t, "nginx.service", host.GetName())
	require.Equal(t, "http", host.GetCheckType())
	require.Equal(t, "127.0.0.1", host.GetHost())
	require.Equal(t, uint32(8080), host.GetPort())
	require.Equal(t, "http", host.GetScheme())
	require.Equal(t, "GET", host.GetMethod())
	require.Equal(t, "/health", host.GetPath())
	require.Equal(t, uint32(5000), host.GetTimeoutMs())
	require.Equal(t, int32(200), host.GetExpectedStatus())
	require.Equal(t, "ok", host.GetExpectedBody())
	require.Equal(t, "1", host.GetHeaders()["X-Test"])
	require.True(t, host.GetTlsSkipVerify())
}

func TestSuccessThresholdDelaysHealthy(t *testing.T) {
	app, serverID := newHealthSchedulerTestApp(t)
	check := newHostAppHealthCheck(t, app, serverID)

	bus := events.NewBus()
	defer bus.Close()
	scheduler := NewHealthScheduler(app, store.NewHealthStore(app), nil, bus)

	scheduler.handleSuccess(check, check.Id)
	reloaded, err := app.FindRecordById("health_checks", check.Id)
	require.NoError(t, err)
	require.Equal(t, "unknown", reloaded.GetString("status"))
	require.Equal(t, 1, reloaded.GetInt("consecutive_successes"))

	scheduler.handleSuccess(reloaded, check.Id)
	reloaded, err = app.FindRecordById("health_checks", check.Id)
	require.NoError(t, err)
	require.Equal(t, "healthy", reloaded.GetString("status"))
	require.Equal(t, 0, reloaded.GetInt("consecutive_successes"))
}

func TestHostAppFailureUsesHostAppEvent(t *testing.T) {
	app, serverID := newHealthSchedulerTestApp(t)
	check := newHostAppHealthCheck(t, app, serverID)
	check.Set("failure_threshold", 1)
	check.Set("status", "healthy")
	check.Set("auto_restart", false)
	require.NoError(t, app.Save(check))

	bus := events.NewBus()
	defer bus.Close()

	var published []events.Event
	bus.Subscribe(events.HandlerFunc(func(event events.Event) {
		if event.Type == events.HostAppHealthDegraded {
			published = append(published, event)
		}
	}))

	scheduler := NewHealthScheduler(app, store.NewHealthStore(app), nil, bus)
	scheduler.handleFailure(context.Background(), check, check.Id, 0, 0, "boom")

	bus.Close()
	require.Len(t, published, 1)
	require.Equal(t, "systemd", published[0].Details["source"])
	require.Equal(t, "nginx.service", published[0].Details["name"])
}
