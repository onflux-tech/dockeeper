package store

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/jkdesenvolvedor/dockeeper/internal/shared/constants"
)

// BackupStore provides CRUD operations for backup_policies and backup_records.
type BackupStore struct {
	app core.App
}

// NewBackupStore creates a BackupStore.
func NewBackupStore(app core.App) *BackupStore {
	return &BackupStore{app: app}
}

// ── Policies ─────────────────────────────────────────

// CreatePolicy creates a new backup policy.
func (s *BackupStore) CreatePolicy(serverID, volumeName, schedule string, retentionDays int, s3Bucket, s3Prefix string, enabled bool) (string, error) {
	col, err := s.app.FindCachedCollectionByNameOrId(constants.CollBackupPolicies)
	if err != nil {
		return "", fmt.Errorf("collection %s: %w", constants.CollBackupPolicies, err)
	}

	rec := core.NewRecord(col)
	rec.Set("server", serverID)
	rec.Set("volume", volumeName)
	rec.Set("schedule", schedule)
	rec.Set("retention_days", retentionDays)
	rec.Set("s3_bucket", s3Bucket)
	rec.Set("s3_prefix", s3Prefix)
	rec.Set("enabled", enabled)

	if err := s.app.Save(rec); err != nil {
		return "", fmt.Errorf("save policy: %w", err)
	}

	return rec.Id, nil
}

// UpdatePolicy updates an existing backup policy fields.
func (s *BackupStore) UpdatePolicy(policyID string, fields map[string]any) error {
	rec, err := s.app.FindRecordById(constants.CollBackupPolicies, policyID)
	if err != nil {
		return fmt.Errorf("policy not found: %w", err)
	}

	for k, v := range fields {
		rec.Set(k, v)
	}

	if err := s.app.Save(rec); err != nil {
		return fmt.Errorf("update policy: %w", err)
	}

	return nil
}

// DeletePolicy removes a backup policy.
func (s *BackupStore) DeletePolicy(policyID string) error {
	rec, err := s.app.FindRecordById(constants.CollBackupPolicies, policyID)
	if err != nil {
		return fmt.Errorf("policy not found: %w", err)
	}

	if err := s.app.Delete(rec); err != nil {
		return fmt.Errorf("delete policy: %w", err)
	}

	return nil
}

// GetEnabledPolicies returns all enabled backup policies.
func (s *BackupStore) GetEnabledPolicies() ([]*core.Record, error) {
	records, err := s.app.FindRecordsByFilter(
		constants.CollBackupPolicies,
		"enabled = true",
		"created",
		0, 0,
	)
	if err != nil {
		return nil, fmt.Errorf("list enabled policies: %w", err)
	}

	return records, nil
}

// ── Records ──────────────────────────────────────────

// CreateBackupRecord creates a new backup record with status "running".
func (s *BackupStore) CreateBackupRecord(policyID, volumeName, serverID string) (string, error) {
	col, err := s.app.FindCachedCollectionByNameOrId(constants.CollBackupRecords)
	if err != nil {
		return "", fmt.Errorf("collection %s: %w", constants.CollBackupRecords, err)
	}

	rec := core.NewRecord(col)
	rec.Set("policy", policyID)
	rec.Set("volume_name", volumeName)
	rec.Set("server", serverID)
	rec.Set("status", "running")
	rec.Set("started_at", time.Now().UTC().Format(time.RFC3339))

	if err := s.app.Save(rec); err != nil {
		return "", fmt.Errorf("save backup record: %w", err)
	}

	return rec.Id, nil
}

// MarkBackupSuccess marks a backup record as successful.
func (s *BackupStore) MarkBackupSuccess(recordID, s3Key string, size int64) error {
	rec, err := s.app.FindRecordById(constants.CollBackupRecords, recordID)
	if err != nil {
		return fmt.Errorf("record not found: %w", err)
	}

	rec.Set("status", "success")
	rec.Set("s3_key", s3Key)
	rec.Set("size", size)
	rec.Set("completed_at", time.Now().UTC().Format(time.RFC3339))

	return s.app.Save(rec)
}

// MarkBackupFailed marks a backup record as failed with an error message.
func (s *BackupStore) MarkBackupFailed(recordID, errMsg string) error {
	rec, err := s.app.FindRecordById(constants.CollBackupRecords, recordID)
	if err != nil {
		return fmt.Errorf("record not found: %w", err)
	}

	rec.Set("status", "failed")
	rec.Set("error", errMsg)
	rec.Set("completed_at", time.Now().UTC().Format(time.RFC3339))

	return s.app.Save(rec)
}

// GetLastBackupForPolicy returns the most recent backup record for a policy.
func (s *BackupStore) GetLastBackupForPolicy(policyID string) (*core.Record, error) {
	records, err := s.app.FindRecordsByFilter(
		constants.CollBackupRecords,
		fmt.Sprintf("policy = '%s'", policyID),
		"-started_at",
		1, 0,
	)
	if err != nil || len(records) == 0 {
		return nil, err
	}

	return records[0], nil
}

// GetExpiredRecords returns backup records older than retentionDays for a given policy.
func (s *BackupStore) GetExpiredRecords(policyID string, retentionDays int) ([]*core.Record, error) {
	cutoff := time.Now().UTC().Add(-time.Duration(retentionDays) * 24 * time.Hour).Format(time.RFC3339)

	records, err := s.app.FindRecordsByFilter(
		constants.CollBackupRecords,
		fmt.Sprintf("policy = '%s' && status = 'success' && completed_at < '%s'", policyID, cutoff),
		"completed_at",
		0, 0,
	)
	if err != nil {
		return nil, fmt.Errorf("list expired records: %w", err)
	}

	return records, nil
}

// DeleteBackupRecord deletes a backup record.
func (s *BackupStore) DeleteBackupRecord(recordID string) error {
	rec, err := s.app.FindRecordById(constants.CollBackupRecords, recordID)
	if err != nil {
		return fmt.Errorf("record not found: %w", err)
	}

	if err := s.app.Delete(rec); err != nil {
		return fmt.Errorf("delete record: %w", err)
	}

	return nil
}

// CleanupExpiredBackups deletes records older than retention period and returns S3 keys to delete.
func (s *BackupStore) CleanupExpiredBackups(policyID string, retentionDays int) ([]string, error) {
	records, err := s.GetExpiredRecords(policyID, retentionDays)
	if err != nil {
		return nil, err
	}

	var keys []string
	for _, rec := range records {
		s3Key := rec.GetString("s3_key")
		if s3Key != "" {
			keys = append(keys, s3Key)
		}
		if delErr := s.app.Delete(rec); delErr != nil {
			slog.Warn("failed to delete expired backup record", "id", rec.Id, "error", delErr)
		}
	}

	return keys, nil
}
