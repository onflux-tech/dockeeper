package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/channel"
	"github.com/onflux-tech/dockeeper/internal/manager/events"
	"github.com/onflux-tech/dockeeper/internal/manager/store"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

var ErrInvalidBackupTarget = errors.New("invalid backup target")

const backupOperationTimeout = 30 * time.Minute

// BackupService orchestrates backup/restore operations.
type BackupService struct {
	app      core.App
	store    *store.BackupStore
	audit    *store.AuditStore
	registry *channel.Registry
	bus      *events.Bus
}

// NewBackupService creates a BackupService.
func NewBackupService(app core.App, backupStore *store.BackupStore, audit *store.AuditStore, registry *channel.Registry, bus *events.Bus) *BackupService {
	return &BackupService{app: app, store: backupStore, audit: audit, registry: registry, bus: bus}
}

// TriggerBackup triggers a backup for the given policy.
func (s *BackupService) TriggerBackup(ctx context.Context, policyID, userID, ip string) (string, error) {
	policy, err := s.app.FindRecordById(constants.CollBackupPolicies, policyID)
	if err != nil {
		return "", fmt.Errorf("policy not found")
	}

	targetType := policy.GetString("target_type")
	if targetType == "" {
		targetType = "volume"
	}
	if targetType == "host_app" {
		return s.triggerAppBackup(policy, userID, ip)
	}

	serverID := policy.GetString("server")
	volumeName := policy.GetString("volume")
	s3Bucket := policy.GetString("s3_bucket")
	s3Prefix := policy.GetString("s3_prefix")

	// Resolve volume → get mountpoint from discovery data.
	volumes, err := s.app.FindRecordsByFilter(
		constants.CollVolumes,
		fmt.Sprintf("server = '%s' && name = '%s'", serverID, volumeName),
		"", 1, 0,
	)
	if err != nil || len(volumes) == 0 {
		return "", fmt.Errorf("volume %s not found on server", volumeName)
	}
	mountpoint := volumes[0].GetString("mountpoint")

	// Resolve server — verify it's online.
	server, err := s.app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		return "", ErrServerNotFound
	}
	if server.GetString("status") != "online" {
		return "", ErrServerOffline
	}

	// Get S3 config from settings.
	s3Cfg, err := s.getS3Settings()
	if err != nil {
		return "", fmt.Errorf("S3 settings not configured: %w", err)
	}

	// Create backup record.
	recordID, err := s.store.CreateBackupRecord(policyID, volumeName, serverID)
	if err != nil {
		return "", err
	}

	// Build S3 key.
	timestamp := time.Now().UTC().Format("20060102-150405")
	s3Key := fmt.Sprintf("%s/%s/%s.tar.gz", s3Prefix, volumeName, timestamp)

	// Dispatch to agent (async via goroutine for long-running).
	go func() {
		bgCtx := context.Background()
		backupErr := s.executeBackup(bgCtx, serverID, volumeName, mountpoint, s3Cfg, s3Bucket, s3Key)
		if backupErr != nil {
			slog.Error("backup failed", "policy", policyID, "volume", volumeName, "error", backupErr)
			_ = s.store.MarkBackupFailed(recordID, backupErr.Error())
			s.publishBackupEvent(events.BackupFailed, recordID, map[string]any{
				"target_label": volumeName,
				"error":        backupErr.Error(),
			})
		} else {
			// Get the size from S3.
			size, _ := s.getS3ObjectInfo(bgCtx, s3Cfg, s3Bucket, s3Key)
			_ = s.store.MarkBackupSuccess(recordID, s3Key, size, "")
			slog.Info("backup succeeded", "policy", policyID, "volume", volumeName, "s3_key", s3Key)
			s.publishBackupEvent(events.BackupCompleted, recordID, map[string]any{
				"target_label": volumeName,
				"size":         size,
				"s3_key":       s3Key,
			})
		}
	}()

	s.audit.Log("backup.trigger", userID, "backup_policy", policyID, ip, map[string]any{
		"volume": volumeName, "server_id": serverID,
	})

	return recordID, nil
}

