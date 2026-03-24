package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	dockeeper "github.com/onflux-tech/dockeeper"
	"github.com/onflux-tech/dockeeper/internal/manager"
	"github.com/onflux-tech/dockeeper/internal/shared/logging"
	_ "github.com/onflux-tech/dockeeper/migrations"
)

// Version and Commit are set via LDFLAGS at build time.
var (
	Version = "0.1.0"
	Commit  = "unknown"
)

func main() {
	logging.ConfigureDefault()

	app := pocketbase.New()

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: true,
	})

	manager.Bootstrap(app, Version, Commit, dockeeper.FrontendDist)

	if err := app.Start(); err != nil {
		slog.Error("manager exited", "error", err)
		log.Fatal(err)
		os.Exit(1)
	}
}
