package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jkdesenvolvedor/dockeeper/internal/agent"
)

// Set at build time via ldflags.
var (
	Version = "dev"
	Commit  = "unknown"
)

func main() {
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

	// Context cancelled on SIGTERM / SIGINT.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	if err := a.Run(ctx); err != nil {
		slog.Error("agent exited with error", "error", err)
		os.Exit(1)
	}

	slog.Info("agent stopped")
}
