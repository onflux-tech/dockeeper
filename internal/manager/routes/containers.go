package routes

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/manager/services"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// RegisterContainerRoutes registers all /api/dk/containers/* routes.
func RegisterContainerRoutes(se *core.ServeEvent, svc *services.ContainerService, cascadeSvc *services.CascadeService, rbac *middleware.RBAC) {
	containers := se.Router.Group("/api/dk/containers")

	containers.POST("/{id}/start", handleAction(svc, agentpb.CommandType_COMMAND_TYPE_START)).BindFunc(rbac.RequireCapability(constants.CapContainersManage))
	containers.POST("/{id}/stop", handleAction(svc, agentpb.CommandType_COMMAND_TYPE_STOP)).BindFunc(rbac.RequireCapability(constants.CapContainersManage))
	containers.POST("/{id}/restart", handleAction(svc, agentpb.CommandType_COMMAND_TYPE_RESTART)).BindFunc(rbac.RequireCapability(constants.CapContainersManage))
	containers.POST("/{id}/remove", handleRemove(svc, cascadeSvc)).BindFunc(rbac.RequireCapability(constants.CapContainersManage))
	containers.POST("/{id}/pause", handleAction(svc, agentpb.CommandType_COMMAND_TYPE_PAUSE)).BindFunc(rbac.RequireCapability(constants.CapContainersManage))
	containers.POST("/{id}/unpause", handleAction(svc, agentpb.CommandType_COMMAND_TYPE_UNPAUSE)).BindFunc(rbac.RequireCapability(constants.CapContainersManage))
	containers.POST("/{id}/update", handleAction(svc, agentpb.CommandType_COMMAND_TYPE_UPDATE_IMAGE)).BindFunc(rbac.RequireCapability(constants.CapContainersManage))
	containers.GET("/{id}/inspect", handleInspect(svc)).BindFunc(rbac.RequireCapability(constants.CapContainersView))
	containers.PUT("/{id}/desired-state", handleDesiredState(svc)).BindFunc(rbac.RequireCapability(constants.CapContainersManage))
}

func handleAction(svc *services.ContainerService, cmdType agentpb.CommandType) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		if id == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{
				"error": "container ID is required",
			})
		}

		var body struct {
			Params map[string]string `json:"params"`
		}
		// Body is optional — ignore parse errors.
		_ = e.BindBody(&body)
		if body.Params == nil {
			body.Params = make(map[string]string)
		}

		result, err := svc.ExecuteAction(
			e.Request.Context(),
			id,
			cmdType,
			body.Params,
			e.Auth.Id,
			e.RealIP(),
		)
		if err != nil {
			return mapServiceError(e, err)
		}

		return e.JSON(http.StatusOK, result)
	}
}

func handleInspect(svc *services.ContainerService) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		if id == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{
				"error": "container ID is required",
			})
		}

		record, err := svc.InspectContainer(id)
		if err != nil {
			return mapServiceError(e, err)
		}

		return e.JSON(http.StatusOK, record)
	}
}

func mapServiceError(e *core.RequestEvent, err error) error {
	switch {
	case errors.Is(err, services.ErrContainerNotFound):
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "container not found",
		})
	case errors.Is(err, services.ErrServerNotFound):
		return e.JSON(http.StatusNotFound, map[string]string{
			"error": "server not found",
		})
	case errors.Is(err, services.ErrServerOffline):
		return e.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "server is offline or unreachable",
		})
	default:
		return e.JSON(http.StatusBadGateway, map[string]string{
			"error": "failed to execute command: " + err.Error(),
		})
	}
}

func handleRemove(svc *services.ContainerService, cascadeSvc *services.CascadeService) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		if id == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{
				"error": "container ID is required",
			})
		}

		var body struct {
			Params         map[string]string                 `json:"params"`
			CascadeOptions *services.ContainerCascadeOptions `json:"cascade_options"`
		}
		_ = e.BindBody(&body)
		if body.Params == nil {
			body.Params = make(map[string]string)
		}

		// Default cascade options: remove_alerts=true, others=false.
		opts := services.ContainerCascadeOptions{
			RemoveAlerts: true,
		}
		if body.CascadeOptions != nil {
			opts = *body.CascadeOptions
		}

		// Read container record to get serverID before Docker removes it.
		containerRecord, err := svc.InspectContainer(id)
		if err != nil {
			return mapServiceError(e, err)
		}
		serverID := containerRecord.GetString("server")

		// Run cascade BEFORE Docker remove (PB record still exists).
		cascadeResult, cascadeErr := cascadeSvc.ExecuteContainerCascade(
			e.Request.Context(),
			id,
			serverID,
			opts,
			e.Auth.Id,
			e.RealIP(),
		)
		if cascadeErr != nil {
			slog.Warn("container cascade failed (best-effort)", "container_id", id, "error", cascadeErr)
		}

		// Execute Docker container remove via agent gRPC.
		result, err := svc.ExecuteAction(
			e.Request.Context(),
			id,
			agentpb.CommandType_COMMAND_TYPE_REMOVE,
			body.Params,
			e.Auth.Id,
			e.RealIP(),
		)
		if err != nil {
			return mapServiceError(e, err)
		}

		resp := map[string]any{
			"success": result.Success,
			"output":  result.Output,
		}
		if result.Error != "" {
			resp["error"] = result.Error
		}
		if cascadeResult != nil {
			resp["cascade"] = map[string]any{
				"deleted_resources":   cascadeResult.DeletedResources,
				"protected_resources": cascadeResult.ProtectedResources,
			}
		}

		return e.JSON(http.StatusOK, resp)
	}
}

func handleDesiredState(svc *services.ContainerService) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		if id == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "container ID is required"})
		}

		var body struct {
			DesiredState string `json:"desired_state"`
		}
		if err := e.BindBody(&body); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		}

		switch body.DesiredState {
		case "none", "running", "stopped":
			// valid
		default:
			return e.JSON(http.StatusBadRequest, map[string]string{
				"error": "desired_state must be one of: none, running, stopped",
			})
		}

		record, err := svc.InspectContainer(id)
		if err != nil {
			return mapServiceError(e, err)
		}

		record.Set("desired_state", body.DesiredState)
		if err := svc.App().Save(record); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save desired state"})
		}

		return e.JSON(http.StatusOK, map[string]any{
			"id":            record.Id,
			"desired_state": body.DesiredState,
		})
	}
}
