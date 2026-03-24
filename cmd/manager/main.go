package main

import (
	"log"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	dockeeper "github.com/jkdesenvolvedor/dockeeper"
	"github.com/jkdesenvolvedor/dockeeper/internal/manager"
	_ "github.com/jkdesenvolvedor/dockeeper/migrations"
)

// Version and Commit are set via LDFLAGS at build time.
var (
	Version = "0.1.0"
	Commit  = "unknown"
)

func main() {
	app := pocketbase.New()

	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		Automigrate: true,
	})

	manager.Bootstrap(app, Version, Commit, dockeeper.FrontendDist)

	if err := app.Start(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
