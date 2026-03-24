package routes

import (
	"errors"
	"net/http"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/manager/services"
	"github.com/onflux-tech/dockeeper/internal/shared/models"
)

// RegisterContainerCreateRoutes registers POST /api/dk/containers/create.
func RegisterContainerCreateRoutes(se *core.ServeEvent, svc *services.ContainerCreateService) {
	se.Router.POST("/api/dk/containers/create", handleContainerCreate(svc)).BindFunc(middleware.RequireSuperuser)
}

func handleContainerCreate(svc *services.ContainerCreateService) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var spec models.ContainerSpec
		if err := e.BindBody(&spec); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid request body",
			})
		}

		result, err := svc.CreateContainer(
			e.Request.Context(),
			&spec,
			e.Auth.Id,
			e.RealIP(),
		)
		if err != nil {
			return mapCreateServiceError(e, err)
		}

		if !result.Success {
			return e.JSON(http.StatusBadGateway, result)
		}

		return e.JSON(http.StatusCreated, result)
	}
}

func mapCreateServiceError(e *core.RequestEvent, err error) error {
	var validationErr *services.ValidationError
	if errors.As(err, &validationErr) {
		return e.JSON(http.StatusUnprocessableEntity, map[string]any{
			"error":  "validation failed",
			"errors": validationErr.Errors,
		})
	}

	var conflictErr *services.ConflictError
	if errors.As(err, &conflictErr) {
		return e.JSON(http.StatusConflict, map[string]string{
			"error": conflictErr.Message,
		})
	}

	switch {
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
			"error": "failed to create container: " + err.Error(),
		})
	}
}
