package agent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/onflux-tech/dockeeper/internal/agent/collector"
)

// DetectOrchestrator probes the local environment for a supported container
// runtime and returns an OrchestratorClient for it.
func DetectOrchestrator(ctx context.Context) (string, collector.OrchestratorClient, error) {
	// Try Docker / Podman (Podman is Docker API compatible).
	dc, err := collector.NewDockerClient()
	if err == nil {
		if pingErr := dc.Ping(ctx); pingErr == nil {
			slog.Info("detected container runtime", "orchestrator", "docker")
			return "docker", dc, nil
		}
		if closeErr := dc.Close(); closeErr != nil {
			slog.Warn("failed to close docker client after failed ping", "error", closeErr)
		}
	}

	// Future: try Kubernetes detection here.

	return "", nil, fmt.Errorf("no supported container runtime detected")
}
