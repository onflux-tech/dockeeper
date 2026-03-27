package manager

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/events"
	dkgrpc "github.com/onflux-tech/dockeeper/internal/manager/grpc"
	"github.com/onflux-tech/dockeeper/internal/manager/hooks"
	"github.com/onflux-tech/dockeeper/internal/manager/jobs"
	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/manager/routes"
	"github.com/onflux-tech/dockeeper/internal/manager/services"
	"github.com/onflux-tech/dockeeper/internal/manager/store"
)

const grpcAddr = ":50051"

// Bootstrap registers all custom routes, hooks, and background jobs on the PocketBase app.
func Bootstrap(app *pocketbase.PocketBase, version, commit string, frontendFS embed.FS) {
	// Auth rate limiter: 5 requests per minute.
	authLimiter := middleware.NewRateLimiter(5, time.Minute)

	// Category rate limiter for RBAC-protected routes.
	categoryLimiter := middleware.NewCategoryRateLimiter(map[string]middleware.RateCategoryConfig{
		"auth":        {Max: 5, Window: time.Minute},
		"read":        {Max: 120, Window: time.Minute},
		"write":       {Max: 30, Window: time.Minute},
		"destructive": {Max: 10, Window: time.Minute},
	})

	// Register hooks.
	hooks.RegisterAdminSetupHook(app)

	// Create EventBus and subscribe handlers (wired to stores inside OnServe).
	eventBus := events.NewBus()

	// Register agent offline checker cron (every 1 min).
	hooks.RegisterOfflineChecker(app, eventBus)

	// Register metrics aggregation and cleanup jobs.
	jobs.RegisterJobs(app)

	// Register audit log cleanup cron (daily at 03:00, retention 90d).
	hooks.RegisterAuditCleanup(app)

	// Cleanup rate limiter entries every 5 minutes.
	if err := app.Cron().Add("rl_cleanup", "*/5 * * * *", func() {
		authLimiter.Cleanup()
		categoryLimiter.Cleanup()
	}); err != nil {
		slog.Error("failed to register rate limiter cleanup cron", "error", err)
	}

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// RBAC stores and middleware.
		rbacStore := store.NewRBACStore(app)
		apiKeyStore := store.NewAPIKeyStore(app)
		rbac := middleware.NewRBAC(rbacStore, apiKeyStore)

		registerHealthRoutes(se, version, commit)
		routes.RegisterAgentPublicRoutes(se, app)
		routes.RegisterAuthRoutes(se, authLimiter)

		// User, group, and API key management routes.
		routes.RegisterUserRoutes(se, app, rbacStore, rbac)
		routes.RegisterAPIKeyRoutes(se, app, apiKeyStore, rbac)

		// Container action routes.
		auditStore := store.NewAuditStore(app)
		dispatcher := services.NewGRPCCommandDispatcher()
		containerSvc := services.NewContainerService(app, dispatcher, auditStore)

		// Subscribe EventBus handlers (now that stores are available).
		eventBus.Subscribe(events.NewDBHandler(app))
		eventBus.Subscribe(events.NewAuditHandler(auditStore))
		eventBus.Subscribe(events.NewAlertHandler())

		// Volume + Cascade services (cascade depends on volumeSvc).
		volumeSvc := services.NewVolumeService(app, auditStore)
		cascadeSvc := services.NewCascadeService(app, auditStore, volumeSvc)
		routes.RegisterVolumeRoutes(se, volumeSvc, cascadeSvc, rbac)
		routes.RegisterCascadeRoutes(se, cascadeSvc, rbac)

		// Register container routes (needs cascadeSvc for remove handler).
		routes.RegisterContainerRoutes(se, containerSvc, cascadeSvc, rbac)

		// Server provisioning and management routes.
		routes.RegisterServerRoutes(se, app, auditStore, cascadeSvc, version, rbac)

		// Container resource limits and env var routes.
		routes.RegisterContainerResourceRoutes(se, containerSvc, rbac)
		routes.RegisterContainerEnvRoutes(se, containerSvc, rbac)

		// Container creation route.
		containerCreateSvc := services.NewContainerCreateService(app, dispatcher, auditStore)
		routes.RegisterContainerCreateRoutes(se, containerCreateSvc, rbac)

		// Container logs SSE streaming route.
		routes.RegisterContainerLogRoutes(se, app, rbac)

		// Terminal WebSocket proxy route.
		termStore := store.NewTerminalStore(app)
		termSvc := services.NewTerminalService(app, termStore, auditStore)
		routes.RegisterTerminalRoutes(se, app, termSvc, rbac)

		// Image management routes.
		imageSvc := services.NewImageService(app, auditStore)
		routes.RegisterImageRoutes(se, app, imageSvc, rbac)

		// Compose stack management routes.
		composeSvc := services.NewComposeService(app, auditStore)
		routes.RegisterComposeRoutes(se, app, composeSvc, rbac)

		// Network management routes.
		networkSvc := services.NewNetworkService(app, auditStore)
		routes.RegisterNetworkRoutes(se, networkSvc, rbac)

		// Backup system routes + scheduler.
		backupStore := store.NewBackupStore(app)
		backupSvc := services.NewBackupService(app, backupStore, auditStore)
		routes.RegisterBackupRoutes(se, app, backupSvc, backupStore, rbac)
		jobs.RegisterBackupJobs(app, backupSvc, backupStore)

		// Alert engine + notification routes.
		alertStore := store.NewAlertStore(app)
		notifySvc := services.NewNotificationService(alertStore)
		alertEngine := services.NewAlertEngine(app, alertStore, notifySvc)
		routes.RegisterAlertRoutes(se, app, alertStore, rbac)
		routes.RegisterNotificationRoutes(se, app, alertStore, notifySvc, rbac)
		jobs.RegisterAlertJobs(app, alertEngine)

		// Traefik route read-only API.
		routes.RegisterTraefikRoutes(se, app, rbac)

		// Global search endpoint.
		routes.RegisterSearchRoutes(se, app, rbac)

		// Audit log query routes + PB hooks.
		routes.RegisterAuditRoutes(se, app, rbac)
		hooks.RegisterAuditHooks(app, auditStore)

		// Events API route.
		routes.RegisterEventRoutes(se, app, rbac)

		// Serve embedded frontend (SPA fallback).
		registerFrontendHandler(se, frontendFS)

		// Initialize internal CA and start gRPC server.
		var extraSANs []string
		if hosts := os.Getenv("DK_GRPC_HOSTS"); hosts != "" {
			for _, h := range strings.Split(hosts, ",") {
				if s := strings.TrimSpace(h); s != "" {
					extraSANs = append(extraSANs, s)
				}
			}
			slog.Info("gRPC extra SANs from DK_GRPC_HOSTS", "hosts", extraSANs)
		}

		ca, err := services.NewCertAuthority(app.DataDir(), extraSANs)
		if err != nil {
			slog.Error("failed to initialize cert authority", "error", err)
			return err
		}

		grpcServer, err := dkgrpc.NewServer(app, ca, grpcAddr, eventBus)
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

		// Register termination hook to stop gRPC server and EventBus.
		app.OnTerminate().BindFunc(func(te *core.TerminateEvent) error {
			grpcServer.Stop()
			eventBus.Close()
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

// registerFrontendHandler serves the embedded React SPA.
// API routes take precedence; all other GET requests fall back to index.html.
func registerFrontendHandler(se *core.ServeEvent, frontendFS embed.FS) {
	distFS, err := fs.Sub(frontendFS, "web/dist")
	if err != nil {
		slog.Warn("frontend embed not available, skipping SPA handler", "error", err)
		return
	}

	fileServer := http.FileServerFS(distFS)

	se.Router.GET("/{path...}", func(e *core.RequestEvent) error {
		path := e.Request.URL.Path

		// Skip API and PocketBase internal routes.
		if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/_/") {
			return e.Next()
		}

		// Try to serve the static file directly.
		cleanPath := strings.TrimPrefix(path, "/")
		if cleanPath != "" {
			if f, err := distFS.(fs.ReadFileFS).ReadFile(cleanPath); err == nil && f != nil {
				fileServer.ServeHTTP(e.Response, e.Request)
				return nil
			}
		}

		// SPA fallback: serve index.html for client-side routing.
		e.Request.URL.Path = "/"
		fileServer.ServeHTTP(e.Response, e.Request)
		return nil
	})
}
