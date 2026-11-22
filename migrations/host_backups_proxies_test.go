package migrations

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func TestHostBackupsProxiesMigration(t *testing.T) {
	app, err := tests.NewTestApp()
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)

	policies, err := app.FindCollectionByNameOrId("backup_policies")
	require.NoError(t, err)

	volume, ok := policies.Fields.GetByName("volume").(*core.TextField)
	require.True(t, ok)
	require.False(t, volume.Required)

	for _, name := range []string{"target_type", "app_source", "app_name", "paths", "exclude", "max_bytes"} {
		require.NotNil(t, policies.Fields.GetByName(name), name)
	}

	targetType, ok := policies.Fields.GetByName("target_type").(*core.SelectField)
	require.True(t, ok)
	require.ElementsMatch(t, []string{"volume", "host_app"}, targetType.Values)

	records, err := app.FindCollectionByNameOrId("backup_records")
	require.NoError(t, err)
	for _, name := range []string{"target_type", "app_source", "app_name", "target_label", "manifest"} {
		require.NotNil(t, records.Fields.GetByName(name), name)
	}

	instances, err := app.FindCollectionByNameOrId("proxy_instances")
	require.NoError(t, err)
	for _, name := range []string{
		"server", "kind", "source", "app", "version", "config_path", "admin_url", "status", "last_seen",
	} {
		require.NotNil(t, instances.Fields.GetByName(name), name)
	}
	instanceIndexes := strings.Join(instances.Indexes, " ")
	require.Contains(t, instanceIndexes, "idx_proxy_instances_server")
	require.Contains(t, instanceIndexes, "idx_proxy_instances_unique")

	routes, err := app.FindCollectionByNameOrId("proxy_routes")
	require.NoError(t, err)
	for _, name := range []string{
		"instance", "server", "name", "hosts", "paths", "upstreams", "tls", "middlewares", "status", "provider",
	} {
		require.NotNil(t, routes.Fields.GetByName(name), name)
	}
	routeIndexes := strings.Join(routes.Indexes, " ")
	require.Contains(t, routeIndexes, "idx_proxy_routes_server")
	require.Contains(t, routeIndexes, "idx_proxy_routes_instance")
}

func TestHostBackupsProxiesBackfill(t *testing.T) {
	app, err := tests.NewTestApp()
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)

	servers, err := app.FindCollectionByNameOrId("servers")
	require.NoError(t, err)
	server := core.NewRecord(servers)
	server.Set("name", "srv1")
	server.Set("status", "online")
	require.NoError(t, app.Save(server))

	policies, err := app.FindCollectionByNameOrId("backup_policies")
	require.NoError(t, err)
	policy := core.NewRecord(policies)
	policy.Set("volume", "data")
	policy.Set("server", server.Id)
	policy.Set("target_type", "")
	require.NoError(t, app.Save(policy))

	require.NoError(t, backfillBackupPolicyTargetTypes(app))

	reloaded, err := app.FindRecordById("backup_policies", policy.Id)
	require.NoError(t, err)
	require.Equal(t, "volume", reloaded.GetString("target_type"))
}
