package routes

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/manager/services"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// RegisterContainerResourceRoutes adds GET/PUT routes for container resource limits.
func RegisterContainerResourceRoutes(se *core.ServeEvent, svc *services.ContainerService, rbac *middleware.RBAC) {
	se.Router.GET("/api/dk/containers/{id}/resources", handleGetResources(svc)).BindFunc(rbac.RequireCapability(constants.CapContainersView))
	se.Router.PUT("/api/dk/containers/{id}/resources", handleUpdateResources(svc)).BindFunc(rbac.RequireCapability(constants.CapContainersManage))
}

// ResourceLimitsResponse is the JSON response for GET /resources.
type ResourceLimitsResponse struct {
	CPULimit  float64 `json:"cpu_limit"`
	MemLimit  int64   `json:"mem_limit"`
	PidsLimit int64   `json:"pids_limit"`
}

func handleGetResources(svc *services.ContainerService) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		if id == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "container ID is required"})
		}

		record, err := svc.InspectContainer(id)
		if err != nil {
			return mapServiceError(e, err)
		}

		// Parse resource_limits JSON field.
		raw := record.Get("resource_limits")
		resp := ResourceLimitsResponse{}

		switch v := raw.(type) {
		case map[string]any:
			if cpu, ok := v["cpu_limit"].(float64); ok {
				resp.CPULimit = cpu
			}
			if mem, ok := v["mem_limit"].(float64); ok {
				resp.MemLimit = int64(mem)
			}
			if pids, ok := v["pids_limit"].(float64); ok {
				resp.PidsLimit = int64(pids)
			}
		case string:
			_ = json.Unmarshal([]byte(v), &resp)
		default:
			// PocketBase may return types.JsonMap or similar; marshal+unmarshal as fallback.
			if b, err := json.Marshal(raw); err == nil {
				_ = json.Unmarshal(b, &resp)
			}
		}

		return e.JSON(http.StatusOK, resp)
	}
}

func handleUpdateResources(svc *services.ContainerService) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		if id == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "container ID is required"})
		}

		var body struct {
			CPULimit  float64 `json:"cpu_limit"`
			MemLimit  int64   `json:"mem_limit"`
			PidsLimit int64   `json:"pids_limit"`
		}
		if err := e.BindBody(&body); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		}

		// Serialize to JSON for the params map.
		resJSON, _ := json.Marshal(body)

		// Get user ID from PB auth or API key context.
		userID := ""
		if e.Auth != nil {
			userID = e.Auth.Id
		} else {
			userID = middleware.GetUserIDFromContext(e.Request.Context())
		}

		result, err := svc.ExecuteAction(
			e.Request.Context(),
			id,
			agentpb.CommandType_COMMAND_TYPE_UPDATE_RESOURCES,
			map[string]string{"resources": string(resJSON)},
			userID,
			e.RealIP(),
		)
		if err != nil {
			return mapServiceError(e, err)
		}

		// Persist the new limits immediately so the UI reflects them before the next agent sync.
		// Only persist when Docker actually applied the update.
		if result.Success {
			if err := svc.UpdateResourceLimits(id, map[string]any{
				"cpu_limit":  body.CPULimit,
				"mem_limit":  body.MemLimit,
				"pids_limit": body.PidsLimit,
			}); err != nil {
				slog.Warn("failed to persist resource limits", "container_id", id, "error", err)
			}
		}

		return e.JSON(http.StatusOK, result)
	}
}
