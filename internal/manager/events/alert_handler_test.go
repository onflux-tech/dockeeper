package events

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func newScopeRule(t *testing.T, scope string) *core.Record {
	t.Helper()

	app, err := tests.NewTestApp()
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)

	collection := core.NewBaseCollection("alert_rules")
	collection.Fields.Add(&core.TextField{Name: "scope"})
	require.NoError(t, app.Save(collection))

	record := core.NewRecord(collection)
	record.Set("scope", scope)
	return record
}

func TestMatchesScopeHostApp(t *testing.T) {
	rule := newScopeRule(t, "host_app:srv1:systemd:nginx.service")

	matching := Event{
		ServerID: "srv1",
		Details:  map[string]any{"source": "systemd", "name": "nginx.service"},
	}
	require.True(t, matchesScope(rule, matching))

	wrongName := Event{
		ServerID: "srv1",
		Details:  map[string]any{"source": "systemd", "name": "other.service"},
	}
	require.False(t, matchesScope(rule, wrongName))

	wrongServer := Event{
		ServerID: "srv2",
		Details:  map[string]any{"source": "systemd", "name": "nginx.service"},
	}
	require.False(t, matchesScope(rule, wrongServer))
}

func TestMatchesScopeConnectivity(t *testing.T) {
	rule := newScopeRule(t, "connectivity:srv1")

	require.True(t, matchesScope(rule, Event{ServerID: "srv1"}))
	require.False(t, matchesScope(rule, Event{ServerID: "srv2"}))
}

func TestResolutionEventsForHostAppAndConnectivity(t *testing.T) {
	require.True(t, isResolutionEvent(string(HostAppHealthDegraded), Event{Type: HostAppHealthRecovered}))
	require.True(t, isResolutionEvent(string(ServerConnectivityDegraded), Event{Type: ServerConnectivityRecovered}))
	require.False(t, isResolutionEvent(string(HostAppAction), Event{Type: HostAppHealthRecovered}))
}
