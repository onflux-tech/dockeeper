package agent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeOrchestrator(t *testing.T) {
	require.Equal(t, "none", normalizeOrchestrator(""))
	require.Equal(t, "docker", normalizeOrchestrator("docker"))
}
