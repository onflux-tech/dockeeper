package grpc_handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/onflux-tech/dockeeper/internal/agent/executor"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// UpdateHandler implements AgentUpdateServiceServer on the agent side.
type UpdateHandler struct {
	agentpb.UnimplementedAgentUpdateServiceServer
	exec    executor.Executor
	logger  *slog.Logger
	dataDir string
}

// NewUpdateHandler creates an update handler.
func NewUpdateHandler(exec executor.Executor, logger *slog.Logger, dataDir ...string) *UpdateHandler {
	dd := ""
	if len(dataDir) > 0 {
		dd = dataDir[0]
	}
	return &UpdateHandler{exec: exec, logger: logger, dataDir: dd}
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
			_ = reader.Close()
			send("failed", "Pull error: "+evt.Error, 0)
			return nil
		}
		msg := evt.Status
		if evt.ID != "" {
			msg = evt.ID + ": " + evt.Status
		}
		send("pulling", msg, 0.3)
	}
	_ = reader.Close()

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

// UpdateAgentBinary receives a new agent binary via streaming, verifies checksum, and replaces.
func (h *UpdateHandler) UpdateAgentBinary(stream agentpb.AgentUpdateService_UpdateAgentBinaryServer) error {
	h.logger.Info("receiving agent binary update")

	tmpFile, err := os.CreateTemp("", "dockeeper-agent-update-*")
	if err != nil {
		return stream.SendAndClose(&agentpb.UpdateBinaryResponse{
			Success: false,
			Error:   "failed to create temp file: " + err.Error(),
		})
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	hasher := sha256.New()
	var expectedChecksum, targetVersion string
	var totalBytes int64

	for {
		chunk, recvErr := stream.Recv()
		if recvErr == io.EOF {
			break
		}
		if recvErr != nil {
			_ = tmpFile.Close()
			return recvErr
		}

		if chunk.GetVersion() != "" {
			targetVersion = chunk.GetVersion()
		}
		if chunk.GetChecksum() != "" {
			expectedChecksum = chunk.GetChecksum()
		}

		data := chunk.GetChunk()
		if len(data) > 0 {
			n, writeErr := tmpFile.Write(data)
			if writeErr != nil {
				_ = tmpFile.Close()
				return stream.SendAndClose(&agentpb.UpdateBinaryResponse{
					Success: false,
					Error:   "write error: " + writeErr.Error(),
				})
			}
			hasher.Write(data)
			totalBytes += int64(n)
		}

		if chunk.GetIsLastChunk() {
			break
		}
	}
	_ = tmpFile.Close()

	h.logger.Info("binary received", "bytes", totalBytes, "version", targetVersion)

	// Verify checksum
	actualChecksum := hex.EncodeToString(hasher.Sum(nil))
	if expectedChecksum != "" && actualChecksum != expectedChecksum {
		return stream.SendAndClose(&agentpb.UpdateBinaryResponse{
			Success: false,
			Error:   fmt.Sprintf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum),
		})
	}

	// Get current binary path
	currentBinary, err := os.Executable()
	if err != nil {
		return stream.SendAndClose(&agentpb.UpdateBinaryResponse{
			Success: false,
			Error:   "cannot determine current binary path: " + err.Error(),
		})
	}
	currentBinary, _ = filepath.EvalSymlinks(currentBinary)

	// Make temp file executable
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return stream.SendAndClose(&agentpb.UpdateBinaryResponse{
			Success: false,
			Error:   "chmod failed: " + err.Error(),
		})
	}

	// Atomic replace: current → .backup, temp → current
	backupPath := currentBinary + ".backup"
	_ = os.Remove(backupPath)
	if err := os.Rename(currentBinary, backupPath); err != nil {
		return stream.SendAndClose(&agentpb.UpdateBinaryResponse{
			Success: false,
			Error:   "failed to backup current binary: " + err.Error(),
		})
	}

	if err := copyFile(tmpPath, currentBinary); err != nil {
		// Restore backup
		_ = os.Rename(backupPath, currentBinary)
		return stream.SendAndClose(&agentpb.UpdateBinaryResponse{
			Success: false,
			Error:   "failed to install new binary: " + err.Error(),
		})
	}
	_ = os.Chmod(currentBinary, 0o755)

	h.logger.Info("agent binary replaced, restart required", "version", targetVersion)

	return stream.SendAndClose(&agentpb.UpdateBinaryResponse{
		Success:        true,
		CurrentVersion: targetVersion,
	})
}

// copyFile copies src to dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	_, err = io.Copy(out, in)
	return err
}
