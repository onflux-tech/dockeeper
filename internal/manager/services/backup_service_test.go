package services

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

func TestRecordStringListReadsJSONField(t *testing.T) {
	app, err := tests.NewTestApp()
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)

	collection := core.NewBaseCollection("json_list_records")
	collection.Fields.Add(
		&core.JSONField{Name: "paths"},
		&core.JSONField{Name: "exclude"},
	)
	require.NoError(t, app.Save(collection))

	record := core.NewRecord(collection)
	record.Set("paths", []string{"/etc/nginx"})
	record.Set("exclude", []string{"*.log"})
	require.NoError(t, app.Save(record))

	reloaded, err := app.FindRecordById(collection.Id, record.Id)
	require.NoError(t, err)
	require.Equal(t, []string{"/etc/nginx"}, recordStringList(reloaded, "paths"))
	require.Equal(t, []string{"*.log"}, recordStringList(reloaded, "exclude"))
}

func TestValidateBackupPolicy(t *testing.T) {
	require.NoError(t, ValidateBackupPolicy("volume", "data", "", "", nil))
	require.NoError(t, ValidateBackupPolicy("host_app", "", "systemd", "nginx.service", []string{"/etc/nginx", "/var/www"}))

	require.ErrorIs(t, ValidateBackupPolicy("host_app", "", "systemd", "nginx.service", nil), ErrInvalidBackupTarget)
	require.ErrorIs(t, ValidateBackupPolicy("host_app", "", "", "nginx", []string{"/etc/nginx"}), ErrInvalidBackupTarget)
	require.ErrorIs(t, ValidateBackupPolicy("host_app", "", "systemd", "nginx", []string{"../etc"}), ErrInvalidBackupTarget)
	require.ErrorIs(t, ValidateBackupPolicy("host_app", "", "systemd", "nginx", []string{"/etc/shadow"}), ErrInvalidBackupTarget)
	require.ErrorIs(t, ValidateBackupPolicy("volume", "", "", "", nil), ErrInvalidBackupTarget)
	require.ErrorIs(t, ValidateBackupPolicy("bogus", "", "", "", nil), ErrInvalidBackupTarget)
}
