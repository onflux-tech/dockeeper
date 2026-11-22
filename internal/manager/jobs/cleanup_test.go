package jobs

import (
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func TestCleanupRemovesOldHostAppMetrics(t *testing.T) {
	app, err := tests.NewTestApp()
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)

	col := core.NewBaseCollection("host_app_metrics_raw")
	col.Fields.Add(
		&core.TextField{Name: "app"},
		&core.DateField{Name: "timestamp"},
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	require.NoError(t, app.Save(col))

	current := core.NewRecord(col)
	current.Set("app", "current")
	require.NoError(t, app.Save(current))

	old := core.NewRecord(col)
	old.Set("app", "old")
	require.NoError(t, app.Save(old))

	cutoff := time.Now().UTC().Add(-8 * 24 * time.Hour).Format("2006-01-02 15:04:05.000Z")
	_, err = app.DB().NewQuery("UPDATE host_app_metrics_raw SET created = {:cutoff} WHERE id = {:id}").
		Bind(dbx.Params{"cutoff": cutoff, "id": old.Id}).Execute()
	require.NoError(t, err)

	CleanupOldMetrics(app)

	rows, err := app.FindRecordsByFilter("host_app_metrics_raw", "", "", 0, 0)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "current", rows[0].GetString("app"))
}
