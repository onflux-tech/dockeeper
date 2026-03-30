package manager

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/channel"
	"github.com/onflux-tech/dockeeper/internal/manager/events"
	dkgrpc "github.com/onflux-tech/dockeeper/internal/manager/grpc"
	"github.com/onflux-tech/dockeeper/internal/manager/hooks"
	"github.com/onflux-tech/dockeeper/internal/manager/jobs"
	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/manager/routes"
	"github.com/onflux-tech/dockeeper/internal/manager/services"
	"github.com/onflux-tech/dockeeper/internal/manager/store"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
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

		// Channel registry for reverse tunnel agent communication.
		registry := channel.NewRegistry(app, eventBus)

		registerHealthRoutes(se, version, commit)
		routes.RegisterAgentPublicRoutes(se, app)
		routes.RegisterAuthRoutes(se, authLimiter, rbacStore)

		// User, group, and API key management routes.
		routes.RegisterUserRoutes(se, app, rbacStore, rbac)
		routes.RegisterAPIKeyRoutes(se, app, apiKeyStore, rbac)

		// Registry credentials routes (encryption key from env or auto-generated).
		encKey := os.Getenv("DOCKEEPER_ENCRYPTION_KEY")
		if encKey == "" {
			encKey = ensureEncryptionKey(app.DataDir())
		}
		routes.RegisterRegistryCredentialRoutes(se, app, encKey, rbac)

		// Container action routes.
		auditStore := store.NewAuditStore(app)
		dispatcher := services.NewRegistryCommandDispatcher(registry)
		containerSvc := services.NewContainerService(app, dispatcher, auditStore)

		// Subscribe EventBus handlers (now that stores are available).
		eventBus.Subscribe(events.NewDBHandler(app))
		eventBus.Subscribe(events.NewAuditHandler(auditStore))
		// AlertHandler is subscribed below after alertEngine is created.

		// Volume + Cascade services (cascade depends on volumeSvc).
		volumeSvc := services.NewVolumeService(app, auditStore, registry)
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
		routes.RegisterContainerLogRoutes(se, app, registry, rbac)

		// Terminal WebSocket proxy route.
		termStore := store.NewTerminalStore(app)
		termSvc := services.NewTerminalService(app, termStore, auditStore)
		routes.RegisterTerminalRoutes(se, app, termSvc, registry, rbac)

		// Image management routes.
		imageSvc := services.NewImageService(app, auditStore, registry)
		routes.RegisterImageRoutes(se, app, imageSvc, registry, rbac)

		// Compose stack management routes.
		composeSvc := services.NewComposeService(app, auditStore, registry)
		routes.RegisterComposeRoutes(se, app, composeSvc, registry, rbac)

		// Network management routes.
		networkSvc := services.NewNetworkService(app, auditStore, registry)
		routes.RegisterNetworkRoutes(se, networkSvc, rbac)

		// Backup system routes + scheduler.
		backupStore := store.NewBackupStore(app)
		backupSvc := services.NewBackupService(app, backupStore, auditStore, registry)
		routes.RegisterBackupRoutes(se, app, backupSvc, backupStore, rbac)
		jobs.RegisterBackupJobs(app, backupSvc, backupStore)

		// Alert engine + notification routes.
		alertStore := store.NewAlertStore(app)
		notifySvc := services.NewNotificationService(alertStore)
		alertEngine := services.NewAlertEngine(app, alertStore, notifySvc)
		routes.RegisterAlertRoutes(se, app, alertStore, rbac)
		routes.RegisterNotificationRoutes(se, app, alertStore, notifySvc, rbac)
		jobs.RegisterAlertJobs(app, alertEngine)

		// Blackout windows service + routes.
		blackoutSvc := services.NewBlackoutService(app)
		routes.RegisterBlackoutRoutes(se, blackoutSvc, rbac)

		// Subscribe event-based alert handler (needs alertEngine + blackoutSvc).
		eventBus.Subscribe(events.NewAlertHandler(app, alertStore, alertEngine, blackoutSvc))

		// Health checks system.
		healthStore := store.NewHealthStore(app)
		healthScheduler := services.NewHealthScheduler(app, healthStore, registry, eventBus)
		routes.RegisterHealthCheckRoutes(se, app, healthStore, healthScheduler, rbac)

		// Dashboard layout persistence.
		routes.RegisterDashboardRoutes(se, app, rbac)

		// Retention settings routes.
		routes.RegisterRetentionRoutes(se, app, rbac)

		// Traefik route read-only API.
		routes.RegisterTraefikRoutes(se, app, rbac)

		// Global search endpoint.
		routes.RegisterSearchRoutes(se, app, rbac)

		// Audit log query routes + PB hooks.
		routes.RegisterAuditRoutes(se, app, rbac)
		hooks.RegisterAuditHooks(app, auditStore)

		// Events API route.
		routes.RegisterEventRoutes(se, app, rbac)

		// Batch operations runner + tags system.
		batchStore := store.NewBatchStore(app)
		batchRunner := services.NewBatchJobRunner(app, containerSvc, eventBus, batchStore)
		tagStore := store.NewTagStore(app)
		tagSvc := services.NewTagService(app, tagStore)
		routes.RegisterBatchRoutes(se, batchRunner, batchStore, tagSvc, rbac)
		routes.RegisterTagRoutes(se, tagSvc, rbac)

		// Update orchestrator + registry adapter + cron.
		registryAdapter := services.NewRegistryAdapter(encKey)
		loadRegistryCredentials(app, registryAdapter)
		updateOrch := services.NewUpdateOrchestrator(app, registryAdapter, registry, eventBus)
		routes.RegisterUpdateRoutes(se, updateOrch, rbac)
		registerUpdateCron(app, updateOrch)

		// Action tokens system.
		actionTokenStore := store.NewActionTokenStore(app)
		actionTokenSvc := services.NewActionTokenService(app, actionTokenStore, containerSvc)
		routes.RegisterActionTokenRoutes(se, actionTokenSvc, auditStore)

		// Wire action tokens into notification service.
		baseURL := os.Getenv("DOCKEEPER_BASE_URL")
		if baseURL != "" {
			notifySvc.SetActionTokenService(actionTokenSvc, baseURL)
		}

		// Self-update system routes.
		selfUpdateSvc := services.NewSelfUpdateService(app, eventBus, registry, version)
		routes.RegisterSystemRoutes(se, app, selfUpdateSvc, rbac)

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

		grpcServer, err := dkgrpc.NewServer(app, ca, grpcAddr, eventBus, registry)
		if err != nil {
			slog.Error("failed to create gRPC server", "error", err)
			return err
		}

		// Start health scheduler and gRPC server in background goroutines.
		go healthScheduler.Start()

		go func() {
			if err := grpcServer.Start(); err != nil {
				slog.Error("gRPC server exited", "error", err)
			}
		}()

		// Register termination hook to stop gRPC server, scheduler, and EventBus.
		app.OnTerminate().BindFunc(func(te *core.TerminateEvent) error {
			healthScheduler.Stop()
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

// loadRegistryCredentials populates the RegistryAdapter's credential cache
// from the registry_credentials collection.
func loadRegistryCredentials(app core.App, adapter *services.RegistryAdapter) {
	records, err := app.FindRecordsByFilter(
		constants.CollRegistryCredentials,
		"", "", 0, 0,
	)
	if err != nil {
		slog.Warn("failed to load registry credentials", "error", err)
		return
	}

	creds := make(map[string]services.RegistryCredential, len(records))
	for _, r := range records {
		encConfig := r.GetString("config")
		cred, err := adapter.LoadCredential(encConfig)
		if err != nil {
			slog.Warn("failed to decrypt registry credential",
				"id", r.Id,
				"host", r.GetString("registry_host"),
				"error", err,
			)
			continue
		}
		creds[r.GetString("registry_host")] = *cred
	}

	adapter.SetCredentials(creds)
	slog.Info("registry credentials loaded", "count", len(creds))
}

// registerUpdateCron registers the update check cron job.
func registerUpdateCron(app core.App, orch *services.UpdateOrchestrator) {
	if err := app.Cron().Add("update_check", "0 */6 * * *", func() {
		orch.RunCheckAndApply()
	}); err != nil {
		slog.Error("failed to register update check cron", "error", err)
	}
}

// ensureEncryptionKey loads or generates a persistent encryption key in the data directory.
func ensureEncryptionKey(dataDir string) string {
	keyPath := filepath.Join(dataDir, ".encryption_key")

	if data, err := os.ReadFile(keyPath); err == nil {
		key := strings.TrimSpace(string(data))
		if key != "" {
			return key
		}
	}

	// Generate a random 32-byte hex key.
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		slog.Error("failed to generate encryption key", "error", err)
		return ""
	}
	key := hex.EncodeToString(buf)

	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		slog.Error("failed to create data dir for encryption key", "error", err)
		return key
	}
	if err := os.WriteFile(keyPath, []byte(key), 0o600); err != nil {
		slog.Error("failed to persist encryption key", "error", err)
	} else {
		slog.Info("auto-generated encryption key", "path", keyPath)
	}

	return key
}
