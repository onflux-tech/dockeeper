package routes

import (
	"errors"
	"net/http"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/manager/services"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// RegisterContainerRoutes registers all /api/dk/containers/* routes.
func RegisterContainerRoutes(se *core.ServeEvent, svc *services.ContainerService) {
	containers := se.Router.Group("/api/dk/containers")

	containers.POST("/{id}/start", handleAction(svc, agentpb.CommandType_COMMAND_TYPE_START)).BindFunc(middleware.RequireSuperuser)
	containers.POST("/{id}/stop", handleAction(svc, agentpb.CommandType_COMMAND_TYPE_STOP)).BindFunc(middleware.RequireSuperuser)
	containers.POST("/{id}/restart", handleAction(svc, agentpb.CommandType_COMMAND_TYPE_RESTART)).BindFunc(middleware.RequireSuperuser)
	containers.POST("/{id}/remove", handleAction(svc, agentpb.CommandType_COMMAND_TYPE_REMOVE)).BindFunc(middleware.RequireSuperuser)
	containers.POST("/{id}/pause", handleAction(svc, agentpb.CommandType_COMMAND_TYPE_PAUSE)).BindFunc(middleware.RequireSuperuser)
	containers.POST("/{id}/unpause", handleAction(svc, agentpb.CommandType_COMMAND_TYPE_UNPAUSE)).BindFunc(middleware.RequireSuperuser)
	containers.GET("/{id}/inspect", handleInspect(svc)).BindFunc(middleware.RequireSuperuser)
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
