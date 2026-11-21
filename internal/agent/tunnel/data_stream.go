package tunnel

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/onflux-tech/dockeeper/internal/agent/executor"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// handleDataStreamSignal opens a DataStream to the manager in response to an
// OpenDataStreamSignal received on the command channel.
func (c *Client) handleDataStreamSignal(ctx context.Context, signal *agentpb.OpenDataStreamSignal) {
	streamID := signal.StreamId
	log := c.logger.With("stream_id", streamID, "type", signal.StreamType.String())
	log.Info("opening data stream")

	stream, err := c.ds.OpenDataStream(ctx)
	if err != nil {
		log.Error("failed to open data stream", "error", err)
		return
	}

	// Send initial message with stream_id.
	if err := stream.Send(&agentpb.OpenDataStreamRequest{StreamId: streamID}); err != nil {
		log.Error("failed to send stream init", "error", err)
		return
	}

	// Start cancellation listener.
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start cancel listener only for stream types where the agent SENDS data.
	// For AGENT_UPDATE the agent RECEIVES data via Recv(), so a concurrent
	// listenForCancel (which also calls Recv) would race and corrupt the stream.
	if signal.StreamType != agentpb.DataStreamType_DATA_STREAM_TYPE_AGENT_UPDATE {
		go c.listenForCancel(cancelCtx, cancel, stream)
	}

	var streamErr error
	switch signal.StreamType {
	case agentpb.DataStreamType_DATA_STREAM_TYPE_LOGS:
		streamErr = c.streamLogs(cancelCtx, signal.GetStreamLogs(), stream, streamID)

	case agentpb.DataStreamType_DATA_STREAM_TYPE_IMAGE_PULL:
		streamErr = c.streamImagePull(cancelCtx, signal.GetPullImage(), stream, streamID)

	case agentpb.DataStreamType_DATA_STREAM_TYPE_COMPOSE_UP:
		streamErr = c.streamCompose(cancelCtx, "up", signal.GetComposeUp(), stream, streamID)

	case agentpb.DataStreamType_DATA_STREAM_TYPE_COMPOSE_DOWN:
		streamErr = c.streamCompose(cancelCtx, "down", signal.GetComposeDown(), stream, streamID)

	case agentpb.DataStreamType_DATA_STREAM_TYPE_COMPOSE_PULL:
		streamErr = c.streamCompose(cancelCtx, "pull", signal.GetComposePull(), stream, streamID)

	case agentpb.DataStreamType_DATA_STREAM_TYPE_BACKUP:
		streamErr = c.streamBackup(cancelCtx, signal.GetBackupVolume(), stream, streamID)

	case agentpb.DataStreamType_DATA_STREAM_TYPE_RESTORE:
		streamErr = c.streamRestore(cancelCtx, signal.GetRestoreVolume(), stream, streamID)

	case agentpb.DataStreamType_DATA_STREAM_TYPE_CONTAINER_UPDATE:
		streamErr = c.streamContainerUpdate(cancelCtx, signal.GetContainerUpdate(), stream, streamID)

	case agentpb.DataStreamType_DATA_STREAM_TYPE_AGENT_UPDATE:
		streamErr = c.streamAgentUpdate(cancelCtx, signal.GetUpdateBinary(), stream, streamID)

	default:
		streamErr = fmt.Errorf("unsupported data stream type: %s", signal.StreamType)
	}

	if streamErr != nil {
		log.Warn("data stream error", "error", streamErr)
		_ = stream.Send(&agentpb.OpenDataStreamRequest{
			StreamId: streamID,
			Payload: &agentpb.OpenDataStreamRequest_Error{
				Error: &agentpb.DataStreamError{
					Code:    500,
					Message: streamErr.Error(),
				},
			},
		})
	}

	_ = stream.CloseSend()
	log.Info("data stream closed")
}

