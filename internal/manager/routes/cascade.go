package routes

import (
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/manager/services"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// RegisterCascadeRoutes adds cascade preview endpoints.
func RegisterCascadeRoutes(se *core.ServeEvent, cascadeSvc *services.CascadeService, rbac *middleware.RBAC) {
	se.Router.GET("/api/dk/containers/{id}/cascade-preview", func(e *core.RequestEvent) error {
		return handleContainerCascadePreview(e, cascadeSvc)
	}).BindFunc(rbac.RequireCapability(constants.CapContainersView))

	se.Router.GET("/api/dk/volumes/{id}/cascade-preview", func(e *core.RequestEvent) error {
		return handleVolumeCascadePreview(e, cascadeSvc)
	}).BindFunc(rbac.RequireCapability(constants.CapContainersView))

	se.Router.GET("/api/dk/servers/{id}/cascade-preview", func(e *core.RequestEvent) error {
		return handleServerCascadePreview(e, cascadeSvc)
	}).BindFunc(rbac.RequireCapability(constants.CapServersView))
}

func handleContainerCascadePreview(e *core.RequestEvent, svc *services.CascadeService) error {
	containerID := e.Request.PathValue("id")

	preview, err := svc.PreviewContainerRemove(containerID)
	if err != nil {
		return mapCascadeError(e, err)
	}

	return e.JSON(http.StatusOK, preview)
}

func handleVolumeCascadePreview(e *core.RequestEvent, svc *services.CascadeService) error {
	volumeID := e.Request.PathValue("id")

	preview, err := svc.PreviewVolumeRemove(volumeID)
	if err != nil {
		return mapCascadeError(e, err)
	}

	return e.JSON(http.StatusOK, preview)
}

func handleServerCascadePreview(e *core.RequestEvent, svc *services.CascadeService) error {
	serverID := e.Request.PathValue("id")

	preview, err := svc.PreviewServerDelete(serverID)
	if err != nil {
		return mapCascadeError(e, err)
	}

	return e.JSON(http.StatusOK, preview)
}

func mapCascadeError(e *core.RequestEvent, err error) error {
	if strings.Contains(err.Error(), "find") {
		return e.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}
	return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
}
