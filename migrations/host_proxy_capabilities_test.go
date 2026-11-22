package migrations

import (
	"testing"

	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func TestHostProxyCapabilitiesMigration(t *testing.T) {
	app, err := tests.NewTestApp()
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)

	for name, expected := range hostProxyCapabilities {
		group, err := app.FindFirstRecordByFilter("groups", "name = {:name}", map[string]any{"name": name})
		require.NoError(t, err)
		capabilities := group.GetStringSlice("capabilities")
		for _, capability := range expected {
			require.Contains(t, capabilities, capability, name)
		}
	}
}

func TestHostProxyCapabilitiesMigrationIsIdempotent(t *testing.T) {
	app, err := tests.NewTestApp()
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)

	require.NoError(t, migrateHostProxyCapabilities(app))

	group, err := app.FindFirstRecordByFilter("groups", "name = {:name}", map[string]any{"name": "Operators"})
	require.NoError(t, err)
	capabilities := group.GetStringSlice("capabilities")
	require.Equal(t, len(capabilities), len(uniqueStrings(capabilities)))
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
