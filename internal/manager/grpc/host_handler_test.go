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

func newHostHandlerTestApp(t *testing.T) (*tests.TestApp, string) {
	t.Helper()

	app, err := tests.NewTestApp()
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)

	servers := core.NewBaseCollection("servers")
	servers.Fields.Add(
		&core.TextField{Name: "name"},
		&core.TextField{Name: "status"},
		&core.DateField{Name: "last_seen"},
	)
	require.NoError(t, app.Save(servers))

	hostApps := core.NewBaseCollection("host_apps")
	hostApps.Fields.Add(
		&core.RelationField{Name: "server", CollectionId: servers.Id, MaxSelect: 1},
		&core.TextField{Name: "source"},
		&core.TextField{Name: "name"},
		&core.TextField{Name: "unit"},
		&core.TextField{Name: "pm_id"},
		&core.TextField{Name: "status"},
		&core.JSONField{Name: "pids"},
		&core.JSONField{Name: "ports"},
		&core.NumberField{Name: "cpu_percent"},
		&core.NumberField{Name: "mem_bytes"},
		&core.NumberField{Name: "disk_read_bytes"},
		&core.NumberField{Name: "disk_write_bytes"},
		&core.NumberField{Name: "restarts"},
		&core.NumberField{Name: "uptime_seconds"},
		&core.TextField{Name: "user"},
		&core.TextField{Name: "cwd"},
		&core.TextField{Name: "exec_path"},
		&core.TextField{Name: "log_path"},
		&core.DateField{Name: "last_seen"},
	)
	require.NoError(t, app.Save(hostApps))

	hostProcesses := core.NewBaseCollection("host_processes")
	hostProcesses.Fields.Add(
		&core.RelationField{Name: "server", CollectionId: servers.Id, MaxSelect: 1},
		&core.NumberField{Name: "pid"},
		&core.TextField{Name: "name"},
		&core.TextField{Name: "app"},
		&core.NumberField{Name: "cpu_percent"},
		&core.NumberField{Name: "mem_bytes"},
		&core.NumberField{Name: "threads"},
		&core.DateField{Name: "last_seen"},
	)
	require.NoError(t, app.Save(hostProcesses))

	hostAppMetrics := core.NewBaseCollection("host_app_metrics_raw")
	hostAppMetrics.Fields.Add(
		&core.RelationField{Name: "server", CollectionId: servers.Id, MaxSelect: 1},
		&core.TextField{Name: "app"},
		&core.TextField{Name: "source"},
		&core.DateField{Name: "timestamp"},
		&core.NumberField{Name: "cpu_percent"},
		&core.NumberField{Name: "mem_bytes"},
		&core.NumberField{Name: "restarts"},
	)
	require.NoError(t, app.Save(hostAppMetrics))

	server := core.NewRecord(servers)
	server.Set("name", "host-a")
	server.Set("status", "online")
	require.NoError(t, app.Save(server))

	return app, server.Id
}

func TestReportHostInventoryDoesNotPublishPerReport(t *testing.T) {
	app, serverID := newHostHandlerTestApp(t)
	bus := events.NewBus()
	defer bus.Close()

	var count int
	bus.Subscribe(events.HandlerFunc(func(event events.Event) {
		if event.Type == events.ServerHostInventory {
			count++
		}
	}))

	handler := NewHostHandler(app, store.NewHostStore(app), bus)
	ctx := context.WithValue(context.Background(), serverIDKey, serverID)
	request := &agentpb.ReportHostInventoryRequest{}

	for range 2 {
		_, err := handler.ReportHostInventory(ctx, request)
		require.NoError(t, err)
	}
	bus.Close()

	require.Zero(t, count)
}

func TestReportHostInventoryPublishesOnChange(t *testing.T) {
	app, serverID := newHostHandlerTestApp(t)
	bus := events.NewBus()
	defer bus.Close()

	var count int
	bus.Subscribe(events.HandlerFunc(func(event events.Event) {
		if event.Type == events.ServerHostInventory {
			count++
		}
	}))

	handler := NewHostHandler(app, store.NewHostStore(app), bus)
	ctx := context.WithValue(context.Background(), serverIDKey, serverID)
	request := &agentpb.ReportHostInventoryRequest{
		Apps: []*agentpb.HostApp{{Source: "systemd", Name: "nginx.service", Status: "running"}},
	}

	for range 2 {
		_, err := handler.ReportHostInventory(ctx, request)
		require.NoError(t, err)
	}
	bus.Close()

	require.Equal(t, 1, count)
}