// TriggerRestore restores a backup to the original volume.
func (s *BackupService) TriggerRestore(ctx context.Context, backupRecordID, userID, ip string) error {
	rec, err := s.app.FindRecordById(constants.CollBackupRecords, backupRecordID)
	if err != nil {
		return fmt.Errorf("backup record not found")
	}

	if rec.GetString("status") != "success" {
		return fmt.Errorf("can only restore successful backups")
	}

	if rec.GetString("target_type") == "host_app" {
		return s.triggerAppRestore(rec, userID, ip)
	}

	serverID := rec.GetString("server")
	volumeName := rec.GetString("volume_name")
	s3Key := rec.GetString("s3_key")

	// Resolve volume mountpoint.
	volumes, err := s.app.FindRecordsByFilter(
		constants.CollVolumes,
		fmt.Sprintf("server = '%s' && name = '%s'", serverID, volumeName),
		"", 1, 0,
	)
	if err != nil || len(volumes) == 0 {
		return fmt.Errorf("volume %s not found on server", volumeName)
	}
	mountpoint := volumes[0].GetString("mountpoint")

	// Resolve server.
	server, err := s.app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		return ErrServerNotFound
	}
	if server.GetString("status") != "online" {
		return ErrServerOffline
	}

	// Get S3 config.
	s3Cfg, err := s.getS3Settings()
	if err != nil {
		return fmt.Errorf("S3 settings not configured: %w", err)
	}

	// Get policy for bucket info.
	policyID := rec.GetString("policy")
	policy, err := s.app.FindRecordById(constants.CollBackupPolicies, policyID)
	if err != nil {
		return fmt.Errorf("policy not found")
	}
	s3Bucket := policy.GetString("s3_bucket")

	// Execute restore asynchronously.
	go func() {
		bgCtx := context.Background()
		restoreErr := s.executeRestore(bgCtx, serverID, volumeName, mountpoint, s3Cfg, s3Bucket, s3Key)
		if restoreErr != nil {
			slog.Error("restore failed", "record", backupRecordID, "error", restoreErr)
			s.publishBackupEvent(events.BackupFailed, backupRecordID, map[string]any{
				"target_label": volumeName,
				"error":        restoreErr.Error(),
			})
		} else {
			slog.Info("restore succeeded", "record", backupRecordID, "volume", volumeName)
			s.publishBackupEvent(events.BackupRestored, backupRecordID, map[string]any{
				"target_label": volumeName,
				"s3_key":       s3Key,
			})
		}
	}()

	s.audit.Log("backup.restore", userID, "backup_record", backupRecordID, ip, map[string]any{
		"volume": volumeName, "server_id": serverID, "s3_key": s3Key,
	})

	return nil
}

// RunRetentionCleanup cleans up expired backups for all policies.
func (s *BackupService) RunRetentionCleanup() {
	policies, err := s.store.GetEnabledPolicies()
	if err != nil {
		slog.Error("retention cleanup: failed to list policies", "error", err)
		return
	}

	s3Cfg, err := s.getS3Settings()
	if err != nil {
		slog.Warn("retention cleanup: S3 settings not configured, skipping")
		return
	}

	for _, policy := range policies {
		retentionDays := policy.GetInt("retention_days")
		if retentionDays <= 0 {
			continue
		}

		s3Bucket := policy.GetString("s3_bucket")
		keys, cleanErr := s.store.CleanupExpiredBackups(policy.Id, retentionDays)
		if cleanErr != nil {
			slog.Error("retention cleanup failed", "policy", policy.Id, "error", cleanErr)
			continue
		}

		// Delete S3 objects.
		for _, key := range keys {
			if delErr := s.deleteS3Object(context.Background(), s3Cfg, s3Bucket, key); delErr != nil {
				slog.Warn("failed to delete S3 object", "key", key, "error", delErr)
			}
		}

		if len(keys) > 0 {
			slog.Info("retention cleanup done", "policy", policy.Id, "deleted", len(keys))
		}
	}
}

// ── Internal helpers ─────────────────────────────────

type s3Settings struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Region    string
	UseSSL    bool
}

