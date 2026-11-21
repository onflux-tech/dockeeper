package migrations

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func TestBumpTextLimits(t *testing.T) {
	app, err := tests.NewTestApp()
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)

	col := core.NewBaseCollection("custom_text_limits")
	col.Fields.Add(
		&core.TextField{Name: "long"},
		&core.TextField{Name: "short", Max: 255},
	)
	require.NoError(t, app.Save(col))

	require.NoError(t, bumpTextLimits(app))

	saved, err := app.FindCollectionByNameOrId("custom_text_limits")
	require.NoError(t, err)

	long, ok := saved.Fields.GetByName("long").(*core.TextField)
	require.True(t, ok)
	require.Equal(t, textFieldMax, long.Max)

	short, ok := saved.Fields.GetByName("short").(*core.TextField)
	require.True(t, ok)
	require.Equal(t, 255, short.Max)
}
