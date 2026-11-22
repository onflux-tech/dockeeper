//go:build !windows

package executor

import (
	"archive/tar"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRestoreOwnerAppliesCurrentOwner(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "app.conf")
	require.NoError(t, os.WriteFile(target, []byte("ok"), 0o600))

	require.NoError(t, restoreOwner(target, &tar.Header{Uid: os.Getuid(), Gid: os.Getgid()}, false))
}
