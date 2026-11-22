package store

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/shared/constants"
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

type BackupPolicyInput struct {
	ServerID      string
	Volume        string
	TargetType    string
	AppSource     string
	AppName       string
	Paths         []string
	Exclude       []string
	MaxBytes      int64
	Schedule      string
	RetentionDays int
	S3Bucket      string
	S3Prefix      string
	Enabled       bool
}

// CreatePolicy creates a new backup policy.
func (s *BackupStore) CreatePolicy(input BackupPolicyInput) (string, error) {
	col, err := s.app.FindCachedCollectionByNameOrId(constants.CollBackupPolicies)
	if err != nil {
		return "", fmt.Errorf("collection %s: %w", constants.CollBackupPolicies, err)
	}

	rec := core.NewRecord(col)
	rec.Set("server", input.ServerID)
	rec.Set("volume", input.Volume)
	rec.Set("target_type", input.TargetType)
	rec.Set("app_source", input.AppSource)
	rec.Set("app_name", input.AppName)
	rec.Set("paths", input.Paths)
	rec.Set("exclude", input.Exclude)
	rec.Set("max_bytes", input.MaxBytes)
	rec.Set("schedule", input.Schedule)
	rec.Set("retention_days", input.RetentionDays)
	rec.Set("s3_bucket", input.S3Bucket)
	rec.Set("s3_prefix", input.S3Prefix)
	rec.Set("enabled", input.Enabled)

	if err := s.app.Save(rec); err != nil {
		return "", fmt.Errorf("save policy: %w", err)
	}

	return rec.Id, nil
}

func (s *BackupStore) GetPolicy(policyID string) (*core.Record, error) {
	return s.app.FindRecordById(constants.CollBackupPolicies, policyID)
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
	rec.Set("target_type", "volume")
	rec.Set("target_label", volumeName)
	rec.Set("status", "running")
	rec.Set("started_at", time.Now().UTC().Format(time.RFC3339))

	if err := s.app.Save(rec); err != nil {
		return "", fmt.Errorf("save backup record: %w", err)
	}

	return rec.Id, nil
}

func (s *BackupStore) CreateAppBackupRecord(policyID, serverID, appSource, appName, targetLabel string) (string, error) {
	col, err := s.app.FindCachedCollectionByNameOrId(constants.CollBackupRecords)
	if err != nil {
		return "", fmt.Errorf("collection %s: %w", constants.CollBackupRecords, err)
	}

	rec := core.NewRecord(col)
	rec.Set("policy", policyID)
	rec.Set("server", serverID)
	rec.Set("target_type", "host_app")
	rec.Set("app_source", appSource)
	rec.Set("app_name", appName)
	rec.Set("target_label", targetLabel)
	rec.Set("status", "running")
	rec.Set("started_at", time.Now().UTC().Format(time.RFC3339))

	if err := s.app.Save(rec); err != nil {
		return "", fmt.Errorf("save backup record: %w", err)
	}

	return rec.Id, nil
}

// MarkBackupSuccess marks a backup record as successful.
func (s *BackupStore) MarkBackupSuccess(recordID, s3Key string, size int64, checksum string) error {
	rec, err := s.app.FindRecordById(constants.CollBackupRecords, recordID)
	if err != nil {
		return fmt.Errorf("record not found: %w", err)
	}

	rec.Set("status", "success")
	rec.Set("s3_key", s3Key)
	rec.Set("size", size)
	rec.Set("completed_at", time.Now().UTC().Format(time.RFC3339))
	if checksum != "" {
		rec.Set("manifest", map[string]any{
			"checksum": checksum,
			"size":     size,
			"s3_key":   s3Key,
		})
	}

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
