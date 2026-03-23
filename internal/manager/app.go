package manager

import (
	"log/slog"
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// Bootstrap registers all custom routes, hooks, and background jobs on the PocketBase app.
func Bootstrap(app *pocketbase.PocketBase, version, commit string) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		registerRoutes(se, version, commit)
		slog.Info("dockeeper manager started", "version", version, "commit", commit)
		return se.Next()
	})
}

func registerRoutes(se *core.ServeEvent, version, commit string) {
	// Custom health endpoint with version info
	se.Router.GET("/api/dk/health", func(e *core.RequestEvent) error {
		return e.JSON(http.StatusOK, map[string]string{
			"status":  "ok",
			"version": version,
			"commit":  commit,
		})
	})
}
