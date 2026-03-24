package routes

import (
	"encoding/json"
	"net/http"

	"github.com/pocketbase/pocketbase/core"

	"github.com/jkdesenvolvedor/dockeeper/internal/manager/middleware"
	"github.com/jkdesenvolvedor/dockeeper/internal/manager/services"
	agentpb "github.com/jkdesenvolvedor/dockeeper/proto/gen/dockeeper/agent/v1"
)

// RegisterContainerEnvRoutes adds GET/PUT routes for container environment variables.
func RegisterContainerEnvRoutes(se *core.ServeEvent, svc *services.ContainerService) {
	se.Router.GET("/api/dk/containers/{id}/env", handleGetEnv(svc)).BindFunc(middleware.RequireSuperuser)
	se.Router.PUT("/api/dk/containers/{id}/env", handleUpdateEnv(svc)).BindFunc(middleware.RequireSuperuser)
}

func handleGetEnv(svc *services.ContainerService) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		if id == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "container ID is required"})
		}

		record, err := svc.InspectContainer(id)
		if err != nil {
			return mapServiceError(e, err)
		}

		// Parse env_vars JSON field — could be []string or map[string]string.
		raw := record.Get("env_vars")
		var envMap map[string]string

		switch v := raw.(type) {
		case []any:
			envMap = make(map[string]string, len(v))
			for _, item := range v {
				s, ok := item.(string)
				if !ok {
					continue
				}
				parts := splitEnvVar(s)
				envMap[parts[0]] = parts[1]
			}
		case map[string]any:
			envMap = make(map[string]string, len(v))
			for k, val := range v {
				envMap[k] = toString(val)
			}
		case string:
			var arr []string
			if jsonErr := json.Unmarshal([]byte(v), &arr); jsonErr == nil {
				envMap = make(map[string]string, len(arr))
				for _, s := range arr {
					parts := splitEnvVar(s)
					envMap[parts[0]] = parts[1]
				}
			}
		}

		if envMap == nil {
			envMap = make(map[string]string)
		}

		return e.JSON(http.StatusOK, map[string]any{"env": envMap})
	}
}

func handleUpdateEnv(svc *services.ContainerService) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		if id == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "container ID is required"})
		}

		var body struct {
			Env map[string]string `json:"env"`
		}
		if err := e.BindBody(&body); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		}

		if body.Env == nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "env is required"})
		}

		// Convert map to KEY=VALUE slice for Docker.
		envSlice := make([]string, 0, len(body.Env))
		for k, v := range body.Env {
			envSlice = append(envSlice, k+"="+v)
		}
		envJSON, _ := json.Marshal(envSlice)

		result, err := svc.ExecuteAction(
			e.Request.Context(),
			id,
			agentpb.CommandType_COMMAND_TYPE_RECREATE_ENV,
			map[string]string{"env": string(envJSON)},
			e.Auth.Id,
			e.RealIP(),
		)
		if err != nil {
			return mapServiceError(e, err)
		}

		return e.JSON(http.StatusOK, result)
	}
}

// splitEnvVar splits "KEY=VALUE" into [key, value]. If no "=", value is "".
func splitEnvVar(s string) [2]string {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return [2]string{s[:i], s[i+1:]}
		}
	}
	return [2]string{s, ""}
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	b, _ := json.Marshal(v)
	return string(b)
}
