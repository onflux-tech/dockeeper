package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/channel"
	"github.com/onflux-tech/dockeeper/internal/manager/store"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// BackupService orchestrates backup/restore operations.
type BackupService struct {
	app      core.App
	store    *store.BackupStore
	audit    *store.AuditStore
	registry *channel.Registry
}

// NewBackupService creates a BackupService.
func NewBackupService(app core.App, backupStore *store.BackupStore, audit *store.AuditStore, registry *channel.Registry) *BackupService {
	return &BackupService{app: app, store: backupStore, audit: audit, registry: registry}
}

// TriggerBackup triggers a backup for the given policy.
func (s *BackupService) TriggerBackup(ctx context.Context, policyID, userID, ip string) (string, error) {
	policy, err := s.app.FindRecordById(constants.CollBackupPolicies, policyID)
	if err != nil {
		return "", fmt.Errorf("policy not found")
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
		} else {
			// Get the size from S3.
			size := s.getS3ObjectSize(bgCtx, s3Cfg, s3Bucket, s3Key)
			_ = s.store.MarkBackupSuccess(recordID, s3Key, size)
			slog.Info("backup succeeded", "policy", policyID, "volume", volumeName, "s3_key", s3Key)
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
		} else {
			slog.Info("restore succeeded", "record", backupRecordID, "volume", volumeName)
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

func (s *BackupService) getS3ObjectSize(ctx context.Context, s3Cfg *s3Settings, bucket, key string) int64 {
	client, err := minio.New(s3Cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s3Cfg.AccessKey, s3Cfg.SecretKey, ""),
		Secure: s3Cfg.UseSSL,
		Region: s3Cfg.Region,
	})
	if err != nil {
		return 0
	}

	info, err := client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return 0
	}

	return info.Size
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