func (s *BackupService) getS3Settings() (*s3Settings, error) {
	fields := []string{"s3_endpoint", "s3_access_key", "s3_secret_key", "s3_region", "s3_use_ssl"}
	settings := make(map[string]string)

	for _, key := range fields {
		records, err := s.app.FindRecordsByFilter(
			constants.CollSettings,
			fmt.Sprintf("key = '%s'", key),
			"", 1, 0,
		)
		if err != nil || len(records) == 0 {
			if key == "s3_endpoint" || key == "s3_access_key" || key == "s3_secret_key" {
				return nil, fmt.Errorf("missing required setting: %s", key)
			}
			continue
		}
		settings[key] = fmt.Sprintf("%v", records[0].Get("value"))
	}

	return &s3Settings{
		Endpoint:  settings["s3_endpoint"],
		AccessKey: settings["s3_access_key"],
		SecretKey: settings["s3_secret_key"],
		Region:    settings["s3_region"],
		UseSSL:    settings["s3_use_ssl"] == "true",
	}, nil
}

func (s *BackupService) executeBackup(ctx context.Context, serverID, volumeName, mountpoint string, s3Cfg *s3Settings, s3Bucket, s3Key string) error {
	ch, ok := s.registry.Get(serverID)
	if !ok {
		return fmt.Errorf("no active channel for server %s", serverID)
	}

	streamID := channel.NewRequestID()
	signal := &agentpb.OpenDataStreamSignal{
		StreamId:   streamID,
		StreamType: agentpb.DataStreamType_DATA_STREAM_TYPE_BACKUP,
		Params: &agentpb.OpenDataStreamSignal_BackupVolume{
			BackupVolume: &agentpb.BackupVolumeRequest{
				VolumeName: volumeName,
				Mountpoint: mountpoint,
				S3: &agentpb.S3Config{
					Endpoint: s3Cfg.Endpoint, Bucket: s3Bucket, Key: s3Key,
					AccessKey: s3Cfg.AccessKey, SecretKey: s3Cfg.SecretKey,
					Region: s3Cfg.Region, UseSsl: s3Cfg.UseSSL,
				},
			},
		},
	}

	streamCh, err := ch.SignalOpenDataStream(ctx, signal)
	if err != nil {
		return fmt.Errorf("signal backup stream: %w", err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case stream, ok := <-streamCh:
		if !ok {
			return fmt.Errorf("backup stream channel closed")
		}
		for {
			msg, recvErr := stream.Recv()
			if recvErr != nil {
				return fmt.Errorf("backup stream: %w", recvErr)
			}
			if complete := msg.GetComplete(); complete != nil {
				return nil
			}
			if errPayload := msg.GetError(); errPayload != nil {
				return fmt.Errorf("agent backup error: %s", errPayload.GetMessage())
			}
			// Progress messages — log debug
			if p := msg.GetProgress(); p != nil {
				slog.Debug("backup progress", "volume", volumeName, "status", p.GetStatus())
			}
		}
	}
}

func (s *BackupService) executeRestore(ctx context.Context, serverID, volumeName, mountpoint string, s3Cfg *s3Settings, s3Bucket, s3Key string) error {
	ch, ok := s.registry.Get(serverID)
	if !ok {
		return fmt.Errorf("no active channel for server %s", serverID)
	}

	streamID := channel.NewRequestID()
	signal := &agentpb.OpenDataStreamSignal{
		StreamId:   streamID,
		StreamType: agentpb.DataStreamType_DATA_STREAM_TYPE_RESTORE,
		Params: &agentpb.OpenDataStreamSignal_RestoreVolume{
			RestoreVolume: &agentpb.RestoreVolumeRequest{
				VolumeName: volumeName,
				Mountpoint: mountpoint,
				S3: &agentpb.S3Config{
					Endpoint: s3Cfg.Endpoint, Bucket: s3Bucket, Key: s3Key,
					AccessKey: s3Cfg.AccessKey, SecretKey: s3Cfg.SecretKey,
					Region: s3Cfg.Region, UseSsl: s3Cfg.UseSSL,
				},
			},
		},
	}

	streamCh, err := ch.SignalOpenDataStream(ctx, signal)
	if err != nil {
		return fmt.Errorf("signal restore stream: %w", err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case stream, ok := <-streamCh:
		if !ok {
			return fmt.Errorf("restore stream channel closed")
		}
		for {
			msg, recvErr := stream.Recv()
			if recvErr != nil {
				return fmt.Errorf("restore stream: %w", recvErr)
			}
			if complete := msg.GetComplete(); complete != nil {
				return nil
			}
			if errPayload := msg.GetError(); errPayload != nil {
				return fmt.Errorf("agent restore error: %s", errPayload.GetMessage())
			}
			if p := msg.GetProgress(); p != nil {
				slog.Debug("restore progress", "volume", volumeName, "status", p.GetStatus())
			}
		}
	}
}

func (s *BackupService) getS3ObjectInfo(ctx context.Context, s3Cfg *s3Settings, bucket, key string) (int64, string) {
	client, err := minio.New(s3Cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s3Cfg.AccessKey, s3Cfg.SecretKey, ""),
		Secure: s3Cfg.UseSSL,
		Region: s3Cfg.Region,
	})
	if err != nil {
		return 0, ""
	}

	info, err := client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return 0, ""
	}

	return info.Size, info.Metadata.Get("X-Amz-Meta-Manifest-Checksum")
}

