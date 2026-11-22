package services

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"

	"github.com/onflux-tech/dockeeper/internal/manager/events"
	"github.com/onflux-tech/dockeeper/internal/manager/store"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

type fakeHostDispatcher struct {
	mu        sync.Mutex
	serverID  string
	command   *agentpb.ChannelCommand
	result    *agentpb.ChannelCommandResult
	err       error
	entered   chan struct{}
	enterOnce sync.Once
	release   chan struct{}
	calls     atomic.Int32
}

func (f *fakeHostDispatcher) SendHostCommand(ctx context.Context, serverID string, command *agentpb.ChannelCommand) (*agentpb.ChannelCommandResult, error) {
	f.calls.Add(1)
	f.mu.Lock()
	f.serverID = serverID
	f.command = command
	f.mu.Unlock()
	if f.entered != nil {
		f.enterOnce.Do(func() { close(f.entered) })
	}
	if f.release != nil {
		select {
		case <-f.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return f.result, f.err
}

func newHostActionTestApp(t *testing.T) (*tests.TestApp, string) {
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
	)
	require.NoError(t, app.Save(servers))

	hostApps := core.NewBaseCollection("host_apps")
	hostApps.Fields.Add(
		&core.RelationField{Name: "server", CollectionId: servers.Id, MaxSelect: 1},
		&core.TextField{Name: "source"},
		&core.TextField{Name: "name"},
		&core.TextField{Name: "user"},
		&core.TextField{Name: "status"},
		&core.DateField{Name: "last_seen"},
	)
	require.NoError(t, app.Save(hostApps))

	auditLogs := core.NewBaseCollection("audit_logs")
	auditLogs.Fields.Add(
		&core.TextField{Name: "action"},
		&core.TextField{Name: "user"},
		&core.TextField{Name: "target_type"},
		&core.TextField{Name: "target_id"},
		&core.TextField{Name: "ip_address"},
		&core.JSONField{Name: "details"},
		&core.DateField{Name: "timestamp"},
	)
	require.NoError(t, app.Save(auditLogs))

	server := core.NewRecord(servers)
	server.Set("name", "srv1")
	server.Set("status", "online")
	require.NoError(t, app.Save(server))

	return app, server.Id
}

func addHostApp(t *testing.T, app *tests.TestApp, serverID, source, name, user string) *core.Record {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId("host_apps")
	require.NoError(t, err)
	record := core.NewRecord(collection)
	record.Set("server", serverID)
	record.Set("source", source)
	record.Set("name", name)
	record.Set("user", user)
	record.Set("status", "stopped")
	require.NoError(t, app.Save(record))
	return record
}

func TestHostActionRejectsUnknownApp(t *testing.T) {
	app, serverID := newHostActionTestApp(t)
	dispatcher := &fakeHostDispatcher{}
	svc := NewHostActionService(app, dispatcher, nil, nil)

	_, err := svc.Execute(context.Background(), serverID, "systemd", "nginx.service", "restart", "user1", "127.0.0.1")
	require.ErrorIs(t, err, ErrHostAppNotFound)
	require.Zero(t, dispatcher.calls.Load())
}

func TestHostActionRejectsInvalidInput(t *testing.T) {
	app, serverID := newHostActionTestApp(t)
	svc := NewHostActionService(app, &fakeHostDispatcher{}, nil, nil)

	_, err := svc.Execute(context.Background(), serverID, "shell", "nginx", "restart", "user1", "127.0.0.1")
	require.ErrorIs(t, err, ErrInvalidHostAction)

	_, err = svc.Execute(context.Background(), serverID, "systemd", "nginx.service", "reload", "user1", "127.0.0.1")
	require.ErrorIs(t, err, ErrInvalidHostAction)
}

func TestHostActionSendsCommand(t *testing.T) {
	app, serverID := newHostActionTestApp(t)
	hostApp := addHostApp(t, app, serverID, "systemd", "nginx.service", "root")

	dispatcher := &fakeHostDispatcher{
		result: &agentpb.ChannelCommandResult{
			Result: &agentpb.ChannelCommandResult_HostAppAction{
				HostAppAction: &agentpb.HostAppActionResult{Success: true, Output: "restarted"},
			},
		},
	}
	bus := events.NewBus()
	defer bus.Close()

	var published []events.Event
	bus.Subscribe(events.HandlerFunc(func(event events.Event) {
		if event.Type == events.HostAppAction {
			published = append(published, event)
		}
	}))

	svc := NewHostActionService(app, dispatcher, bus, store.NewAuditStore(app))
	output, err := svc.Execute(context.Background(), serverID, "systemd", "nginx.service", "restart", "user1", "127.0.0.1")
	require.NoError(t, err)
	require.Equal(t, "restarted", output)
	require.Equal(t, serverID, dispatcher.serverID)
	require.Equal(t, "systemd", dispatcher.command.GetHostAppAction().GetSource())
	require.Equal(t, "nginx.service", dispatcher.command.GetHostAppAction().GetName())
	require.Equal(t, "restart", dispatcher.command.GetHostAppAction().GetAction())

	reloaded, err := app.FindRecordById("host_apps", hostApp.Id)
	require.NoError(t, err)
	require.Equal(t, "running", reloaded.GetString("status"))

	auditRows, err := app.FindRecordsByFilter("audit_logs", "action = 'host_app.restart'", "", 0, 0)
	require.NoError(t, err)
	require.Len(t, auditRows, 1)
	require.Equal(t, "user1", auditRows[0].GetString("user"))
	require.Equal(t, "host_app", auditRows[0].GetString("target_type"))

	bus.Close()
	require.Len(t, published, 1)
	require.Equal(t, "nginx.service", published[0].Details["name"])
}

func TestHostActionRejectsOfflineServer(t *testing.T) {
	app, serverID := newHostActionTestApp(t)
	addHostApp(t, app, serverID, "systemd", "nginx.service", "root")

	dispatcher := &fakeHostDispatcher{err: ErrServerOffline}
	svc := NewHostActionService(app, dispatcher, nil, nil)

	_, err := svc.Execute(context.Background(), serverID, "systemd", "nginx.service", "restart", "user1", "127.0.0.1")
	require.ErrorIs(t, err, ErrServerOffline)
}

func TestHostActionRejectsConcurrentAction(t *testing.T) {
	app, serverID := newHostActionTestApp(t)
	addHostApp(t, app, serverID, "systemd", "nginx.service", "root")

	dispatcher := &fakeHostDispatcher{
		entered: make(chan struct{}),
		release: make(chan struct{}),
		result: &agentpb.ChannelCommandResult{
			Result: &agentpb.ChannelCommandResult_HostAppAction{
				HostAppAction: &agentpb.HostAppActionResult{Success: true},
			},
		},
	}
	svc := NewHostActionService(app, dispatcher, nil, nil)

	firstDone := make(chan error, 1)
	go func() {
		_, err := svc.Execute(context.Background(), serverID, "systemd", "nginx.service", "restart", "user1", "127.0.0.1")
		firstDone <- err
	}()
	<-dispatcher.entered

	_, err := svc.Execute(context.Background(), serverID, "systemd", "nginx.service", "stop", "user2", "127.0.0.1")
	require.ErrorIs(t, err, ErrHostActionInFlight)

	close(dispatcher.release)
	require.NoError(t, <-firstDone)
	require.Equal(t, int32(1), dispatcher.calls.Load())

	time.Sleep(10 * time.Millisecond)
	_, err = svc.Execute(context.Background(), serverID, "systemd", "nginx.service", "start", "user1", "127.0.0.1")
	require.NoError(t, err)
}
