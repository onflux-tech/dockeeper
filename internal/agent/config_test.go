package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, "./dk-agent-data", cfg.DataDir)
	assert.Empty(t, cfg.ManagerURL)
	assert.Empty(t, cfg.ServerID)
}

func TestConfigPaths(t *testing.T) {
	cfg := &Config{DataDir: "/tmp/test-agent"}
	assert.Equal(t, filepath.Join("/tmp/test-agent", "agent.yaml"), cfg.ConfigPath())
	assert.Equal(t, filepath.Join("/tmp/test-agent", "certs"), cfg.CertsDir())
}

func TestSaveAndLoadConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		ManagerURL: "localhost:50051",
		DataDir:    dir,
		ServerID:   "srv_abc123",
	}

	require.NoError(t, cfg.Save())

	loaded, err := LoadConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, "localhost:50051", loaded.ManagerURL)
	assert.Equal(t, "srv_abc123", loaded.ServerID)
	assert.Equal(t, dir, loaded.DataDir)
}

func TestLoadConfigMissingFile(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, dir, cfg.DataDir)
	assert.Empty(t, cfg.ServerID)
}

func TestClearEnrollment(t *testing.T) {
	dir := t.TempDir()

	// Create fake cert files.
	certsDir := filepath.Join(dir, "certs")
	require.NoError(t, os.MkdirAll(certsDir, 0o700))
	for _, name := range []string{"ca.pem", "client.pem", "client-key.pem"} {
		require.NoError(t, os.WriteFile(filepath.Join(certsDir, name), []byte("fake"), 0o600))
	}

	cfg := &Config{
		ManagerURL: "localhost:50051",
		DataDir:    dir,
		ServerID:   "srv_abc123",
	}
	require.NoError(t, cfg.Save())

	// Clear enrollment.
	require.NoError(t, cfg.ClearEnrollment())

	// ServerID should be empty.
	assert.Empty(t, cfg.ServerID)

	// Config file should be updated on disk.
	reloaded, err := LoadConfig(dir)
	require.NoError(t, err)
	assert.Empty(t, reloaded.ServerID)
	assert.Equal(t, "localhost:50051", reloaded.ManagerURL)

	// Cert files should be removed.
	for _, name := range []string{"ca.pem", "client.pem", "client-key.pem"} {
		_, err := os.Stat(filepath.Join(certsDir, name))
		assert.True(t, os.IsNotExist(err), "cert file %s should be removed", name)
	}
}

func TestClearEnrollmentMissingCerts(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		DataDir:  dir,
		ServerID: "srv_test",
	}
	require.NoError(t, cfg.Save())

	// Should not error if cert files don't exist.
	require.NoError(t, cfg.ClearEnrollment())
	assert.Empty(t, cfg.ServerID)
}