func (s *BackupService) deleteS3Object(ctx context.Context, s3Cfg *s3Settings, bucket, key string) error {
	client, err := minio.New(s3Cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s3Cfg.AccessKey, s3Cfg.SecretKey, ""),
		Secure: s3Cfg.UseSSL,
		Region: s3Cfg.Region,
	})
	if err != nil {
		return err
	}

	return client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
}

var deniedBackupPaths = []string{
	"/proc", "/sys", "/dev", "/boot", "/etc/shadow", "/etc/passwd", "/etc/sudoers",
}

func ValidateBackupPolicy(targetType, volume, appSource, appName string, paths []string) error {
	if targetType == "" {
		targetType = "volume"
	}
	switch targetType {
	case "volume":
		if strings.TrimSpace(volume) == "" {
			return fmt.Errorf("%w: volume is required", ErrInvalidBackupTarget)
		}
	case "host_app":
		if appSource == "" || appName == "" {
			return fmt.Errorf("%w: app source and name are required", ErrInvalidBackupTarget)
		}
		if len(paths) == 0 {
			return fmt.Errorf("%w: at least one path is required", ErrInvalidBackupTarget)
		}
		if len(paths) > 20 {
			return fmt.Errorf("%w: too many paths", ErrInvalidBackupTarget)
		}
		for _, path := range paths {
			if !strings.HasPrefix(path, "/") || path == "/" || strings.Contains(path, "..") {
				return fmt.Errorf("%w: unsafe path %s", ErrInvalidBackupTarget, path)
			}
			for _, denied := range deniedBackupPaths {
				if path == denied || strings.HasPrefix(path, denied+"/") {
					return fmt.Errorf("%w: denied path %s", ErrInvalidBackupTarget, path)
				}
			}
		}
	default:
		return fmt.Errorf("%w: unknown target type %s", ErrInvalidBackupTarget, targetType)
	}
	return nil
}

