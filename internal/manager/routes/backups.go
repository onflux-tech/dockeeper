package routes

import (
	"encoding/json"
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/manager/services"
	"github.com/onflux-tech/dockeeper/internal/manager/store"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// RegisterBackupRoutes adds backup management endpoints.
func RegisterBackupRoutes(se *core.ServeEvent, app core.App, backupSvc *services.BackupService, backupStore *store.BackupStore, rbac *middleware.RBAC) {
	backups := se.Router.Group("/api/dk/backups")

	// ── Policies ─────────────────────────────────────
	backups.GET("/policies", func(e *core.RequestEvent) error {
		return handleListPolicies(e, app)
	}).BindFunc(rbac.RequireCapability(constants.CapBackupsView))

	backups.POST("/policies", func(e *core.RequestEvent) error {
		return handleCreatePolicy(e, backupStore)
	}).BindFunc(rbac.RequireCapability(constants.CapBackupsManage))

	backups.PUT("/policies/{id}", func(e *core.RequestEvent) error {
		return handleUpdatePolicy(e, backupStore)
	}).BindFunc(rbac.RequireCapability(constants.CapBackupsManage))

	backups.DELETE("/policies/{id}", func(e *core.RequestEvent) error {
		return handleDeletePolicy(e, backupStore)
	}).BindFunc(rbac.RequireCapability(constants.CapBackupsManage))

	backups.POST("/policies/{id}/trigger", func(e *core.RequestEvent) error {
		return handleTriggerBackup(e, backupSvc)
	}).BindFunc(rbac.RequireCapability(constants.CapBackupsManage))

	// ── Records ──────────────────────────────────────
	backups.GET("/records", func(e *core.RequestEvent) error {
		return handleListRecords(e, app)
	}).BindFunc(rbac.RequireCapability(constants.CapBackupsView))

	backups.POST("/records/{id}/restore", func(e *core.RequestEvent) error {
		return handleTriggerRestore(e, backupSvc)
	}).BindFunc(rbac.RequireCapability(constants.CapBackupsManage))

	// ── Maintenance ──────────────────────────────────
	backups.POST("/cleanup", func(e *core.RequestEvent) error {
		return handleRetentionCleanup(e, backupSvc)
	}).BindFunc(rbac.RequireCapability(constants.CapBackupsManage))
}

func handleListPolicies(e *core.RequestEvent, app core.App) error {
	serverID := e.Request.URL.Query().Get("server_id")
	filter := ""
	var params []dbx.Params
	if serverID != "" {
		filter = "server = {:serverId}"
		params = append(params, dbx.Params{"serverId": serverID})
	}

	records, err := app.FindRecordsByFilter(
		constants.CollBackupPolicies,
		filter,
		"-created",
		200, 0,
		params...,
	)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return e.JSON(http.StatusOK, records)
}

func handleCreatePolicy(e *core.RequestEvent, backupStore *store.BackupStore) error {
	var body struct {
		ServerID      string   `json:"server_id"`
		Volume        string   `json:"volume"`
		TargetType    string   `json:"target_type"`
		AppSource     string   `json:"app_source"`
		AppName       string   `json:"app_name"`
		Paths         []string `json:"paths"`
		Exclude       []string `json:"exclude"`
		MaxBytes      int64    `json:"max_bytes"`
		Schedule      string   `json:"schedule"`
		RetentionDays int      `json:"retention_days"`
		S3Bucket      string   `json:"s3_bucket"`
		S3Prefix      string   `json:"s3_prefix"`
		Enabled       bool     `json:"enabled"`
	}
	if err := e.BindBody(&body); err != nil || body.ServerID == "" || body.Schedule == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "server_id and schedule are required"})
	}

	if body.TargetType == "" {
		body.TargetType = "volume"
	}
	if err := services.ValidateBackupPolicy(body.TargetType, body.Volume, body.AppSource, body.AppName, body.Paths); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if body.RetentionDays <= 0 {
		body.RetentionDays = 30
	}

	id, err := backupStore.CreatePolicy(store.BackupPolicyInput{
		ServerID:      body.ServerID,
		Volume:        body.Volume,
		TargetType:    body.TargetType,
		AppSource:     body.AppSource,
		AppName:       body.AppName,
		Paths:         body.Paths,
		Exclude:       body.Exclude,
		MaxBytes:      body.MaxBytes,
		Schedule:      body.Schedule,
		RetentionDays: body.RetentionDays,
		S3Bucket:      body.S3Bucket,
		S3Prefix:      body.S3Prefix,
		Enabled:       body.Enabled,
	})
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return e.JSON(http.StatusCreated, map[string]string{"id": id})
}

