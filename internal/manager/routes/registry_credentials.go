package routes

import (
	"encoding/json"
	"net/http"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
	"github.com/onflux-tech/dockeeper/internal/shared/crypto"
)

// RegisterRegistryCredentialRoutes adds registry credential management endpoints.
func RegisterRegistryCredentialRoutes(se *core.ServeEvent, app core.App, encKey string, rbac *middleware.RBAC) {
	creds := se.Router.Group("/api/dk/registry-credentials")

	creds.GET("", func(e *core.RequestEvent) error {
		return handleListRegistryCredentials(e, app)
	}).BindFunc(rbac.RequireCapability(constants.CapSettingsView))

	creds.POST("", func(e *core.RequestEvent) error {
		return handleCreateRegistryCredential(e, app, encKey)
	}).BindFunc(rbac.RequireCapability(constants.CapSettingsManage))

	creds.PUT("/{id}", func(e *core.RequestEvent) error {
		return handleUpdateRegistryCredential(e, app, encKey)
	}).BindFunc(rbac.RequireCapability(constants.CapSettingsManage))

	creds.DELETE("/{id}", func(e *core.RequestEvent) error {
		return handleDeleteRegistryCredential(e, app)
	}).BindFunc(rbac.RequireCapability(constants.CapSettingsManage))
}

func handleListRegistryCredentials(e *core.RequestEvent, app core.App) error {
	records, err := app.FindRecordsByFilter(
		constants.CollRegistryCredentials,
		"",
		"-created",
		200, 0,
	)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Return WITHOUT the encrypted config field.
	items := make([]map[string]any, 0, len(records))
	for _, r := range records {
		items = append(items, map[string]any{
			"id":            r.Id,
			"name":          r.GetString("name"),
			"registry_host": r.GetString("registry_host"),
			"created":       r.GetString("created"),
			"updated":       r.GetString("updated"),
		})
	}

	return e.JSON(http.StatusOK, items)
}

type registryCredentialBody struct {
	Name         string `json:"name"`
	RegistryHost string `json:"registry_host"`
	Username     string `json:"username"`
	Password     string `json:"password"`
}

func handleCreateRegistryCredential(e *core.RequestEvent, app core.App, encKey string) error {
	var body registryCredentialBody
	if err := json.NewDecoder(e.Request.Body).Decode(&body); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
	}

	if body.Name == "" || body.RegistryHost == "" || body.Username == "" || body.Password == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "name, registry_host, username, and password are required"})
	}

	if encKey == "" {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": "encryption key not configured"})
	}

	// Encrypt credentials.
	credJSON, _ := json.Marshal(map[string]string{
		"username": body.Username,
		"password": body.Password,
	})

	encrypted, err := crypto.Encrypt(credJSON, encKey)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to encrypt credentials"})
	}

	col, err := app.FindCachedCollectionByNameOrId(constants.CollRegistryCredentials)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	rec := core.NewRecord(col)
	rec.Set("name", body.Name)
	rec.Set("registry_host", body.RegistryHost)
	rec.Set("config", encrypted)

	if err := app.Save(rec); err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return e.JSON(http.StatusCreated, map[string]any{
		"id":            rec.Id,
		"name":          body.Name,
		"registry_host": body.RegistryHost,
		"created":       rec.GetString("created"),
	})
}

func handleUpdateRegistryCredential(e *core.RequestEvent, app core.App, encKey string) error {
	id := e.Request.PathValue("id")

	rec, err := app.FindRecordById(constants.CollRegistryCredentials, id)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{"error": "credential not found"})
	}

	var body registryCredentialBody
	if err := json.NewDecoder(e.Request.Body).Decode(&body); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
	}

	if body.Name != "" {
		rec.Set("name", body.Name)
	}
	if body.RegistryHost != "" {
		rec.Set("registry_host", body.RegistryHost)
	}

	// Re-encrypt if credentials provided.
	if body.Username != "" && body.Password != "" {
		if encKey == "" {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "encryption key not configured"})
		}

		credJSON, _ := json.Marshal(map[string]string{
			"username": body.Username,
			"password": body.Password,
		})

		encrypted, err := crypto.Encrypt(credJSON, encKey)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to encrypt credentials"})
		}

		rec.Set("config", encrypted)
	}

	if err := app.Save(rec); err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return e.JSON(http.StatusOK, map[string]any{
		"id":            rec.Id,
		"name":          rec.GetString("name"),
		"registry_host": rec.GetString("registry_host"),
		"updated":       rec.GetString("updated"),
	})
}

func handleDeleteRegistryCredential(e *core.RequestEvent, app core.App) error {
	id := e.Request.PathValue("id")

	rec, err := app.FindRecordById(constants.CollRegistryCredentials, id)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{"error": "credential not found"})
	}

	if err := app.Delete(rec); err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return e.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}
