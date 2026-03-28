package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/manager/services"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// RegisterSystemRoutes registers self-update and version endpoints.
func RegisterSystemRoutes(se *core.ServeEvent, app core.App, updateSvc *services.SelfUpdateService, rbac *middleware.RBAC) {
	group := se.Router.Group("/api/dk/system")

	// Public: version info.
	group.GET("/version", func(e *core.RequestEvent) error {
		return e.JSON(http.StatusOK, map[string]string{
			"version": updateSvc.CurrentVersion(),
		})
	})

	// Check for updates.
	group.GET("/update/check", handleUpdateCheck(updateSvc)).BindFunc(rbac.RequireCapability(constants.CapSettingsView))

	// Trigger manager self-update.
	group.POST("/update/manager", handleManagerUpdate(updateSvc)).BindFunc(rbac.RequireCapability(constants.CapSettingsManage))

	// Trigger agent update.
	group.POST("/update/agent/{id}", handleAgentUpdate(app, updateSvc)).BindFunc(rbac.RequireCapability(constants.CapSettingsManage))
}

func handleUpdateCheck(svc *services.SelfUpdateService) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		release, isNewer, err := svc.CheckForUpdate()
		if err != nil {
			return e.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
		}

		resp := map[string]any{
			"available": isNewer,
			"current":   svc.CurrentVersion(),
		}

		if release != nil {
			resp["latest"] = release.TagName
			resp["changelog"] = release.Body
			resp["published_at"] = release.PublishedAt

			assets := make([]map[string]any, 0, len(release.Assets))
			for _, a := range release.Assets {
				assets = append(assets, map[string]any{
					"name": a.Name,
					"size": a.Size,
					"url":  a.BrowserDownloadURL,
				})
			}
			resp["assets"] = assets
		}

		return e.JSON(http.StatusOK, resp)
	}
}

func handleManagerUpdate(svc *services.SelfUpdateService) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		release, isNewer, err := svc.CheckForUpdate()
		if err != nil {
			return e.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
		}
		if !isNewer || release == nil {
			return e.JSON(http.StatusOK, map[string]string{"message": "already up to date"})
		}

		// SSE progress
		w := e.Response
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "streaming not supported"})
		}

		sendSSE := func(event string, data any) {
			raw, _ := json.Marshal(data)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, raw)
			flusher.Flush()
		}

		sendSSE("progress", map[string]any{"step": "downloading", "message": "Downloading " + release.TagName + "..."})

		if err := svc.ExecuteManagerUpdate(release); err != nil {
			sendSSE("error", map[string]any{"message": err.Error()})
			return nil
		}

		sendSSE("complete", map[string]any{"step": "done", "message": "Update installed. Restart required.", "version": release.TagName})
		return nil
	}
}

func handleAgentUpdate(app core.App, svc *services.SelfUpdateService) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		serverID := e.Request.PathValue("id")
		if serverID == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "server id required"})
		}

		// Look up the server to get IP, OS, and arch.
		server, err := app.FindRecordById("servers", serverID)
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "server not found"})
		}

		serverIP := server.GetString("ip_address")
		serverOS := server.GetString("os")
		serverArch := server.GetString("arch")
		if serverIP == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "server has no IP address"})
		}
		if serverOS == "" {
			serverOS = "linux"
		}
		if serverArch == "" {
			serverArch = "amd64"
		}

		release, isNewer, err := svc.CheckForUpdate()
		if err != nil {
			return e.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
		}
		if !isNewer || release == nil {
			return e.JSON(http.StatusOK, map[string]string{"message": "already up to date"})
		}

		// SSE progress stream.
		w := e.Response
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "streaming not supported"})
		}

		sendSSE := func(event string, data any) {
			raw, _ := json.Marshal(data)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, raw)
			flusher.Flush()
		}

		sendSSE("progress", map[string]any{
			"step":    "downloading",
			"message": fmt.Sprintf("Downloading agent %s for %s/%s...", release.TagName, serverOS, serverArch),
		})

		// Download the agent binary for the target OS/arch.
		binaryData, checksum, err := svc.DownloadAsset(release, "agent", serverOS, serverArch)
		if err != nil {
			sendSSE("error", map[string]any{"message": err.Error()})
			return nil
		}

		sendSSE("progress", map[string]any{
			"step":    "uploading",
			"message": fmt.Sprintf("Streaming binary to agent (%d bytes)...", len(binaryData)),
		})

		// Stream the binary to the agent via gRPC.
		ctx := e.Request.Context()
		if err := svc.UpdateAgent(ctx, serverID, release.TagName, binaryData, checksum); err != nil {
			sendSSE("error", map[string]any{"message": err.Error()})
			return nil
		}

		sendSSE("complete", map[string]any{
			"step":    "done",
			"message": "Agent updated to " + release.TagName + ". Agent will restart automatically.",
			"version": release.TagName,
		})
		return nil
	}
}
