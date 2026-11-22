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

func TestGetS3SettingsNormalizesEndpoint(t *testing.T) {
	app, err := tests.NewTestApp()
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)

	settings, err := app.FindCollectionByNameOrId("settings")
	if err != nil {
		settings = core.NewBaseCollection("settings")
		settings.Fields.Add(&core.TextField{Name: "key"}, &core.TextField{Name: "value"})
		require.NoError(t, app.Save(settings))
	}

	values := map[string]string{
		"s3_endpoint":   "http://172.17.0.1:9000",
		"s3_access_key": "access",
		"s3_secret_key": "secret",
		"s3_region":     "us-east-1",
		"s3_use_ssl":    "true",
	}
	for key, value := range values {
		record := core.NewRecord(settings)
		record.Set("key", key)
		record.Set("value", value)
		require.NoError(t, app.Save(record))
	}

	service := &BackupService{app: app}
	config, err := service.getS3Settings()
	require.NoError(t, err)
	require.Equal(t, "172.17.0.1:9000", config.Endpoint)
	require.False(t, config.UseSSL)
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
