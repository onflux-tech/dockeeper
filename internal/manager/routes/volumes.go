package routes

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/manager/services"
)

// RegisterVolumeRoutes adds volume management endpoints.
func RegisterVolumeRoutes(se *core.ServeEvent, svc *services.VolumeService) {
	volumes := se.Router.Group("/api/dk/volumes")

	volumes.POST("", func(e *core.RequestEvent) error {
		return handleCreateVolume(e, svc)
	}).BindFunc(middleware.RequireSuperuser)

	volumes.DELETE("/{id}", func(e *core.RequestEvent) error {
		return handleRemoveVolume(e, svc)
	}).BindFunc(middleware.RequireSuperuser)

	volumes.GET("/{id}/inspect", func(e *core.RequestEvent) error {
		return handleInspectVolume(e, svc)
	}).BindFunc(middleware.RequireSuperuser)
}

func handleCreateVolume(e *core.RequestEvent, svc *services.VolumeService) error {
	var body struct {
		ServerID string `json:"server_id"`
		Name     string `json:"name"`
		Driver   string `json:"driver"`
	}
	if err := e.BindBody(&body); err != nil || body.ServerID == "" || body.Name == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "server_id and name are required"})
	}

	if body.Driver == "" {
		body.Driver = "local"
	}

	userID := ""
	if e.Auth != nil {
		userID = e.Auth.Id
	}

	resp, err := svc.CreateVolume(e.Request.Context(), body.ServerID, body.Name, body.Driver, userID, e.RealIP())
	if err != nil {
		return mapVolumeServiceError(e, err)
	}

	if !resp.GetSuccess() {
		return e.JSON(http.StatusBadGateway, map[string]string{"error": resp.GetError()})
	}

	return e.JSON(http.StatusCreated, resp)
}

func handleRemoveVolume(e *core.RequestEvent, svc *services.VolumeService) error {
	volumeID := e.Request.PathValue("id")

	var body struct {
		Force bool `json:"force"`
	}
	// Body is optional for DELETE; ignore bind errors.
	_ = e.BindBody(&body)

	userID := ""
	if e.Auth != nil {
		userID = e.Auth.Id
	}

	resp, err := svc.RemoveVolume(e.Request.Context(), volumeID, body.Force, userID, e.RealIP())
	if err != nil {
		return mapVolumeServiceError(e, err)
	}

	if !resp.GetSuccess() {
		return e.JSON(http.StatusBadGateway, map[string]string{"error": resp.GetError()})
	}

	return e.JSON(http.StatusOK, resp)
}

func handleInspectVolume(e *core.RequestEvent, svc *services.VolumeService) error {
	volumeID := e.Request.PathValue("id")

	resp, err := svc.InspectVolume(e.Request.Context(), volumeID)
	if err != nil {
		return mapVolumeServiceError(e, err)
	}

	if !resp.GetSuccess() {
		return e.JSON(http.StatusBadGateway, map[string]string{"error": resp.GetError()})
	}

	return e.JSON(http.StatusOK, resp)
}

func mapVolumeServiceError(e *core.RequestEvent, err error) error {
	switch err {
	case services.ErrServerNotFound:
		return e.JSON(http.StatusNotFound, map[string]string{"error": "server not found"})
	case services.ErrServerOffline:
		return e.JSON(http.StatusServiceUnavailable, map[string]string{"error": "server is offline"})
	default:
		if err.Error() == "volume not found" {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "volume not found"})
		}
		return e.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
	}
}
