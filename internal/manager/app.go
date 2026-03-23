package manager

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	dkgrpc "github.com/jkdesenvolvedor/dockeeper/internal/manager/grpc"
	"github.com/jkdesenvolvedor/dockeeper/internal/manager/hooks"
	"github.com/jkdesenvolvedor/dockeeper/internal/manager/middleware"
	"github.com/jkdesenvolvedor/dockeeper/internal/manager/routes"
	"github.com/jkdesenvolvedor/dockeeper/internal/manager/services"
)

const grpcAddr = ":50051"

// Bootstrap registers all custom routes, hooks, and background jobs on the PocketBase app.
func Bootstrap(app *pocketbase.PocketBase, version, commit string) {
	// Auth rate limiter: 5 requests per minute.
	authLimiter := middleware.NewRateLimiter(5, time.Minute)

	// Register hooks.
	hooks.RegisterAdminSetupHook(app)

	// Register agent offline checker cron (every 1 min).
	hooks.RegisterOfflineChecker(app)

	// Cleanup rate limiter entries every 5 minutes.
	if err := app.Cron().Add("rl_cleanup", "*/5 * * * *", func() {
		authLimiter.Cleanup()
	}); err != nil {
		slog.Error("failed to register rate limiter cleanup cron", "error", err)
	}

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		registerHealthRoutes(se, version, commit)
		routes.RegisterAuthRoutes(se, authLimiter)

		// Initialize internal CA and start gRPC server.
		ca, err := services.NewCertAuthority(app.DataDir())
		if err != nil {
			slog.Error("failed to initialize cert authority", "error", err)
			return err
		}

		grpcServer, err := dkgrpc.NewServer(app, ca, grpcAddr)
		if err != nil {
			slog.Error("failed to create gRPC server", "error", err)
			return err
		}

		// Start gRPC server in background goroutine.
		go func() {
			if err := grpcServer.Start(); err != nil {
				slog.Error("gRPC server exited", "error", err)
			}
		}()

		// Register termination hook to stop gRPC server.
		app.OnTerminate().BindFunc(func(te *core.TerminateEvent) error {
			grpcServer.Stop()
			return te.Next()
		})

		slog.Info("dockeeper manager started", "version", version, "commit", commit, "grpc", grpcAddr)
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
