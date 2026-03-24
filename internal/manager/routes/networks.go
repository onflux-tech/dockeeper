package routes

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/manager/services"
)

// RegisterNetworkRoutes adds network management endpoints.
func RegisterNetworkRoutes(se *core.ServeEvent, svc *services.NetworkService) {
	networks := se.Router.Group("/api/dk/networks")

	networks.POST("", func(e *core.RequestEvent) error {
		return handleCreateNetwork(e, svc)
	}).BindFunc(middleware.RequireSuperuser)

	networks.DELETE("/{id}", func(e *core.RequestEvent) error {
		return handleRemoveNetwork(e, svc)
	}).BindFunc(middleware.RequireSuperuser)

	networks.POST("/{id}/connect", func(e *core.RequestEvent) error {
		return handleConnectContainer(e, svc)
	}).BindFunc(middleware.RequireSuperuser)

	networks.POST("/{id}/disconnect", func(e *core.RequestEvent) error {
		return handleDisconnectContainer(e, svc)
	}).BindFunc(middleware.RequireSuperuser)
}

func handleCreateNetwork(e *core.RequestEvent, svc *services.NetworkService) error {
	var body struct {
		ServerID string `json:"server_id"`
		Name     string `json:"name"`
		Driver   string `json:"driver"`
		Subnet   string `json:"subnet"`
		Gateway  string `json:"gateway"`
		Internal bool   `json:"internal"`
	}
	if err := e.BindBody(&body); err != nil || body.ServerID == "" || body.Name == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "server_id and name are required"})
	}

	if body.Driver == "" {
		body.Driver = "bridge"
	}

	userID := ""
	if e.Auth != nil {
		userID = e.Auth.Id
	}

	resp, err := svc.CreateNetwork(
		e.Request.Context(), body.ServerID,
		body.Name, body.Driver, body.Subnet, body.Gateway, body.Internal,
		userID, e.RealIP(),
	)
	if err != nil {
		return mapNetworkServiceError(e, err)
	}

	if !resp.GetSuccess() {
		return e.JSON(http.StatusBadGateway, map[string]string{"error": resp.GetError()})
	}

	return e.JSON(http.StatusCreated, resp)
}

func handleRemoveNetwork(e *core.RequestEvent, svc *services.NetworkService) error {
	networkID := e.Request.PathValue("id")

	userID := ""
	if e.Auth != nil {
		userID = e.Auth.Id
	}

	resp, err := svc.RemoveNetwork(e.Request.Context(), networkID, userID, e.RealIP())
	if err != nil {
		return mapNetworkServiceError(e, err)
	}

	if !resp.GetSuccess() {
		return e.JSON(http.StatusBadGateway, map[string]string{"error": resp.GetError()})
	}

	return e.JSON(http.StatusOK, resp)
}

func handleConnectContainer(e *core.RequestEvent, svc *services.NetworkService) error {
	networkID := e.Request.PathValue("id")

	var body struct {
		ContainerID string `json:"container_id"`
	}
	if err := e.BindBody(&body); err != nil || body.ContainerID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "container_id is required"})
	}

	userID := ""
	if e.Auth != nil {
		userID = e.Auth.Id
	}

	resp, err := svc.ConnectContainer(e.Request.Context(), networkID, body.ContainerID, userID, e.RealIP())
	if err != nil {
		return mapNetworkServiceError(e, err)
	}

	if !resp.GetSuccess() {
		return e.JSON(http.StatusBadGateway, map[string]string{"error": resp.GetError()})
	}

	return e.JSON(http.StatusOK, resp)
}

func handleDisconnectContainer(e *core.RequestEvent, svc *services.NetworkService) error {
	networkID := e.Request.PathValue("id")

	var body struct {
		ContainerID string `json:"container_id"`
		Force       bool   `json:"force"`
	}
	if err := e.BindBody(&body); err != nil || body.ContainerID == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "container_id is required"})
	}

	userID := ""
	if e.Auth != nil {
		userID = e.Auth.Id
	}

	resp, err := svc.DisconnectContainer(e.Request.Context(), networkID, body.ContainerID, body.Force, userID, e.RealIP())
	if err != nil {
		return mapNetworkServiceError(e, err)
	}

	if !resp.GetSuccess() {
		return e.JSON(http.StatusBadGateway, map[string]string{"error": resp.GetError()})
	}

	return e.JSON(http.StatusOK, resp)
}

func mapNetworkServiceError(e *core.RequestEvent, err error) error {
	switch err {
	case services.ErrServerNotFound:
		return e.JSON(http.StatusNotFound, map[string]string{"error": "server not found"})
	case services.ErrServerOffline:
		return e.JSON(http.StatusServiceUnavailable, map[string]string{"error": "server is offline"})
	case services.ErrContainerNotFound:
		return e.JSON(http.StatusNotFound, map[string]string{"error": "container not found"})
	case services.ErrProtectedNetwork:
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "docker built-in networks cannot be removed"})
	default:
		if err.Error() == "network not found" {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "network not found"})
		}
		return e.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
	}
}
