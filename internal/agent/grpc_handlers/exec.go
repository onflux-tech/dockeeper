package grpc_handlers

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	dockerclient "github.com/docker/docker/client"

	"github.com/onflux-tech/dockeeper/internal/agent/executor"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

const execTimeout = 1 * time.Hour

// ExecHandler implements AgentExecServiceServer on the agent side.
type ExecHandler struct {
	agentpb.UnimplementedAgentExecServiceServer
	cli    *dockerclient.Client
	logger *slog.Logger
}

// NewExecHandler creates a new exec handler with its own Docker client.
func NewExecHandler(logger *slog.Logger) (*ExecHandler, error) {
	cli, err := dockerclient.NewClientWithOpts(
		dockerclient.FromEnv,
		dockerclient.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("create docker client for exec: %w", err)
	}
	return &ExecHandler{cli: cli, logger: logger}, nil
}

// Exec handles a bidirectional streaming exec session.
func (h *ExecHandler) Exec(stream agentpb.AgentExecService_ExecServer) error {
	// 1. Receive the first message — must be ExecStart.
	firstMsg, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("recv start message: %w", err)
	}

	start := firstMsg.GetStart()
	if start == nil {
		return fmt.Errorf("first message must be ExecStart")
	}

	containerID := start.GetContainerId()
	cmd := start.GetCommand()
	tty := start.GetTty()

	h.logger.Info("exec session starting",
		"container_id", containerID,
		"command", cmd,
		"tty", tty,
	)

	// 2. Create context with timeout.
	ctx, cancel := context.WithTimeout(stream.Context(), execTimeout)
	defer cancel()

	// 3. Create Docker exec session.
	session, err := executor.NewExecSession(ctx, h.cli, containerID, cmd, tty)
	if err != nil {
		return fmt.Errorf("create exec session: %w", err)
	}
	defer func() {
		if closeErr := session.Close(); closeErr != nil {
			h.logger.Warn("failed to close exec session", "error", closeErr)
		}
	}()

	h.logger.Info("exec session attached", "container_id", containerID)

	// 4. Bidirectional I/O.
	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	// Goroutine: read Docker output → send to gRPC stream.
	wg.Add(1)
	go func() {
		defer wg.Done()
		if readErr := h.forwardOutput(session, stream); readErr != nil {
			errCh <- readErr
		}
	}()

	// Goroutine: read gRPC stream → write to Docker stdin / handle resize.
	wg.Add(1)
	go func() {
		defer wg.Done()
		if inputErr := h.forwardInput(ctx, session, stream); inputErr != nil {
			errCh <- inputErr
		}
	}()

	// Wait for both goroutines to finish.
	wg.Wait()
	close(errCh)

	// 5. Get exit code.
	exitCode, inspectErr := session.ExitCode(ctx)
	if inspectErr != nil {
		h.logger.Warn("failed to get exec exit code", "error", inspectErr)
		exitCode = -1
	}

	h.logger.Info("exec session ended",
		"container_id", containerID,
		"exit_code", exitCode,
	)

	// 6. Send final response with exit code.
	exitCodeVal := int32(exitCode)
	_ = stream.Send(&agentpb.ExecResponse{
		ExitCode: &exitCodeVal,
	})

	return nil
}

// forwardOutput reads from the Docker exec session and sends to the gRPC stream.
func (h *ExecHandler) forwardOutput(session *executor.ExecSession, stream agentpb.AgentExecService_ExecServer) error {
	if session.IsTTY() {
		return h.forwardTTYOutput(session.Reader(), stream)
	}
	return h.forwardMuxOutput(session.Reader(), stream)
}

// forwardTTYOutput reads raw bytes from a TTY session.
func (h *ExecHandler) forwardTTYOutput(reader io.Reader, stream agentpb.AgentExecService_ExecServer) error {
	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			if sendErr := stream.Send(&agentpb.ExecResponse{
				Stdout: buf[:n],
			}); sendErr != nil {
				return sendErr
			}
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

// forwardMuxOutput reads Docker multiplexed frames (non-TTY).
func (h *ExecHandler) forwardMuxOutput(reader io.Reader, stream agentpb.AgentExecService_ExecServer) error {
	header := make([]byte, 8)
	for {
		if _, err := io.ReadFull(reader, header); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		isStderr := header[0] == 2
		size := binary.BigEndian.Uint32(header[4:8])
		if size == 0 {
			continue
		}

		payload := make([]byte, size)
		if _, err := io.ReadFull(reader, payload); err != nil {
			return err
		}

		resp := &agentpb.ExecResponse{}
		if isStderr {
			resp.Stderr = payload
		} else {
			resp.Stdout = payload
		}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
}

// forwardInput reads from the gRPC stream and writes stdin/resize to Docker.
func (h *ExecHandler) forwardInput(ctx context.Context, session *executor.ExecSession, stream agentpb.AgentExecService_ExecServer) error {
	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		switch {
		case msg.GetStdinData() != nil:
			if _, writeErr := session.Write(msg.GetStdinData()); writeErr != nil {
				return writeErr
			}

		case msg.GetResize() != nil:
			resize := msg.GetResize()
			if resizeErr := session.Resize(ctx, resize.GetRows(), resize.GetCols()); resizeErr != nil {
				h.logger.Warn("exec resize failed", "error", resizeErr)
			}
		}
	}
}
