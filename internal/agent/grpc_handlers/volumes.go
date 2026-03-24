package grpc_handlers

import (
	"context"
	"log/slog"

	"github.com/jkdesenvolvedor/dockeeper/internal/agent/executor"
	agentpb "github.com/jkdesenvolvedor/dockeeper/proto/gen/dockeeper/agent/v1"
)

// VolumeHandler implements AgentVolumeServiceServer on the agent side.
type VolumeHandler struct {
	agentpb.UnimplementedAgentVolumeServiceServer
	exec   executor.Executor
	logger *slog.Logger
}

// NewVolumeHandler creates a volume handler with the given executor.
func NewVolumeHandler(exec executor.Executor, logger *slog.Logger) *VolumeHandler {
	return &VolumeHandler{exec: exec, logger: logger}
}

// CreateVolume creates a Docker volume.
func (h *VolumeHandler) CreateVolume(ctx context.Context, req *agentpb.CreateVolumeRequest) (*agentpb.CreateVolumeResponse, error) {
	name := req.GetName()
	h.logger.Info("creating volume", "name", name, "driver", req.GetDriver())

	volName, err := h.exec.CreateVolume(ctx, name, req.GetDriver(), req.GetLabels())
	if err != nil {
		h.logger.Warn("volume create failed", "name", name, "error", err)
		return &agentpb.CreateVolumeResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	h.logger.Info("volume created", "name", volName)
	return &agentpb.CreateVolumeResponse{
		Name:    volName,
		Success: true,
	}, nil
}

// RemoveVolume removes a Docker volume.
func (h *VolumeHandler) RemoveVolume(ctx context.Context, req *agentpb.RemoveVolumeRequest) (*agentpb.RemoveVolumeResponse, error) {
	name := req.GetName()
	h.logger.Info("removing volume", "name", name, "force", req.GetForce())

	if err := h.exec.RemoveVolume(ctx, name, req.GetForce()); err != nil {
		h.logger.Warn("volume remove failed", "name", name, "error", err)
		return &agentpb.RemoveVolumeResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	h.logger.Info("volume removed", "name", name)
	return &agentpb.RemoveVolumeResponse{Success: true}, nil
}

// InspectVolume inspects a Docker volume and returns its details.
func (h *VolumeHandler) InspectVolume(ctx context.Context, req *agentpb.InspectVolumeRequest) (*agentpb.InspectVolumeResponse, error) {
	name := req.GetName()
	h.logger.Info("inspecting volume", "name", name)

	result, err := h.exec.InspectVolume(ctx, name)
	if err != nil {
		h.logger.Warn("volume inspect failed", "name", name, "error", err)
		return &agentpb.InspectVolumeResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &agentpb.InspectVolumeResponse{
		Name:       result.Name,
		Driver:     result.Driver,
		Mountpoint: result.Mountpoint,
		SizeBytes:  result.SizeBytes,
		Labels:     result.Labels,
		CreatedAt:  result.CreatedAt,
		Success:    true,
	}, nil
}
