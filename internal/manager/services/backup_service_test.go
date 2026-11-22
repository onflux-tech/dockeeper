package services

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
