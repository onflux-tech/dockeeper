package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/onflux-tech/dockeeper/internal/agent"
	"github.com/onflux-tech/dockeeper/internal/shared/logging"
)

// Set at build time via ldflags.
var (
	Version = "0.1.0"
	Commit  = "unknown"
)

func main() {
	logging.ConfigureDefault()

	managerURL := flag.String("manager-url", "", "Manager gRPC address (host:port)")
	token := flag.String("token", "", "One-time enrollment token")
	dataDir := flag.String("data-dir", "./dk-agent-data", "Directory for config and certificates")
	flag.Parse()

	if *managerURL == "" {
		fmt.Fprintln(os.Stderr, "error: --manager-url is required")
		flag.Usage()
		os.Exit(1)
	}

	slog.Info("dockeeper-agent starting", "version", Version, "commit", Commit)

	// Propagate version to agent package.
	agent.Version = Version

	// Context cancelled on SIGTERM / SIGINT.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	for {
		cfg, err := agent.LoadConfig(*dataDir)
		if err != nil {
			slog.Error("failed to load config", "error", err)
			os.Exit(1)
		}

		// CLI flags override persisted config.
		cfg.ManagerURL = *managerURL

		a, err := agent.NewAgent(cfg, *token)
		if err != nil {
			slog.Error("failed to create agent", "error", err)
			os.Exit(1)
		}

		err = a.Run(ctx)
		if errors.Is(err, agent.ErrServerGone) {
			slog.Warn("server was deleted, agent will re-enroll")
			// Check if the parent context is still alive before retrying.
			if ctx.Err() != nil {
				break
			}
			continue
		}
		if err != nil {
			slog.Error("agent exited with error", "error", err)
			os.Exit(1)
		}
		break
	}

	slog.Info("agent stopped")
}
