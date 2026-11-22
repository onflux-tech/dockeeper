package executor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNoopExecutorReturnsSentinel(t *testing.T) {
	exec := NewNoopExecutor()

	result := exec.Start(context.Background(), "abc")
	require.False(t, result.Success)
	require.Contains(t, result.Error, ErrNoContainerRuntime.Error())

	_, err := exec.PullImageStream(context.Background(), "nginx")
	require.ErrorIs(t, err, ErrNoContainerRuntime)

	_, err = exec.SnapshotContainer(context.Background(), "abc")
	require.ErrorIs(t, err, ErrNoContainerRuntime)
}
