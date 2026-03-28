package store

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// HealthStore provides CRUD for health_checks and health_check_logs.
type HealthStore struct {
	app core.App
}

// NewHealthStore creates a HealthStore.
func NewHealthStore(app core.App) *HealthStore {
	return &HealthStore{app: app}
}

// ── Health Checks ────────────────────────────────────────

// GetEnabledChecks returns all enabled health checks.
func (s *HealthStore) GetEnabledChecks() ([]*core.Record, error) {
	records, err := s.app.FindRecordsByFilter(
		constants.CollHealthChecks,
		"enabled = true",
		"next_check_at",
		0, 0,
	)
	if err != nil {
		return nil, fmt.Errorf("list enabled checks: %w", err)
	}
	return records, nil
}

// GetAllChecks returns all health checks, optionally filtered by server.
func (s *HealthStore) GetAllChecks(serverID string) ([]*core.Record, error) {
	filter := ""
	if serverID != "" {
		filter = fmt.Sprintf("server = '%s'", serverID)
	}
	records, err := s.app.FindRecordsByFilter(
		constants.CollHealthChecks,
		filter,
		"-created",
		0, 0,
	)
	if err != nil {
		return nil, fmt.Errorf("list checks: %w", err)
	}
	return records, nil
}

// GetCheckByID returns a health check by ID.
func (s *HealthStore) GetCheckByID(id string) (*core.Record, error) {
	return s.app.FindRecordById(constants.CollHealthChecks, id)
}

// CreateCheck creates a new health check with defaults.
func (s *HealthStore) CreateCheck(fields map[string]any) (string, error) {
	col, err := s.app.FindCachedCollectionByNameOrId(constants.CollHealthChecks)
	if err != nil {
		return "", fmt.Errorf("collection %s: %w", constants.CollHealthChecks, err)
	}

	rec := core.NewRecord(col)
	for k, v := range fields {
		rec.Set(k, v)
	}

	// Apply defaults for unset fields.
	defaults := map[string]any{
		"url_path":               "/",
		"expected_status":        200,
		"interval_seconds":       30,
		"timeout_seconds":        5,
		"failure_threshold":      3,
		"success_threshold":      1,
		"auto_restart":           true,
		"max_restarts":           3,
		"restart_window_minutes": 60,
		"enabled":                true,
		"status":                 "unknown",
		"consecutive_failures":   0,
	}
	for k, v := range defaults {
		if rec.Get(k) == nil || rec.Get(k) == "" || rec.Get(k) == 0 {
			rec.Set(k, v)
		}
	}

	now := time.Now().UTC()
	rec.Set("next_check_at", now.Format(time.RFC3339))

	if err := s.app.Save(rec); err != nil {
		return "", fmt.Errorf("save check: %w", err)
	}
	return rec.Id, nil
}

// UpdateCheck updates an existing health check.
func (s *HealthStore) UpdateCheck(id string, fields map[string]any) error {
	rec, err := s.app.FindRecordById(constants.CollHealthChecks, id)
	if err != nil {
		return fmt.Errorf("check not found: %w", err)
	}
	for k, v := range fields {
		rec.Set(k, v)
	}
	return s.app.Save(rec)
}

// DeleteCheck deletes a health check and its logs (via cascade).
func (s *HealthStore) DeleteCheck(id string) error {
	rec, err := s.app.FindRecordById(constants.CollHealthChecks, id)
	if err != nil {
		return fmt.Errorf("check not found: %w", err)
	}
	return s.app.Delete(rec)
}

// UpdateCheckStatus updates status, consecutive failures, and timing fields.
func (s *HealthStore) UpdateCheckStatus(id, status string, consecutiveFailures int, lastCheckAt, nextCheckAt time.Time) error {
	rec, err := s.app.FindRecordById(constants.CollHealthChecks, id)
	if err != nil {
		return fmt.Errorf("check not found: %w", err)
	}
	rec.Set("status", status)
	rec.Set("consecutive_failures", consecutiveFailures)
	rec.Set("last_check_at", lastCheckAt.Format(time.RFC3339))
	rec.Set("next_check_at", nextCheckAt.Format(time.RFC3339))
	return s.app.Save(rec)
}

// ── Health Check Logs ────────────────────────────────────

// LogCheckResult saves a health check result.
func (s *HealthStore) LogCheckResult(checkID string, statusCode int, responseTimeMs int64, success bool, errMsg string, checkedAt time.Time) error {
	col, err := s.app.FindCachedCollectionByNameOrId(constants.CollHealthCheckLogs)
	if err != nil {
		return fmt.Errorf("collection %s: %w", constants.CollHealthCheckLogs, err)
	}

	rec := core.NewRecord(col)
	rec.Set("health_check", checkID)
	rec.Set("status_code", statusCode)
	rec.Set("response_time_ms", responseTimeMs)
	rec.Set("success", success)
	rec.Set("error", errMsg)
	rec.Set("checked_at", checkedAt.Format(time.RFC3339))

	return s.app.Save(rec)
}

// GetCheckLogs returns paginated logs for a health check.
func (s *HealthStore) GetCheckLogs(checkID string, page, perPage int) ([]*core.Record, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 50
	}

	filter := fmt.Sprintf("health_check = '%s'", checkID)

	// Count total.
	all, err := s.app.FindRecordsByFilter(constants.CollHealthCheckLogs, filter, "", 0, 0)
	if err != nil {
		return nil, 0, fmt.Errorf("count logs: %w", err)
	}
	total := len(all)

	// Paginated fetch.
	offset := (page - 1) * perPage
	records, err := s.app.FindRecordsByFilter(
		constants.CollHealthCheckLogs,
		filter,
		"-checked_at",
		perPage, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list logs: %w", err)
	}

	return records, total, nil
}

// CleanupOldLogs deletes health check logs older than the given duration.
func (s *HealthStore) CleanupOldLogs(retention time.Duration) (int, error) {
	cutoff := time.Now().UTC().Add(-retention).Format(time.RFC3339)
	filter := fmt.Sprintf("checked_at < '%s'", cutoff)

	records, err := s.app.FindRecordsByFilter(constants.CollHealthCheckLogs, filter, "", 0, 0)
	if err != nil {
		return 0, fmt.Errorf("find old logs: %w", err)
	}

	for _, rec := range records {
		if err := s.app.Delete(rec); err != nil {
			return 0, fmt.Errorf("delete log %s: %w", rec.Id, err)
		}
	}
	return len(records), nil
}
