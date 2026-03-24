package grpc_handlers

import (
	"log/slog"

	"github.com/onflux-tech/dockeeper/internal/agent/executor"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// BackupHandler implements AgentBackupServiceServer on the agent side.
type BackupHandler struct {
	agentpb.UnimplementedAgentBackupServiceServer
	exec   executor.Executor
	logger *slog.Logger
}

// NewBackupHandler creates a backup handler with the given executor.
func NewBackupHandler(exec executor.Executor, logger *slog.Logger) *BackupHandler {
	return &BackupHandler{exec: exec, logger: logger}
}

// BackupVolume creates a backup of a volume and uploads to S3.
func (h *BackupHandler) BackupVolume(req *agentpb.BackupVolumeRequest, stream agentpb.AgentBackupService_BackupVolumeServer) error {
	h.logger.Info("starting volume backup", "volume", req.GetVolumeName(), "mountpoint", req.GetMountpoint())

	s3Cfg := protoToS3Config(req.GetS3())

	sendProgress := func(p executor.BackupProgressInfo) {
		_ = stream.Send(&agentpb.BackupProgress{
			Status:         p.Status,
			Message:        p.Message,
			BytesProcessed: p.BytesProcessed,
			TotalBytes:     p.TotalBytes,
		})
	}

	err := h.exec.BackupVolume(stream.Context(), req.GetMountpoint(), s3Cfg, sendProgress)
	if err != nil {
		h.logger.Error("volume backup failed", "volume", req.GetVolumeName(), "error", err)
		return stream.Send(&agentpb.BackupProgress{
			Status: "done",
			Done:   true,
			Error:  err.Error(),
		})
	}

	h.logger.Info("volume backup completed", "volume", req.GetVolumeName())
	return stream.Send(&agentpb.BackupProgress{
		Status: "done",
		Done:   true,
	})
}

// RestoreVolume downloads a backup from S3 and restores to a volume.
func (h *BackupHandler) RestoreVolume(req *agentpb.RestoreVolumeRequest, stream agentpb.AgentBackupService_RestoreVolumeServer) error {
	h.logger.Info("starting volume restore", "volume", req.GetVolumeName(), "mountpoint", req.GetMountpoint())

	s3Cfg := protoToS3Config(req.GetS3())

	sendProgress := func(p executor.BackupProgressInfo) {
		_ = stream.Send(&agentpb.BackupProgress{
			Status:         p.Status,
			Message:        p.Message,
			BytesProcessed: p.BytesProcessed,
			TotalBytes:     p.TotalBytes,
		})
	}

	err := h.exec.RestoreVolume(stream.Context(), req.GetMountpoint(), s3Cfg, sendProgress)
	if err != nil {
		h.logger.Error("volume restore failed", "volume", req.GetVolumeName(), "error", err)
		return stream.Send(&agentpb.BackupProgress{
			Status: "done",
			Done:   true,
			Error:  err.Error(),
		})
	}

	h.logger.Info("volume restore completed", "volume", req.GetVolumeName())
	return stream.Send(&agentpb.BackupProgress{
		Status: "done",
		Done:   true,
	})
}

func protoToS3Config(pb *agentpb.S3Config) executor.S3Config {
	if pb == nil {
		return executor.S3Config{}
	}
	return executor.S3Config{
		Endpoint:  pb.GetEndpoint(),
		Bucket:    pb.GetBucket(),
		Key:       pb.GetKey(),
		AccessKey: pb.GetAccessKey(),
		SecretKey: pb.GetSecretKey(),
		Region:    pb.GetRegion(),
		UseSSL:    pb.GetUseSsl(),
	}
}
