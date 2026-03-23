package manager

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"github.com/jkdesenvolvedor/dockeeper/internal/manager/hooks"
	"github.com/jkdesenvolvedor/dockeeper/internal/manager/middleware"
	"github.com/jkdesenvolvedor/dockeeper/internal/manager/routes"
)

// Bootstrap registers all custom routes, hooks, and background jobs on the PocketBase app.
func Bootstrap(app *pocketbase.PocketBase, version, commit string) {
	// Auth rate limiter: 5 requests per minute.
	authLimiter := middleware.NewRateLimiter(5, time.Minute)

	// Register hooks.
	hooks.RegisterAdminSetupHook(app)

	// Cleanup rate limiter entries every 5 minutes.
	if err := app.Cron().Add("rl_cleanup", "*/5 * * * *", func() {
		authLimiter.Cleanup()
	}); err != nil {
		slog.Error("failed to register rate limiter cleanup cron", "error", err)
	}

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		registerHealthRoutes(se, version, commit)
		routes.RegisterAuthRoutes(se, authLimiter)

		slog.Info("dockeeper manager started", "version", version, "commit", commit)
		return se.Next()
	})
}

func registerHealthRoutes(se *core.ServeEvent, version, commit string) {
	se.Router.GET("/api/dk/health", func(e *core.RequestEvent) error {
		return e.JSON(http.StatusOK, map[string]string{
			"status":  "ok",
			"version": version,
			"commit":  commit,
		})
	})
}