func handleUpdatePolicy(e *core.RequestEvent, backupStore *store.BackupStore) error {
	policyID := e.Request.PathValue("id")

	var body map[string]any
	if err := e.BindBody(&body); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid body"})
	}

	existing, err := backupStore.GetPolicy(policyID)
	if err != nil {
		return e.JSON(http.StatusNotFound, map[string]string{"error": "policy not found"})
	}

	// Only allow updating known fields.
	allowed := map[string]bool{
		"schedule": true, "retention_days": true, "s3_bucket": true,
		"s3_prefix": true, "enabled": true, "volume": true,
		"target_type": true, "app_source": true, "app_name": true,
		"paths": true, "exclude": true, "max_bytes": true,
	}
	fields := make(map[string]any)
	for k, v := range body {
		if allowed[k] {
			fields[k] = v
		}
	}

	targetType := existing.GetString("target_type")
	volume := existing.GetString("volume")
	appSource := existing.GetString("app_source")
	appName := existing.GetString("app_name")
	paths := recordStringList(existing, "paths")
	if value, ok := fields["target_type"].(string); ok {
		targetType = value
	}
	if value, ok := fields["volume"].(string); ok {
		volume = value
	}
	if value, ok := fields["app_source"].(string); ok {
		appSource = value
	}
	if value, ok := fields["app_name"].(string); ok {
		appName = value
	}
	if value, ok := fields["paths"]; ok {
		paths = anyStringList(value)
	}
	if err := services.ValidateBackupPolicy(targetType, volume, appSource, appName, paths); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if err := backupStore.UpdatePolicy(policyID, fields); err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return e.JSON(http.StatusOK, map[string]string{"status": "updated"})
}

func recordStringList(record *core.Record, field string) []string {
	return anyStringList(record.Get(field))
}

func anyStringList(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok {
				out = append(out, text)
			}
		}
		return out
	case []byte:
		var list []string
		if err := json.Unmarshal(typed, &list); err == nil {
			return list
		}
	case string:
		var list []string
		if err := json.Unmarshal([]byte(typed), &list); err == nil {
			return list
		}
	}
	return nil
}

func handleDeletePolicy(e *core.RequestEvent, backupStore *store.BackupStore) error {
	policyID := e.Request.PathValue("id")

	if err := backupStore.DeletePolicy(policyID); err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return e.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

func handleTriggerBackup(e *core.RequestEvent, backupSvc *services.BackupService) error {
	policyID := e.Request.PathValue("id")

	userID := ""
	if e.Auth != nil {
		userID = e.Auth.Id
	}

	recordID, err := backupSvc.TriggerBackup(e.Request.Context(), policyID, userID, e.RealIP())
	if err != nil {
		switch {
		case err.Error() == "policy not found":
			return e.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		case err == services.ErrServerNotFound:
			return e.JSON(http.StatusNotFound, map[string]string{"error": "server not found"})
		case err == services.ErrServerOffline:
			return e.JSON(http.StatusServiceUnavailable, map[string]string{"error": "server offline"})
		default:
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	return e.JSON(http.StatusAccepted, map[string]string{"record_id": recordID, "status": "running"})
}

func handleListRecords(e *core.RequestEvent, app core.App) error {
	policyID := e.Request.URL.Query().Get("policy_id")
	serverID := e.Request.URL.Query().Get("server_id")

	filter := ""
	var params []dbx.Params
	if policyID != "" {
		filter = "policy = {:policyId}"
		params = append(params, dbx.Params{"policyId": policyID})
	} else if serverID != "" {
		filter = "server = {:serverId}"
		params = append(params, dbx.Params{"serverId": serverID})
	}

	records, err := app.FindRecordsByFilter(
		constants.CollBackupRecords,
		filter,
		"-started_at",
		200, 0,
		params...,
	)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return e.JSON(http.StatusOK, records)
}

func handleTriggerRestore(e *core.RequestEvent, backupSvc *services.BackupService) error {
	recordID := e.Request.PathValue("id")

	userID := ""
	if e.Auth != nil {
		userID = e.Auth.Id
	}

	if err := backupSvc.TriggerRestore(e.Request.Context(), recordID, userID, e.RealIP()); err != nil {
		switch err {
		case services.ErrServerNotFound:
			return e.JSON(http.StatusNotFound, map[string]string{"error": "server not found"})
		case services.ErrServerOffline:
			return e.JSON(http.StatusServiceUnavailable, map[string]string{"error": "server offline"})
		default:
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
	}

	return e.JSON(http.StatusAccepted, map[string]string{"status": "restoring"})
}

func handleRetentionCleanup(e *core.RequestEvent, backupSvc *services.BackupService) error {
	go backupSvc.RunRetentionCleanup()
	return e.JSON(http.StatusAccepted, map[string]string{"status": "cleanup started"})
}
