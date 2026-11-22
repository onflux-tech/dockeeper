package migrations

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func TestHostActionsHealthMigration(t *testing.T) {
	app, err := tests.NewTestApp()
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)

	healthChecks, err := app.FindCollectionByNameOrId("health_checks")
	require.NoError(t, err)

	container, ok := healthChecks.Fields.GetByName("container").(*core.RelationField)
	require.True(t, ok)
	require.False(t, container.Required)

	for _, name := range []string{
		"target_type", "app_source", "app_name", "check_type", "host", "scheme",
		"method", "expected_body", "headers", "tls_skip_verify", "consecutive_successes",
	} {
		require.NotNil(t, healthChecks.Fields.GetByName(name), name)
	}

	indexes := strings.Join(healthChecks.Indexes, " ")
	require.Contains(t, indexes, "idx_health_checks_target_type")
	require.Contains(t, indexes, "idx_health_checks_server_app")

	targetType, ok := healthChecks.Fields.GetByName("target_type").(*core.SelectField)
	require.True(t, ok)
	require.ElementsMatch(t, []string{"container", "host_app"}, targetType.Values)

	_, err = app.FindCollectionByNameOrId("server_connectivity_raw")
	require.NoError(t, err)

	servers, err := app.FindCollectionByNameOrId("servers")
	require.NoError(t, err)
	viewScope, ok := servers.Fields.GetByName("view_scope").(*core.SelectField)
	require.True(t, ok)
	require.ElementsMatch(t, []string{"auto", "server", "docker", "both"}, viewScope.Values)
	for _, name := range []string{
		"connectivity_status", "connectivity_checked_at", "connectivity_latency_ms", "connectivity_details",
	} {
		require.NotNil(t, servers.Fields.GetByName(name), name)
	}

	alerts, err := app.FindCollectionByNameOrId("alert_rules")
	require.NoError(t, err)
	alertTarget, ok := alerts.Fields.GetByName("target_type").(*core.SelectField)
	require.True(t, ok)
	require.Contains(t, alertTarget.Values, "host_app")
}

func TestHostActionsHealthBackfill(t *testing.T) {
	app, err := tests.NewTestApp()
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)

	servers, err := app.FindCollectionByNameOrId("servers")
	require.NoError(t, err)
	server := core.NewRecord(servers)
	server.Set("name", "legacy")
	server.Set("status", "online")
	require.NoError(t, app.Save(server))

	checks, err := app.FindCollectionByNameOrId("health_checks")
	require.NoError(t, err)
	check := core.NewRecord(checks)
	check.Set("name", "legacy-check")
	check.Set("server", server.Id)
	check.Set("url_path", "/")
	check.Set("port", 80)
	require.NoError(t, app.Save(check))

	check.Set("target_type", "")
	require.NoError(t, app.Save(check))

	require.NoError(t, backfillHealthCheckTargetTypes(app))

	reloaded, err := app.FindRecordById("health_checks", check.Id)
	require.NoError(t, err)
	require.Equal(t, "container", reloaded.GetString("target_type"))
}
