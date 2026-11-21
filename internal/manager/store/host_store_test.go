package store

import (
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"

	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

func newHostTestApp(t *testing.T) (*tests.TestApp, string) {
	t.Helper()

	app, err := tests.NewTestApp()
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)

	servers := core.NewBaseCollection("servers")
	servers.Fields.Add(
		&core.TextField{Name: "name"},
		&core.SelectField{
			Name:      "status",
			Values:    []string{"online", "offline", "error"},
			MaxSelect: 1,
		},
	)
	require.NoError(t, app.Save(servers))

	hostApps := core.NewBaseCollection("host_apps")
	hostApps.Fields.Add(
		&core.RelationField{Name: "server", CollectionId: servers.Id, MaxSelect: 1},
		&core.SelectField{
			Name:      "source",
			Values:    []string{"systemd", "pm2", "process"},
			MaxSelect: 1,
		},
		&core.TextField{Name: "name"},
		&core.TextField{Name: "unit"},
		&core.TextField{Name: "pm_id"},
		&core.SelectField{
			Name:      "status",
			Values:    []string{"running", "stopped", "failed", "unknown"},
			MaxSelect: 1,
		},
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

func TestHostStoreUpsertAndPrune(t *testing.T) {
	app, serverID := newHostTestApp(t)
	store := NewHostStore(app)

	count, err := store.UpsertInventory(serverID, []*agentpb.HostApp{
		{Source: "systemd", Name: "nginx.service", Status: "running"},
	}, nil, nil)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	_, err = store.UpsertInventory(serverID, []*agentpb.HostApp{
		{Source: "pm2", Name: "api", Status: "running"},
	}, nil, nil)
	require.NoError(t, err)

	rows, err := app.FindRecordsByFilter("host_apps", "server = {:server}", "", 0, 0, map[string]any{"server": serverID})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "api", rows[0].GetString("name"))
}

func TestHostStoreUpsertsProcesses(t *testing.T) {
	app, serverID := newHostTestApp(t)
	store := NewHostStore(app)

	_, err := store.UpsertInventory(serverID, nil, []*agentpb.HostProcess{
		{Pid: 4122, Name: "node", CpuPercent: 8.2, MemBytes: 1024, Threads: 11},
	}, nil)
	require.NoError(t, err)

	_, err = store.UpsertInventory(serverID, nil, []*agentpb.HostProcess{
		{Pid: 4122, Name: "node", CpuPercent: 12.5, MemBytes: 2048, Threads: 12},
	}, nil)
	require.NoError(t, err)

	rows, err := app.FindRecordsByFilter("host_processes", "server = {:server}", "", 0, 0, map[string]any{"server": serverID})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, 12.5, rows[0].GetFloat("cpu_percent"))
}

func TestHostStoreSaveAppMetrics(t *testing.T) {
	app, serverID := newHostTestApp(t)
	store := NewHostStore(app)

	at := time.Now().UTC().Truncate(time.Second)
	err := store.SaveAppMetrics(serverID, []*agentpb.HostApp{
		{Source: "systemd", Name: "nginx.service", CpuPercent: 12.5, MemBytes: 1024, Restarts: 2},
	}, at)
	require.NoError(t, err)

	rows, err := app.FindRecordsByFilter("host_app_metrics_raw", "server = {:server}", "", 0, 0, map[string]any{"server": serverID})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "nginx.service", rows[0].GetString("app"))
	require.Equal(t, 12.5, rows[0].GetFloat("cpu_percent"))
}
