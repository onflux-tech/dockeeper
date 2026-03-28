package routes

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/manager/services"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// RegisterTagRoutes registers all /api/dk/tags/* routes.
func RegisterTagRoutes(se *core.ServeEvent, tagSvc *services.TagService, rbac *middleware.RBAC) {
	group := se.Router.Group("/api/dk/tags")

	group.GET("", handleListTags(tagSvc)).BindFunc(rbac.RequireCapability(constants.CapContainersView))
	group.GET("/names", handleListTagNames(tagSvc)).BindFunc(rbac.RequireCapability(constants.CapContainersView))
	group.POST("", handleApplyTag(tagSvc)).BindFunc(rbac.RequireCapability(constants.CapContainersManage))
	group.DELETE("/{id}", handleRemoveTag(tagSvc)).BindFunc(rbac.RequireCapability(constants.CapContainersManage))
	group.POST("/bulk", handleBulkApplyTag(tagSvc)).BindFunc(rbac.RequireCapability(constants.CapContainersManage))
	group.DELETE("/bulk", handleBulkRemoveTag(tagSvc)).BindFunc(rbac.RequireCapability(constants.CapContainersManage))
}

func handleListTags(svc *services.TagService) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		resourceType := e.Request.URL.Query().Get("resource_type")
		records, err := svc.GetAllTags(resourceType)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		tags := make([]map[string]any, 0, len(records))
		for _, r := range records {
			tags = append(tags, map[string]any{
				"id":            r.Id,
				"name":          r.GetString("name"),
				"color":         r.GetString("color"),
				"resource_type": r.GetString("resource_type"),
				"resource_id":   r.GetString("resource_id"),
				"created":       r.GetString("created"),
			})
		}
		return e.JSON(http.StatusOK, map[string]any{"items": tags})
	}
}

func handleListTagNames(svc *services.TagService) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		resourceType := e.Request.URL.Query().Get("resource_type")
		if resourceType == "" {
			resourceType = "container"
		}
		names, err := svc.GetUniqueTagNames(resourceType)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return e.JSON(http.StatusOK, map[string]any{"items": names})
	}
}

func handleApplyTag(svc *services.TagService) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var body struct {
			Name         string `json:"name"`
			Color        string `json:"color"`
			ResourceType string `json:"resource_type"`
			ResourceID   string `json:"resource_id"`
		}
		if err := e.BindBody(&body); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		}
		if body.Name == "" || body.ResourceType == "" || body.ResourceID == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "name, resource_type, and resource_id are required"})
		}
		if body.ResourceType != "container" && body.ResourceType != "server" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "resource_type must be 'container' or 'server'"})
		}

		rec, err := svc.ApplyTag(body.Name, body.Color, body.ResourceType, body.ResourceID)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return e.JSON(http.StatusCreated, map[string]any{
			"id":            rec.Id,
			"name":          rec.GetString("name"),
			"color":         rec.GetString("color"),
			"resource_type": rec.GetString("resource_type"),
			"resource_id":   rec.GetString("resource_id"),
		})
	}
}

func handleRemoveTag(svc *services.TagService) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		if id == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "tag ID is required"})
		}

		if err := svc.RemoveTag(id); err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "tag not found"})
		}

		return e.JSON(http.StatusOK, map[string]string{"status": "removed"})
	}
}

func handleBulkApplyTag(svc *services.TagService) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var body struct {
			Tag          string   `json:"tag"`
			Color        string   `json:"color"`
			ResourceType string   `json:"resource_type"`
			ResourceIDs  []string `json:"resource_ids"`
		}
		if err := e.BindBody(&body); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		}
		if body.Tag == "" || body.ResourceType == "" || len(body.ResourceIDs) == 0 {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "tag, resource_type, and resource_ids are required"})
		}

		applied, err := svc.BulkApply(body.Tag, body.Color, body.ResourceType, body.ResourceIDs)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return e.JSON(http.StatusOK, map[string]any{
			"applied": applied,
			"total":   len(body.ResourceIDs),
		})
	}
}

func handleBulkRemoveTag(svc *services.TagService) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var body struct {
			Tag          string   `json:"tag"`
			ResourceType string   `json:"resource_type"`
			ResourceIDs  []string `json:"resource_ids"`
		}
		if err := e.BindBody(&body); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		}
		if body.Tag == "" || body.ResourceType == "" || len(body.ResourceIDs) == 0 {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "tag, resource_type, and resource_ids are required"})
		}

		removed, err := svc.BulkRemove(body.Tag, body.ResourceType, body.ResourceIDs)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return e.JSON(http.StatusOK, map[string]any{
			"removed": removed,
			"total":   len(body.ResourceIDs),
		})
	}
}
