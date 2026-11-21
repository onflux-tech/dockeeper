package grpc

import (
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"

	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

func newEnrollmentTestApp(t *testing.T) (*tests.TestApp, *core.Collection) {
	t.Helper()

	app, err := tests.NewTestApp()
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)

	col := core.NewBaseCollection("servers")
	col.Fields.Add(
		&core.TextField{Name: "name"},
		&core.TextField{Name: "hostname"},
		&core.DateField{Name: "last_seen"},
	)
	require.NoError(t, app.Save(col))

	return app, col
}

func TestServerForEnrollmentReusesHostname(t *testing.T) {
	app, col := newEnrollmentTestApp(t)

	existing := core.NewRecord(col)
	existing.Set("name", "host-a")
	existing.Set("hostname", "host-a")
	existing.Set("last_seen", time.Now().UTC().Add(-time.Hour))
	require.NoError(t, app.Save(existing))

	h := &EnrollmentHandler{app: app}

	server, err := h.serverForEnrollment(&agentpb.ServerInfo{Hostname: "host-a"})
	require.NoError(t, err)
	require.Equal(t, existing.Id, server.Id)
}

func TestServerForEnrollmentCreatesNewForUnknownHostname(t *testing.T) {
	app, col := newEnrollmentTestApp(t)

	existing := core.NewRecord(col)
	existing.Set("name", "host-a")
	existing.Set("hostname", "host-a")
	require.NoError(t, app.Save(existing))

	h := &EnrollmentHandler{app: app}

	server, err := h.serverForEnrollment(&agentpb.ServerInfo{Hostname: "host-b"})
	require.NoError(t, err)
	require.NotEqual(t, existing.Id, server.Id)
	require.Equal(t, "host-b", server.GetString("name"))
}

func TestServerForEnrollmentPicksNewestDuplicate(t *testing.T) {
	app, col := newEnrollmentTestApp(t)

	older := core.NewRecord(col)
	older.Set("name", "host-a")
	older.Set("hostname", "host-a")
	older.Set("last_seen", time.Now().UTC().Add(-2*time.Hour))
	require.NoError(t, app.Save(older))

	newer := core.NewRecord(col)
	newer.Set("name", "host-a")
	newer.Set("hostname", "host-a")
	newer.Set("last_seen", time.Now().UTC().Add(-time.Minute))
	require.NoError(t, app.Save(newer))

	h := &EnrollmentHandler{app: app}

	server, err := h.serverForEnrollment(&agentpb.ServerInfo{Hostname: "host-a"})
	require.NoError(t, err)
	require.Equal(t, newer.Id, server.Id)
}
