package executor

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

func TestAppBackupManifestAndPaths(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.conf"), []byte("ok"), 0o600))

	e := NewAppBackupExecutor(nil, nil)
	e.tempDir = t.TempDir()

	archive, manifest, err := e.buildArchive(context.Background(), Manifest{Paths: []string{dir}}, 1<<20)
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

	archive, _, err := e.buildArchive(context.Background(), Manifest{Paths: []string{dir}}, 1<<20)
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

	_, _, err := e.buildArchive(context.Background(), Manifest{Paths: []string{dir}}, 128)
	require.ErrorIs(t, err, ErrBackupTooLarge)
}

func TestAppBackupStopsOnCanceledContext(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.conf"), []byte("ok"), 0o600))

	e := NewAppBackupExecutor(nil, nil)
	e.tempDir = t.TempDir()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := e.buildArchive(ctx, Manifest{Paths: []string{dir}}, 1<<20)
	require.ErrorIs(t, err, context.Canceled)
}

func TestAppRestoreRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	require.NoError(t, os.Symlink(outside, filepath.Join(root, "link")))

	archive := writeTestArchive(t, []testArchiveEntry{
		{header: &tar.Header{Name: "files/0/link/evil.txt", Mode: 0o600, Typeflag: tar.TypeReg}, body: "evil"},
	})

	err := restoreArchive(archive, []string{root})
	require.ErrorIs(t, err, ErrUnsafePath)
	require.NoFileExists(t, filepath.Join(outside, "evil.txt"))
}

func TestAppRestoreRejectsSymlinkEntryEscape(t *testing.T) {
	root := t.TempDir()
	archive := writeTestArchive(t, []testArchiveEntry{
		{header: &tar.Header{Name: "files/0/escape", Typeflag: tar.TypeSymlink, Linkname: "../outside"}},
	})

	err := restoreArchive(archive, []string{root})
	require.ErrorIs(t, err, ErrUnsafePath)
	require.NoFileExists(t, filepath.Join(root, "escape"))
}

func TestAppSafetyArchiveRollbackUsesSafetyPaths(t *testing.T) {
	root := t.TempDir()
	existing := filepath.Join(root, "existing")
	missing := filepath.Join(root, "missing")
	require.NoError(t, os.MkdirAll(existing, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(existing, "app.conf"), []byte("keep"), 0o600))

	e := NewAppBackupExecutor(nil, nil)
	e.tempDir = t.TempDir()

	safety, safetyPaths, err := e.createSafetyArchive(context.Background(), []string{existing, missing})
	require.NoError(t, err)
	require.Equal(t, []string{existing}, safetyPaths)

	require.NoError(t, os.RemoveAll(existing))
	require.NoError(t, restoreArchive(safety, safetyPaths))

	body, err := os.ReadFile(filepath.Join(existing, "app.conf"))
	require.NoError(t, err)
	require.Equal(t, "keep", string(body))
	require.NoDirExists(t, missing)
}

func TestAppRestoreRejectsCrossServerManifest(t *testing.T) {
	req := &agentpb.RestoreAppRequest{ServerId: "srv1", AppSource: "systemd", AppName: "nginx.service"}

	require.NoError(t, validateManifestIdentity(&Manifest{Server: "srv1", Source: "systemd", App: "nginx.service"}, req))
	require.ErrorIs(t, validateManifestIdentity(&Manifest{Server: "srv2"}, req), ErrInvalidManifest)
	require.ErrorIs(t, validateManifestIdentity(&Manifest{App: "other.service"}, req), ErrInvalidManifest)
	require.ErrorIs(t, validateManifestIdentity(&Manifest{Source: "pm2"}, req), ErrInvalidManifest)
}

type testArchiveEntry struct {
	header *tar.Header
	body   string
}

func writeTestArchive(t *testing.T, entries []testArchiveEntry) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "archive.tar.gz")
	file, err := os.Create(path)
	require.NoError(t, err)

	gz := gzip.NewWriter(file)
	tw := tar.NewWriter(gz)
	for _, entry := range entries {
		header := *entry.header
		if header.Typeflag == tar.TypeReg {
			header.Size = int64(len(entry.body))
		}
		require.NoError(t, tw.WriteHeader(&header))
		if entry.body != "" {
			_, err := io.WriteString(tw, entry.body)
			require.NoError(t, err)
		}
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	require.NoError(t, file.Close())

	return path
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
