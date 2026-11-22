package store

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func TestCreateBackupRecordSetsVolumeTarget(t *testing.T) {
	app, err := tests.NewTestApp()
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)

	records := core.NewBaseCollection("backup_records")
	records.Fields.Add(
		&core.TextField{Name: "policy"},
		&core.TextField{Name: "volume_name"},
		&core.TextField{Name: "server"},
		&core.SelectField{Name: "target_type", Values: []string{"volume", "host_app"}, MaxSelect: 1},
		&core.TextField{Name: "target_label"},
		&core.TextField{Name: "status"},
		&core.DateField{Name: "started_at"},
	)
	require.NoError(t, app.Save(records))

	s := NewBackupStore(app)
	id, err := s.CreateBackupRecord("pol1", "data", "srv1")
	require.NoError(t, err)

	reloaded, err := app.FindRecordById("backup_records", id)
	require.NoError(t, err)
	require.Equal(t, "volume", reloaded.GetString("target_type"))
	require.Equal(t, "data", reloaded.GetString("target_label"))
	require.Equal(t, "running", reloaded.GetString("status"))
}

func TestMarkBackupSuccessStoresChecksum(t *testing.T) {
	app, err := tests.NewTestApp()
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)

	records := core.NewBaseCollection("backup_records")
	records.Fields.Add(
		&core.TextField{Name: "status"},
		&core.TextField{Name: "s3_key"},
		&core.NumberField{Name: "size", OnlyInt: true},
		&core.DateField{Name: "completed_at"},
		&core.JSONField{Name: "manifest"},
	)
	require.NoError(t, app.Save(records))

	record := core.NewRecord(records)
	record.Set("status", "running")
	require.NoError(t, app.Save(record))

	s := NewBackupStore(app)
	require.NoError(t, s.MarkBackupSuccess(record.Id, "backups/app.tar.gz", 42, "checksum-abc"))

	reloaded, err := app.FindRecordById("backup_records", record.Id)
	require.NoError(t, err)
	require.Equal(t, "success", reloaded.GetString("status"))
	require.Equal(t, "backups/app.tar.gz", reloaded.GetString("s3_key"))
	require.EqualValues(t, 42, reloaded.GetInt("size"))

	var manifest map[string]any
	require.NoError(t, reloaded.UnmarshalJSONField("manifest", &manifest))
	require.Equal(t, "checksum-abc", manifest["checksum"])
	require.EqualValues(t, 42, manifest["size"])
}
