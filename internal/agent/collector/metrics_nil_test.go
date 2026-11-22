package collector

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCollectContainersWithoutOrchestrator(t *testing.T) {
	m := &MetricsCollector{orch: nil, logger: slog.Default()}
	require.NotPanics(t, func() {
		m.collectContainers(context.Background())
	})
}
