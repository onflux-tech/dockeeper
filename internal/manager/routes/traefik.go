package routes

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"

	"github.com/jkdesenvolvedor/dockeeper/internal/manager/middleware"
	"github.com/jkdesenvolvedor/dockeeper/internal/shared/constants"
)

// RegisterTraefikRoutes adds Traefik route query endpoints.
func RegisterTraefikRoutes(se *core.ServeEvent, app core.App) {
	traefik := se.Router.Group("/api/dk/traefik")

	traefik.GET("/routes", func(e *core.RequestEvent) error {
		return handleListTraefikRoutes(e, app)
	}).BindFunc(middleware.RequireSuperuser)
}

func handleListTraefikRoutes(e *core.RequestEvent, app core.App) error {
	serverID := e.Request.URL.Query().Get("server_id")

	filter := ""
	if serverID != "" {
		filter = "server = '" + serverID + "'"
	}

	records, err := app.FindRecordsByFilter(
		constants.CollTraefikRoutes,
		filter,
		"-created",
		200, 0,
	)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return e.JSON(http.StatusOK, records)
}