// listenForCancel watches for manager-initiated cancellation on the data stream.
func (c *Client) listenForCancel(ctx context.Context, cancel context.CancelFunc, stream agentpb.AgentDataStreamService_OpenDataStreamClient) {
	for {
		msg, err := stream.Recv()
		if err != nil {
			return
		}
		if msg.GetCancel() != nil {
			c.logger.Info("data stream cancelled by manager", "stream_id", msg.StreamId)
			cancel()
			return
		}

		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

// --- Log streaming ---

func (c *Client) streamLogs(ctx context.Context, req *agentpb.StreamLogsRequest, stream agentpb.AgentDataStreamService_OpenDataStreamClient, streamID string) error {
	if c.orch == nil {
		return fmt.Errorf("no container runtime available")
	}

	tail := "100"
	if req.GetTailLines() > 0 {
		tail = fmt.Sprintf("%d", req.GetTailLines())
	}

	reader, tty, err := c.orch.ContainerLogs(ctx, req.GetContainerId(), req.GetFollow(), tail, req.GetSince())
	if err != nil {
		return fmt.Errorf("open container logs: %w", err)
	}
	defer func() { _ = reader.Close() }()

	if tty {
		return c.forwardTTYLogs(reader, stream, streamID)
	}
	return c.forwardMuxLogs(reader, stream, streamID)
}

func (c *Client) forwardTTYLogs(reader io.Reader, stream agentpb.AgentDataStreamService_OpenDataStreamClient, streamID string) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		ts, msg := parseTimestampedLine(line)

		if err := stream.Send(&agentpb.OpenDataStreamRequest{
			StreamId: streamID,
			Payload: &agentpb.OpenDataStreamRequest_Text{
				Text: &agentpb.DataStreamText{
					Line:      msg,
					Stream:    "stdout",
					Timestamp: ts,
				},
			},
		}); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (c *Client) forwardMuxLogs(reader io.Reader, stream agentpb.AgentDataStreamService_OpenDataStreamClient, streamID string) error {
	header := make([]byte, 8)
	for {
		if _, err := io.ReadFull(reader, header); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		streamType := "stdout"
		if header[0] == 2 {
			streamType = "stderr"
		}

		size := binary.BigEndian.Uint32(header[4:8])
		if size == 0 {
			continue
		}

		payload := make([]byte, size)
		if _, err := io.ReadFull(reader, payload); err != nil {
			return err
		}

		lines := strings.Split(strings.TrimRight(string(payload), "\n"), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			ts, msg := parseTimestampedLine(line)
			if err := stream.Send(&agentpb.OpenDataStreamRequest{
				StreamId: streamID,
				Payload: &agentpb.OpenDataStreamRequest_Text{
					Text: &agentpb.DataStreamText{
						Line:      msg,
						Stream:    streamType,
						Timestamp: ts,
					},
				},
			}); err != nil {
				return err
			}
		}
	}
}

// parseTimestampedLine splits a Docker log line with optional RFC3339Nano timestamp prefix.
func parseTimestampedLine(line string) (*timestamppb.Timestamp, string) {
	if len(line) > 30 && line[4] == '-' && line[10] == 'T' {
		spaceIdx := strings.IndexByte(line, ' ')
		if spaceIdx > 0 {
			t, err := time.Parse(time.RFC3339Nano, line[:spaceIdx])
			if err == nil {
				return timestamppb.New(t), line[spaceIdx+1:]
			}
		}
	}
	return timestamppb.Now(), line
}

// --- Image pull streaming ---

type dockerPullEvent struct {
	ID             string                 `json:"id"`
	Status         string                 `json:"status"`
	ProgressDetail map[string]json.Number `json:"progressDetail"`
	Error          string                 `json:"error"`
}

func (c *Client) streamImagePull(ctx context.Context, req *agentpb.ImagePullRequest, stream agentpb.AgentDataStreamService_OpenDataStreamClient, streamID string) error {
	reader, err := c.exec.PullImageStream(ctx, req.GetImageRef())
	if err != nil {
		return fmt.Errorf("pull image: %w", err)
	}
	defer func() { _ = reader.Close() }()

	decoder := json.NewDecoder(reader)
	for {
		var event dockerPullEvent
		if decErr := decoder.Decode(&event); decErr != nil {
			if decErr == io.EOF {
				break
			}
			return fmt.Errorf("decode pull event: %w", decErr)
		}

		if event.Error != "" {
			return fmt.Errorf("pull error: %s", event.Error)
		}

		var current, total int64
		if cv, ok := event.ProgressDetail["current"]; ok {
			current, _ = cv.Int64()
		}
		if tv, ok := event.ProgressDetail["total"]; ok {
			total, _ = tv.Int64()
		}

		if err := stream.Send(&agentpb.OpenDataStreamRequest{
			StreamId: streamID,
			Payload: &agentpb.OpenDataStreamRequest_Progress{
				Progress: &agentpb.DataStreamProgress{
					Id:      event.ID,
					Status:  event.Status,
					Message: event.Status,
					Current: current,
					Total:   total,
				},
			},
		}); err != nil {
			return err
		}
	}

	return stream.Send(&agentpb.OpenDataStreamRequest{
		StreamId: streamID,
		Payload: &agentpb.OpenDataStreamRequest_Complete{
			Complete: &agentpb.DataStreamComplete{Summary: "pull complete"},
		},
	})
}

// --- Compose streaming ---

func (c *Client) streamCompose(ctx context.Context, op string, req *agentpb.ComposeRequest, stream agentpb.AgentDataStreamService_OpenDataStreamClient, streamID string) error {
	runner := executor.NewComposeRunner()

	var outCh <-chan executor.ComposeOutputLine
	var doneCh <-chan executor.ComposeResult

	switch op {
	case "up":
		outCh, doneCh = runner.Up(ctx, req.GetProjectName(), req.GetWorkingDir(), req.GetFiles())
	case "down":
		outCh, doneCh = runner.Down(ctx, req.GetProjectName(), req.GetWorkingDir(), req.GetFiles())
	case "pull":
		outCh, doneCh = runner.Pull(ctx, req.GetProjectName(), req.GetWorkingDir(), req.GetFiles())
	default:
		return fmt.Errorf("unsupported compose operation: %s", op)
	}

	for line := range outCh {
		if err := stream.Send(&agentpb.OpenDataStreamRequest{
			StreamId: streamID,
			Payload: &agentpb.OpenDataStreamRequest_Text{
				Text: &agentpb.DataStreamText{
					Line:   line.Line,
					Stream: line.Stream,
				},
			},
		}); err != nil {
			return err
		}
	}

	result := <-doneCh

	return stream.Send(&agentpb.OpenDataStreamRequest{
		StreamId: streamID,
		Payload: &agentpb.OpenDataStreamRequest_Complete{
			Complete: &agentpb.DataStreamComplete{
				Summary:  fmt.Sprintf("compose %s finished", op),
				ExitCode: int32(result.ExitCode),
			},
		},
	})
}

// --- Backup/Restore streaming ---

func (c *Client) streamBackup(ctx context.Context, req *agentpb.BackupVolumeRequest, stream agentpb.AgentDataStreamService_OpenDataStreamClient, streamID string) error {
	s3Cfg := protoToS3Config(req.GetS3())

	sendProgress := func(p executor.BackupProgressInfo) {
		_ = stream.Send(&agentpb.OpenDataStreamRequest{
			StreamId: streamID,
			Payload: &agentpb.OpenDataStreamRequest_Progress{
				Progress: &agentpb.DataStreamProgress{
					Status:  p.Status,
					Message: p.Message,
					Current: p.BytesProcessed,
					Total:   p.TotalBytes,
				},
			},
		})
	}

	if err := c.exec.BackupVolume(ctx, req.GetMountpoint(), s3Cfg, sendProgress); err != nil {
		return err
	}

	return stream.Send(&agentpb.OpenDataStreamRequest{
		StreamId: streamID,
		Payload: &agentpb.OpenDataStreamRequest_Complete{
			Complete: &agentpb.DataStreamComplete{Summary: "backup complete"},
		},
	})
}

func (c *Client) streamRestore(ctx context.Context, req *agentpb.RestoreVolumeRequest, stream agentpb.AgentDataStreamService_OpenDataStreamClient, streamID string) error {
	s3Cfg := protoToS3Config(req.GetS3())

	sendProgress := func(p executor.BackupProgressInfo) {
		_ = stream.Send(&agentpb.OpenDataStreamRequest{
			StreamId: streamID,
			Payload: &agentpb.OpenDataStreamRequest_Progress{
				Progress: &agentpb.DataStreamProgress{
					Status:  p.Status,
					Message: p.Message,
					Current: p.BytesProcessed,
					Total:   p.TotalBytes,
				},
			},
		})
	}

	if err := c.exec.RestoreVolume(ctx, req.GetMountpoint(), s3Cfg, sendProgress); err != nil {
		return err
	}

	return stream.Send(&agentpb.OpenDataStreamRequest{
		StreamId: streamID,
		Payload: &agentpb.OpenDataStreamRequest_Complete{
			Complete: &agentpb.DataStreamComplete{Summary: "restore complete"},
		},
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

// --- Container update streaming ---

func (c *Client) streamContainerUpdate(ctx context.Context, req *agentpb.ExecuteContainerUpdateRequest, stream agentpb.AgentDataStreamService_OpenDataStreamClient, streamID string) error {
	containerID := req.GetContainerId()
	newImage := req.GetNewImage()
	registryAuth := req.GetRegistryAuth()

	send := func(step, message string, progress float32) {
		_ = stream.Send(&agentpb.OpenDataStreamRequest{
			StreamId: streamID,
			Payload: &agentpb.OpenDataStreamRequest_Text{
				Text: &agentpb.DataStreamText{
					Line:   fmt.Sprintf("[%s] %s (%.0f%%)", step, message, progress*100),
					Stream: "stdout",
				},
			},
		})
	}

	// Step 1: Snapshot
	send("snapshot", "Capturing container configuration...", 0)
	snap, err := c.exec.SnapshotContainer(ctx, containerID)
	if err != nil {
		send("failed", "Failed to snapshot container: "+err.Error(), 0)
		return nil
	}

	// Step 2: Pull new image
	send("pulling", "Pulling "+newImage+"...", 0)
	reader, err := c.exec.PullImageWithAuth(ctx, newImage, registryAuth)
	if err != nil {
		send("failed", "Failed to pull image: "+err.Error(), 0)
		return nil
	}

	dec := json.NewDecoder(reader)
	for {
		var evt struct {
			Status string `json:"status"`
			ID     string `json:"id"`
			Error  string `json:"error"`
		}
		if decErr := dec.Decode(&evt); decErr != nil {
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

	// Step 3: Replace
	send("replacing", "Stopping old container and creating new one...", 0.5)
	newID, err := c.exec.ReplaceContainer(ctx, containerID, newImage, snap)
	if err != nil {
		send("rolledback", "Replace failed, rolling back: "+err.Error(), 0)
		rollbackID, rbErr := c.exec.RollbackContainer(ctx, newID, snap)
		if rbErr != nil {
			send("failed", "Rollback also failed: "+rbErr.Error(), 0)
			return nil
		}
		send("rolledback", "Rolled back to previous version ("+rollbackID+")", 0)
		return nil
	}

	// Step 4: Health check
	send("verifying", "Verifying container health...", 0.7)
	if err := c.exec.WaitHealthy(ctx, newID, 60*time.Second); err != nil {
		send("rolledback", "Health check failed: "+err.Error()+", rolling back...", 0)
		rollbackID, rbErr := c.exec.RollbackContainer(ctx, newID, snap)
		if rbErr != nil {
			send("failed", "Rollback failed: "+rbErr.Error(), 0)
			return nil
		}
		send("rolledback", "Rolled back to previous version ("+rollbackID+")", 0)
		return nil
	}

	send("done", "Container updated successfully (new ID: "+newID+")", 1.0)

	return stream.Send(&agentpb.OpenDataStreamRequest{
		StreamId: streamID,
		Payload: &agentpb.OpenDataStreamRequest_Complete{
			Complete: &agentpb.DataStreamComplete{Summary: "container update complete"},
		},
	})
}

// --- Agent binary update ---

func (c *Client) streamAgentUpdate(ctx context.Context, params *agentpb.DataStreamUpdateBinaryParams, stream agentpb.AgentDataStreamService_OpenDataStreamClient, streamID string) error {
	c.logger.Info("receiving agent binary update via data stream",
		"version", params.GetVersion(), "checksum", params.GetChecksum(),
	)

	tmpFile, err := os.CreateTemp("", "dockeeper-agent-update-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	hasher := sha256.New()
	var totalBytes int64

	// Receive binary chunks from the manager.
	for {
		msg, recvErr := stream.Recv()
		if recvErr != nil {
			_ = tmpFile.Close()
			if recvErr == io.EOF {
				break
			}
			return fmt.Errorf("recv binary chunk: %w", recvErr)
		}

		if msg.GetCancel() != nil {
			_ = tmpFile.Close()
			return fmt.Errorf("update cancelled by manager")
		}

		data := msg.GetData()
		if len(data) == 0 {
			// Empty data message = EOF sentinel from manager.
			break
		}

		n, writeErr := tmpFile.Write(data)
		if writeErr != nil {
			_ = tmpFile.Close()
			return fmt.Errorf("write chunk: %w", writeErr)
		}
		hasher.Write(data)
		totalBytes += int64(n)
	}
	_ = tmpFile.Close()

	c.logger.Info("agent binary received", "bytes", totalBytes)

	// Verify checksum.
	expectedChecksum := params.GetChecksum()
	actualChecksum := hex.EncodeToString(hasher.Sum(nil))
	if expectedChecksum != "" && actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	// Get current binary path.
	currentBinary, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine current binary path: %w", err)
	}
	currentBinary, _ = filepath.EvalSymlinks(currentBinary)

	// Make temp file executable.
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}

	// Atomic replace: current → .backup, temp → current.
	backupPath := currentBinary + ".backup"
	_ = os.Remove(backupPath)
	if err := os.Rename(currentBinary, backupPath); err != nil {
		return fmt.Errorf("backup current binary: %w", err)
	}

	if err := copyFile(tmpPath, currentBinary); err != nil {
		_ = os.Rename(backupPath, currentBinary)
		return fmt.Errorf("install new binary: %w", err)
	}
	_ = os.Chmod(currentBinary, 0o755)

	c.logger.Info("agent binary replaced", "version", params.GetVersion())

	// Notify manager of success.
	_ = stream.Send(&agentpb.OpenDataStreamRequest{
		StreamId: streamID,
		Payload: &agentpb.OpenDataStreamRequest_Complete{
			Complete: &agentpb.DataStreamComplete{
				Summary: "agent binary updated to " + params.GetVersion(),
			},
		},
	})

	// Trigger graceful restart so the new binary takes effect.
	// Short delay allows the complete message to be sent first.
	if c.onShutdown != nil {
		go func() {
			time.Sleep(1 * time.Second)
			c.logger.Info("restarting agent after binary update")
			c.onShutdown()
		}()
	}

	return nil
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
