package grpc_handlers

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"

	"github.com/jkdesenvolvedor/dockeeper/internal/agent/executor"
	agentpb "github.com/jkdesenvolvedor/dockeeper/proto/gen/dockeeper/agent/v1"
)

// ImageHandler implements AgentImageServiceServer on the agent side.
type ImageHandler struct {
	agentpb.UnimplementedAgentImageServiceServer
	exec   executor.Executor
	logger *slog.Logger
}

// NewImageHandler creates an image handler with the given executor.
func NewImageHandler(exec executor.Executor, logger *slog.Logger) *ImageHandler {
	return &ImageHandler{exec: exec, logger: logger}
}

// dockerPullEvent mirrors the JSON structure Docker emits during image pull.
type dockerPullEvent struct {
	ID             string                 `json:"id"`
	Status         string                 `json:"status"`
	ProgressDetail map[string]json.Number `json:"progressDetail"`
	Error          string                 `json:"error"`
}

// PullImage streams image pull progress back to the caller.
func (h *ImageHandler) PullImage(req *agentpb.ImagePullRequest, stream agentpb.AgentImageService_PullImageServer) error {
	imageRef := req.GetImageRef()
	h.logger.Info("pulling image with progress", "image", imageRef)

	reader, err := h.exec.PullImageStream(stream.Context(), imageRef)
	if err != nil {
		return stream.Send(&agentpb.ImagePullProgress{
			Done:  true,
			Error: err.Error(),
		})
	}
	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			h.logger.Warn("failed to close image pull reader", "error", closeErr)
		}
	}()

	decoder := json.NewDecoder(reader)
	for {
		var event dockerPullEvent
		if decErr := decoder.Decode(&event); decErr != nil {
			if decErr == io.EOF {
				break
			}
			return stream.Send(&agentpb.ImagePullProgress{
				Done:  true,
				Error: decErr.Error(),
			})
		}

		if event.Error != "" {
			return stream.Send(&agentpb.ImagePullProgress{
				Done:  true,
				Error: event.Error,
			})
		}

		var current, total int64
		if c, ok := event.ProgressDetail["current"]; ok {
			current, _ = c.Int64()
		}
		if t, ok := event.ProgressDetail["total"]; ok {
			total, _ = t.Int64()
		}

		if sendErr := stream.Send(&agentpb.ImagePullProgress{
			Id:      event.ID,
			Status:  event.Status,
			Current: current,
			Total:   total,
		}); sendErr != nil {
			return sendErr
		}
	}

	h.logger.Info("image pull completed", "image", imageRef)
	return stream.Send(&agentpb.ImagePullProgress{
		Status: "Pull complete",
		Done:   true,
	})
}

// RemoveImage removes a Docker image.
func (h *ImageHandler) RemoveImage(ctx context.Context, req *agentpb.ImageRemoveRequest) (*agentpb.ImageRemoveResponse, error) {
	imageID := req.GetImageId()
	h.logger.Info("removing image", "image_id", imageID)

	result := h.exec.RemoveImage(ctx, imageID, req.GetForce(), req.GetPruneChildren())

	if result.Error != "" {
		h.logger.Warn("image remove failed", "image_id", imageID, "error", result.Error)
	} else {
		h.logger.Info("image removed", "image_id", imageID, "deleted", len(result.Deleted))
	}

	return &agentpb.ImageRemoveResponse{
		Success:  result.Success,
		Error:    result.Error,
		Deleted:  result.Deleted,
		Untagged: result.Untagged,
	}, nil
}

// PruneImages removes unused Docker images.
func (h *ImageHandler) PruneImages(ctx context.Context, req *agentpb.ImagePruneRequest) (*agentpb.ImagePruneResponse, error) {
	h.logger.Info("pruning images", "all", req.GetAll())

	result := h.exec.PruneImages(ctx, req.GetAll())

	if result.Error != "" {
		h.logger.Warn("image prune failed", "error", result.Error)
	} else {
		h.logger.Info("images pruned", "deleted", len(result.Deleted), "reclaimed", result.SpaceReclaimed)
	}

	return &agentpb.ImagePruneResponse{
		Success:        result.Success,
		Error:          result.Error,
		Deleted:        result.Deleted,
		SpaceReclaimed: result.SpaceReclaimed,
	}, nil
}
