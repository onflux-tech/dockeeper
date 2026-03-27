package routes

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/manager/services"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// RegisterUpdateRoutes adds container update management endpoints.
func RegisterUpdateRoutes(se *core.ServeEvent, orch *services.UpdateOrchestrator, rbac *middleware.RBAC) {
	updates := se.Router.Group("/api/dk/updates")

	// Trigger manual update check.
	updates.GET("/check", func(e *core.RequestEvent) error {
		return handleCheckUpdates(e, orch)
	}).BindFunc(rbac.RequireCapability(constants.CapContainersManage))

	// List update history.
	updates.GET("/history", func(e *core.RequestEvent) error {
		return handleUpdateHistory(e, orch)
	}).BindFunc(rbac.RequireCapability(constants.CapContainersView))

	// Manually trigger update for a specific container.
	updates.POST("/{containerId}/apply", func(e *core.RequestEvent) error {
		return handleApplyUpdate(e, orch)
	}).BindFunc(rbac.RequireCapability(constants.CapContainersManage))
}

func handleCheckUpdates(e *core.RequestEvent, orch *services.UpdateOrchestrator) error {
	results, err := orch.CheckForUpdates(e.Request.Context())
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return e.JSON(http.StatusOK, map[string]any{
		"total":   len(results),
		"results": results,
	})
}

func handleUpdateHistory(e *core.RequestEvent, orch *services.UpdateOrchestrator) error {
	q := e.Request.URL.Query()
	containerID := q.Get("container_id")
	serverID := q.Get("server_id")
	status := q.Get("status")

	filter := ""
	params := map[string]any{}

	if containerID != "" {
		filter = "container = {:cid}"
		params["cid"] = containerID
	}
	if serverID != "" {
		if filter != "" {
			filter += " && "
		}
		filter += "server = {:sid}"
		params["sid"] = serverID
	}
	if status != "" {
		if filter != "" {
			filter += " && "
		}
		filter += "status = {:status}"
		params["status"] = status
	}

	records, err := orch.App().FindRecordsByFilter(
		constants.CollUpdateHistory,
		filter,
		"-created",
		200, 0,
		params,
	)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return e.JSON(http.StatusOK, records)
}

func handleApplyUpdate(e *core.RequestEvent, orch *services.UpdateOrchestrator) error {
	containerID := e.Request.PathValue("containerId")
	if containerID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "container ID required"})
	}

	if err := orch.ExecuteUpdate(e.Request.Context(), containerID); err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return e.JSON(http.StatusOK, map[string]string{
		"status":  "completed",
		"message": "update applied successfully",
	})
}
