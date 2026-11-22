package routes

import (
	"errors"
	"net/http"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/manager/services"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

func RegisterHostAppRoutes(se *core.ServeEvent, svc *services.HostActionService, rbac *middleware.RBAC) {
	se.Router.POST("/api/dk/servers/{id}/apps/action", func(e *core.RequestEvent) error {
		serverID := e.Request.PathValue("id")
		if serverID == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "server ID is required"})
		}

		var body struct {
			Source string `json:"source"`
			Name   string `json:"name"`
			Action string `json:"action"`
		}
		if err := e.BindBody(&body); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		}

		userID := ""
		if e.Auth != nil {
			userID = e.Auth.Id
		}

		output, err := svc.Execute(e.Request.Context(), serverID, body.Source, body.Name, body.Action, userID, e.RealIP())
		if err != nil {
			switch {
			case errors.Is(err, services.ErrHostAppNotFound):
				return e.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
			case errors.Is(err, services.ErrInvalidHostAction):
				return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
			case errors.Is(err, services.ErrServerOffline), errors.Is(err, services.ErrHostActionInFlight):
				return e.JSON(http.StatusConflict, map[string]string{"error": err.Error()})
			case errors.Is(err, services.ErrHostActionFailed):
				return e.JSON(http.StatusBadGateway, map[string]string{"error": err.Error()})
			default:
				return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}
		}

		return e.JSON(http.StatusOK, map[string]string{"output": output})
	}).BindFunc(rbac.RequireCapability(constants.CapHostAppsManage))
}
