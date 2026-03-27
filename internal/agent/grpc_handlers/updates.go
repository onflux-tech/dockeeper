package grpc_handlers

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"time"

	"github.com/onflux-tech/dockeeper/internal/agent/executor"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// UpdateHandler implements AgentUpdateServiceServer on the agent side.
type UpdateHandler struct {
	agentpb.UnimplementedAgentUpdateServiceServer
	exec   executor.Executor
	logger *slog.Logger
}

// NewUpdateHandler creates an update handler.
func NewUpdateHandler(exec executor.Executor, logger *slog.Logger) *UpdateHandler {
	return &UpdateHandler{exec: exec, logger: logger}
}

// CheckImageDigest inspects the local image digest for a given reference.
func (h *UpdateHandler) CheckImageDigest(ctx context.Context, req *agentpb.CheckImageDigestRequest) (*agentpb.CheckImageDigestResponse, error) {
	imageRef := req.GetImageRef()
	h.logger.Info("checking image digest", "image", imageRef)

	digest, err := h.exec.CheckImageDigest(ctx, imageRef)
	if err != nil {
		h.logger.Warn("digest check failed", "image", imageRef, "error", err)
		return &agentpb.CheckImageDigestResponse{}, nil
	}

	return &agentpb.CheckImageDigestResponse{
		CurrentDigest: digest,
	}, nil
}

// ExecuteContainerUpdate performs the full update saga with progress streaming.
func (h *UpdateHandler) ExecuteContainerUpdate(req *agentpb.ExecuteContainerUpdateRequest, stream agentpb.AgentUpdateService_ExecuteContainerUpdateServer) error {
	containerID := req.GetContainerId()
	newImage := req.GetNewImage()
	registryAuth := req.GetRegistryAuth()

	h.logger.Info("starting container update", "container", containerID, "new_image", newImage)

	send := func(step, message string, progress float32) {
		_ = stream.Send(&agentpb.ContainerUpdateProgress{
			Step:     step,
			Message:  message,
			Progress: progress,
		})
	}

	// Step 1: Snapshot current container config
	send("snapshot", "Capturing container configuration...", 0)
	snap, err := h.exec.SnapshotContainer(stream.Context(), containerID)
	if err != nil {
		send("failed", "Failed to snapshot container: "+err.Error(), 0)
		return nil
	}

	// Step 2: Pull new image
	send("pulling", "Pulling "+newImage+"...", 0)
	reader, err := h.exec.PullImageWithAuth(stream.Context(), newImage, registryAuth)
	if err != nil {
		send("failed", "Failed to pull image: "+err.Error(), 0)
		return nil
	}

	// Stream pull progress
	dec := json.NewDecoder(reader)
	for {
		var evt struct {
			Status   string `json:"status"`
			ID       string `json:"id"`
			Error    string `json:"error"`
			Progress string `json:"progress"`
		}
		if decErr := dec.Decode(&evt); decErr != nil {
			if decErr == io.EOF {
				break
			}
			break
		}
		if evt.Error != "" {
			reader.Close()
			send("failed", "Pull error: "+evt.Error, 0)
			return nil
		}
		msg := evt.Status
		if evt.ID != "" {
			msg = evt.ID + ": " + evt.Status
		}
		send("pulling", msg, 0.3)
	}
	reader.Close()

	// Step 3: Replace container
	send("replacing", "Stopping old container and creating new one...", 0.5)
	newID, err := h.exec.ReplaceContainer(stream.Context(), containerID, newImage, snap)
	if err != nil {
		h.logger.Warn("replace failed, rolling back", "error", err)
		send("rolledback", "Replace failed, rolling back: "+err.Error(), 0)

		rollbackID, rbErr := h.exec.RollbackContainer(stream.Context(), newID, snap)
		if rbErr != nil {
			send("failed", "Rollback also failed: "+rbErr.Error(), 0)
			return nil
		}
		send("rolledback", "Rolled back to previous version ("+rollbackID+")", 0)
		return nil
	}

	// Step 4: Verify health
	send("verifying", "Verifying container health...", 0.7)
	if err := h.exec.WaitHealthy(stream.Context(), newID, 60*time.Second); err != nil {
		h.logger.Warn("health check failed, rolling back", "error", err)
		send("rolledback", "Health check failed: "+err.Error()+", rolling back...", 0)

		rollbackID, rbErr := h.exec.RollbackContainer(stream.Context(), newID, snap)
		if rbErr != nil {
			send("failed", "Rollback failed: "+rbErr.Error(), 0)
			return nil
		}
		send("rolledback", "Rolled back to previous version ("+rollbackID+")", 0)
		return nil
	}

	// Step 5: Success
	send("done", "Container updated successfully (new ID: "+newID+")", 1.0)
	h.logger.Info("container update completed", "old_id", containerID, "new_id", newID, "image", newImage)

	return nil
}
