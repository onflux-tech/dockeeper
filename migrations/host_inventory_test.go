package migrations

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func TestHostInventoryMigration(t *testing.T) {
	app, err := tests.NewTestApp()
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)

	for _, name := range []string{"host_apps", "host_processes", "host_app_metrics_raw"} {
		_, err := app.FindCollectionByNameOrId(name)
		require.NoError(t, err)
	}

	saved, err := app.FindCollectionByNameOrId("servers")
	require.NoError(t, err)
	field, ok := saved.Fields.GetByName("orchestrator").(*core.SelectField)
	require.True(t, ok)
	require.Contains(t, field.Values, "none")

	hostApps, err := app.FindCollectionByNameOrId("host_apps")
	require.NoError(t, err)
	require.NotEmpty(t, hostApps.Indexes)
}
