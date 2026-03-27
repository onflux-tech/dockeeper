package routes

import (
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/manager/store"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// RegisterAPIKeyRoutes adds API key management endpoints.
func RegisterAPIKeyRoutes(se *core.ServeEvent, app core.App, apiKeyStore *store.APIKeyStore, rbac *middleware.RBAC) {
	keys := se.Router.Group("/api/dk/api-keys")

	keys.GET("", handleListAPIKeys(apiKeyStore)).BindFunc(rbac.RequireCapability(constants.CapUsersView))
	keys.POST("", handleCreateAPIKey(apiKeyStore)).BindFunc(rbac.RequireCapability(constants.CapUsersManage))
	keys.DELETE("/{id}", handleDeleteAPIKey(apiKeyStore)).BindFunc(rbac.RequireCapability(constants.CapUsersManage))
}

func handleListAPIKeys(apiKeyStore *store.APIKeyStore) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// Admins (superusers) see all keys; others see only their own.
		userID := ""
		if e.Auth != nil && !e.Auth.IsSuperuser() {
			userID = e.Auth.Id
		}

		records, err := apiKeyStore.ListByUser(userID)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		// Strip sensitive fields — never return key_hash.
		result := make([]map[string]any, 0, len(records))
		for _, r := range records {
			result = append(result, map[string]any{
				"id":           r.Id,
				"name":         r.GetString("name"),
				"key_prefix":   r.GetString("key_prefix"),
				"user":         r.GetString("user"),
				"capabilities": r.Get("capabilities"),
				"enabled":      r.GetBool("enabled"),
				"last_used_at": r.GetString("last_used_at"),
				"expires_at":   r.GetString("expires_at"),
				"created":      r.GetString("created"),
			})
		}

		return e.JSON(http.StatusOK, result)
	}
}

func handleCreateAPIKey(apiKeyStore *store.APIKeyStore) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var body struct {
			Name         string   `json:"name"`
			Capabilities []string `json:"capabilities"`
			ExpiresAt    string   `json:"expires_at"`
		}
		if err := e.BindBody(&body); err != nil || body.Name == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "name is required"})
		}

		userID := ""
		if e.Auth != nil {
			userID = e.Auth.Id
		}

		var expiresAt *time.Time
		if body.ExpiresAt != "" {
			t, err := time.Parse(time.RFC3339, body.ExpiresAt)
			if err != nil {
				return e.JSON(http.StatusBadRequest, map[string]string{"error": "expires_at must be RFC3339 format"})
			}
			expiresAt = &t
		}

		plaintext, rec, err := apiKeyStore.Create(body.Name, userID, body.Capabilities, expiresAt)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return e.JSON(http.StatusCreated, map[string]any{
			"id":         rec.Id,
			"key":        plaintext,
			"key_prefix": rec.GetString("key_prefix"),
			"name":       rec.GetString("name"),
		})
	}
}

func handleDeleteAPIKey(apiKeyStore *store.APIKeyStore) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		keyID := e.Request.PathValue("id")

		if err := apiKeyStore.Delete(keyID); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return e.JSON(http.StatusOK, map[string]string{"status": "deleted"})
	}
}