func recordStringList(rec *core.Record, field string) []string {
	switch typed := rec.Get(field).(type) {
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

func sanitizeS3Key(name string) string {
	return strings.NewReplacer("/", "-", "\\", "-", " ", "-").Replace(name)
}

func s3ConfigFor(cfg *s3Settings, bucket, key string) *agentpb.S3Config {
	return &agentpb.S3Config{
		Endpoint:  cfg.Endpoint,
		Bucket:    bucket,
		Key:       key,
		AccessKey: cfg.AccessKey,
		SecretKey: cfg.SecretKey,
		Region:    cfg.Region,
		UseSsl:    cfg.UseSSL,
	}
}

func (s *BackupService) triggerAppBackup(policy *core.Record, userID, ip string) (string, error) {
	policyID := policy.Id
	serverID := policy.GetString("server")
	appSource := policy.GetString("app_source")
	appName := policy.GetString("app_name")
	paths := recordStringList(policy, "paths")
	exclude := recordStringList(policy, "exclude")
	maxBytes := int64(policy.GetInt("max_bytes"))

	if err := ValidateBackupPolicy("host_app", "", appSource, appName, paths); err != nil {
		return "", err
	}

	if _, err := s.app.FindFirstRecordByFilter(
		constants.CollHostApps,
		"server = {:server} && source = {:source} && name = {:name}",
		map[string]any{"server": serverID, "source": appSource, "name": appName},
	); err != nil {
		return "", fmt.Errorf("host app %s/%s not found on server", appSource, appName)
	}

	server, err := s.app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		return "", ErrServerNotFound
	}
	if server.GetString("status") != "online" {
		return "", ErrServerOffline
	}

	s3Cfg, err := s.getS3Settings()
	if err != nil {
		return "", fmt.Errorf("S3 settings not configured: %w", err)
	}

	label := appSource + "/" + appName
	recordID, err := s.store.CreateAppBackupRecord(policyID, serverID, appSource, appName, label)
	if err != nil {
		return "", err
	}

	timestamp := time.Now().UTC().Format("20060102-150405")
	s3Bucket := policy.GetString("s3_bucket")
	s3Key := fmt.Sprintf("%s/%s/%s.tar.gz", strings.TrimSuffix(policy.GetString("s3_prefix"), "/"), sanitizeS3Key(appName), timestamp)

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), backupOperationTimeout)
		defer cancel()
		backupErr := s.executeAppBackup(bgCtx, serverID, appSource, appName, paths, exclude, maxBytes, s3Cfg, s3Bucket, s3Key)
		if backupErr != nil {
			slog.Error("app backup failed", "policy", policyID, "app", label, "error", backupErr)
			_ = s.store.MarkBackupFailed(recordID, backupErr.Error())
			s.publishBackupEvent(events.BackupFailed, recordID, map[string]any{
				"target_label": label,
				"error":        backupErr.Error(),
			})
			return
		}
		size, checksum := s.getS3ObjectInfo(bgCtx, s3Cfg, s3Bucket, s3Key)
		_ = s.store.MarkBackupSuccess(recordID, s3Key, size, checksum)
		slog.Info("app backup succeeded", "policy", policyID, "app", label, "s3_key", s3Key)
		s.publishBackupEvent(events.BackupCompleted, recordID, map[string]any{
			"target_label": label,
			"size":         size,
			"s3_key":       s3Key,
		})
	}()

	s.audit.Log("backup.trigger", userID, "backup_policy", policyID, ip, map[string]any{
		"target": label, "server_id": serverID,
	})

	return recordID, nil
}

func (s *BackupService) executeAppBackup(ctx context.Context, serverID, appSource, appName string, paths, exclude []string, maxBytes int64, s3Cfg *s3Settings, s3Bucket, s3Key string) error {
	ch, ok := s.registry.Get(serverID)
	if !ok {
		return fmt.Errorf("no active channel for server %s", serverID)
	}

	signal := &agentpb.OpenDataStreamSignal{
		StreamId:   channel.NewRequestID(),
		StreamType: agentpb.DataStreamType_DATA_STREAM_TYPE_BACKUP_APP,
		Params: &agentpb.OpenDataStreamSignal_BackupApp{
			BackupApp: &agentpb.BackupAppRequest{
				ServerId:  serverID,
				AppSource: appSource,
				AppName:   appName,
				Paths:     paths,
				Exclude:   exclude,
				MaxBytes:  uint64(maxBytes),
				S3:        s3ConfigFor(s3Cfg, s3Bucket, s3Key),
			},
		},
	}

	return s.consumeDataStream(ctx, ch, signal, "backup")
}

