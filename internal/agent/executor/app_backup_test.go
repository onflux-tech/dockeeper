package executor

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppBackupManifestAndPaths(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.conf"), []byte("ok"), 0o600))

	e := NewAppBackupExecutor(nil, nil)
	e.tempDir = t.TempDir()

	archive, manifest, err := e.buildArchive(Manifest{Paths: []string{dir}}, 1<<20)
	require.NoError(t, err)
	require.FileExists(t, archive)
	require.Equal(t, []string{dir}, manifest.Paths)
	require.NotEmpty(t, manifest.Checksum)

	stat, err := os.Stat(archive)
	require.NoError(t, err)
	require.NotZero(t, stat.Size())
}

func TestAppBackupArchiveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "app.conf")
	require.NoError(t, os.WriteFile(target, []byte("original"), 0o600))

	e := NewAppBackupExecutor(nil, nil)
	e.tempDir = t.TempDir()

	archive, _, err := e.buildArchive(Manifest{Paths: []string{dir}}, 1<<20)
	require.NoError(t, err)

	require.NoError(t, os.Remove(target))
	manifest, err := readManifest(archive)
	require.NoError(t, err)
	require.NoError(t, validateManifest(&manifest, []string{dir}))
	require.NoError(t, restoreArchive(archive, manifest.Paths))

	body, err := os.ReadFile(target)
	require.NoError(t, err)
	require.Equal(t, "original", string(body))
}

func TestAppRestoreRejectsTraversal(t *testing.T) {
	_, err := readManifestFromBytes(t, map[string]string{
		appBackupManifestName: `{"paths":["../etc"],"checksum":""}`,
	})
	require.ErrorIs(t, err, ErrInvalidManifest)

	require.ErrorIs(t, validateBackupPaths([]string{"../etc"}), ErrUnsafePath)
	require.ErrorIs(t, validateBackupPaths([]string{"/"}), ErrUnsafePath)
	require.ErrorIs(t, validateBackupPaths([]string{"/proc"}), ErrUnsafePath)
	require.ErrorIs(t, validateBackupPaths([]string{"/etc/shadow"}), ErrUnsafePath)
}

func TestAppBackupRejectsMissingAndDeniedPaths(t *testing.T) {
	require.ErrorIs(t, validateBackupPaths(nil), ErrNoBackupPaths)

	paths := make([]string, maxAppBackupPaths+1)
	for i := range paths {
		paths[i] = filepath.Join(t.TempDir(), "p")
	}
	require.ErrorIs(t, validateBackupPaths(paths), ErrTooManyBackupPaths)
}

func TestAppBackupMaxBytes(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "big.bin"), make([]byte, 4096), 0o600))

	e := NewAppBackupExecutor(nil, nil)
	e.tempDir = t.TempDir()

	_, _, err := e.buildArchive(Manifest{Paths: []string{dir}}, 128)
	require.ErrorIs(t, err, ErrBackupTooLarge)
}

func readManifestFromBytes(t *testing.T, entries map[string]string) (Manifest, error) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "archive.tar.gz")
	file, err := os.Create(path)
	require.NoError(t, err)

	gz := gzip.NewWriter(file)
	tw := tar.NewWriter(gz)
	for name, body := range entries {
		require.NoError(t, tw.WriteHeader(&tar.Header{
			Name: name,
			Mode: 0o600,
			Size: int64(len(body)),
		}))
		_, err := io.WriteString(tw, body)
		require.NoError(t, err)
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	require.NoError(t, file.Close())

	return readManifest(path)
}
