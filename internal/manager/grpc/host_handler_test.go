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
	)
	require.NoError(t, app.Save(hostApps))

	hostProcesses := core.NewBaseCollection("host_processes")
	hostProcesses.Fields.Add(
		&core.RelationField{Name: "server", CollectionId: servers.Id, MaxSelect: 1},
		&core.NumberField{Name: "pid"},
	)
	require.NoError(t, app.Save(hostProcesses))

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