func (s *BackupService) triggerAppRestore(rec *core.Record, userID, ip string) error {
	serverID := rec.GetString("server")
	appSource := rec.GetString("app_source")
	appName := rec.GetString("app_name")
	s3Key := rec.GetString("s3_key")
	recordID := rec.Id

	server, err := s.app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		return ErrServerNotFound
	}
	if server.GetString("status") != "online" {
		return ErrServerOffline
	}

	policy, err := s.app.FindRecordById(constants.CollBackupPolicies, rec.GetString("policy"))
	if err != nil {
		return fmt.Errorf("policy not found")
	}
	paths := recordStringList(policy, "paths")
	if err := ValidateBackupPolicy("host_app", "", appSource, appName, paths); err != nil {
		return err
	}

	s3Cfg, err := s.getS3Settings()
	if err != nil {
		return fmt.Errorf("S3 settings not configured: %w", err)
	}

	s3Bucket := policy.GetString("s3_bucket")
	label := appSource + "/" + appName

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), backupOperationTimeout)
		defer cancel()
		restoreErr := s.executeAppRestore(bgCtx, serverID, recordID, appSource, appName, paths, s3Cfg, s3Bucket, s3Key)
		if restoreErr != nil {
			slog.Error("app restore failed", "record", recordID, "error", restoreErr)
			s.publishBackupEvent(events.BackupFailed, recordID, map[string]any{
				"target_label": label,
				"error":        restoreErr.Error(),
			})
			return
		}
		slog.Info("app restore succeeded", "record", recordID, "app", label)
		s.publishBackupEvent(events.BackupRestored, recordID, map[string]any{
			"target_label": label,
			"s3_key":       s3Key,
		})
	}()

	s.audit.Log("backup.restore", userID, "backup_record", recordID, ip, map[string]any{
		"target": label, "server_id": serverID, "s3_key": s3Key,
	})

	return nil
}

func (s *BackupService) executeAppRestore(ctx context.Context, serverID, recordID, appSource, appName string, paths []string, s3Cfg *s3Settings, s3Bucket, s3Key string) error {
	ch, ok := s.registry.Get(serverID)
	if !ok {
		return fmt.Errorf("no active channel for server %s", serverID)
	}

	signal := &agentpb.OpenDataStreamSignal{
		StreamId:   channel.NewRequestID(),
		StreamType: agentpb.DataStreamType_DATA_STREAM_TYPE_RESTORE_APP,
		Params: &agentpb.OpenDataStreamSignal_RestoreApp{
			RestoreApp: &agentpb.RestoreAppRequest{
				ServerId:  serverID,
				RecordId:  recordID,
				AppSource: appSource,
				AppName:   appName,
				Paths:     paths,
				S3Key:     s3Key,
				S3:        s3ConfigFor(s3Cfg, s3Bucket, s3Key),
			},
		},
	}

	return s.consumeDataStream(ctx, ch, signal, "restore")
}

func (s *BackupService) consumeDataStream(ctx context.Context, ch *channel.AgentChannel, signal *agentpb.OpenDataStreamSignal, label string) error {
	streamCh, err := ch.SignalOpenDataStream(ctx, signal)
	if err != nil {
		return fmt.Errorf("signal %s stream: %w", label, err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case stream, ok := <-streamCh:
		if !ok {
			return fmt.Errorf("%s stream channel closed", label)
		}
		for {
			msg, recvErr := stream.Recv()
			if recvErr != nil {
				return fmt.Errorf("%s stream: %w", label, recvErr)
			}
			if complete := msg.GetComplete(); complete != nil {
				return nil
			}
			if errPayload := msg.GetError(); errPayload != nil {
				return fmt.Errorf("agent %s error: %s", label, errPayload.GetMessage())
			}
			if progress := msg.GetProgress(); progress != nil {
				slog.Debug(label+" progress", "status", progress.GetStatus())
			}
		}
	}
}

func (s *BackupService) publishBackupEvent(eventType events.EventType, recordID string, details map[string]any) {
	if s.bus == nil {
		return
	}
	record, err := s.app.FindRecordById(constants.CollBackupRecords, recordID)
	if err != nil {
		return
	}

	event := events.Event{
		Type:          eventType,
		Severity:      events.SeverityFor(eventType),
		ServerID:      record.GetString("server"),
		ContainerName: record.GetString("target_label"),
		Details:       details,
		Timestamp:     time.Now().UTC(),
	}
	if server, err := s.app.FindRecordById(constants.CollServers, event.ServerID); err == nil {
		event.ServerName = server.GetString("name")
		if event.ServerName == "" {
			event.ServerName = server.GetString("hostname")
		}
	}
	s.bus.Publish(event)
}
