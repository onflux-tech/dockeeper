package tunnel

import (
	"context"
	"encoding/binary"
	"io"
	"sync"
	"time"

	dockerclient "github.com/docker/docker/client"

	"github.com/onflux-tech/dockeeper/internal/agent/executor"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

const execTimeout = 1 * time.Hour

// handleExecStreamSignal opens an ExecStream to the manager in response to an
// OpenExecStreamSignal received on the command channel.
func (c *Client) handleExecStreamSignal(ctx context.Context, signal *agentpb.OpenExecStreamSignal) {
	sessionID := signal.SessionId
	containerID := signal.ContainerId
	cmd := signal.Command
	tty := signal.Tty

	log := c.logger.With("session_id", sessionID, "container_id", containerID)
	log.Info("opening exec stream")

	stream, err := c.es.OpenExecStream(ctx)
	if err != nil {
		log.Error("failed to open exec stream", "error", err)
		return
	}

	// Send initial message with session_id.
	if err := stream.Send(&agentpb.OpenExecStreamRequest{
		SessionId: sessionID,
	}); err != nil {
		log.Error("failed to send exec stream init", "error", err)
		return
	}

	// Create Docker client for exec.
	cli, err := dockerclient.NewClientWithOpts(
		dockerclient.FromEnv,
		dockerclient.WithAPIVersionNegotiation(),
	)
	if err != nil {
		log.Error("failed to create docker client for exec", "error", err)
		c.sendExecExit(stream, sessionID, -1)
		return
	}
	defer func() { _ = cli.Close() }()

	// Create Docker exec session with timeout.
	execCtx, cancel := context.WithTimeout(ctx, execTimeout)
	defer cancel()

	session, err := executor.NewExecSession(execCtx, cli, containerID, cmd, tty)
	if err != nil {
		log.Error("failed to create exec session", "error", err)
		c.sendExecExit(stream, sessionID, -1)
		return
	}
	defer func() {
		if closeErr := session.Close(); closeErr != nil {
			log.Warn("failed to close exec session", "error", closeErr)
		}
	}()

	log.Info("exec session attached")

	// Bidirectional I/O.
	var wg sync.WaitGroup

	// Goroutine: read Docker output → send to ExecStream.
	wg.Add(1)
	go func() {
		defer wg.Done()
		if tty {
			c.forwardTTYExecOutput(session.Reader(), stream, sessionID)
		} else {
			c.forwardMuxExecOutput(session.Reader(), stream, sessionID)
		}
	}()

	// Goroutine: read ExecStream → write to Docker stdin / handle resize.
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.forwardExecInput(execCtx, session, stream)
	}()

	wg.Wait()

	// Get exit code.
	exitCode, inspectErr := session.ExitCode(execCtx)
	if inspectErr != nil {
		log.Warn("failed to get exec exit code", "error", inspectErr)
		exitCode = -1
	}

	log.Info("exec session ended", "exit_code", exitCode)
	c.sendExecExit(stream, sessionID, int32(exitCode))
	_ = stream.CloseSend()
}

// forwardTTYExecOutput reads raw bytes from a TTY exec session and sends them
// to the ExecStream as stdout.
func (c *Client) forwardTTYExecOutput(reader io.Reader, stream agentpb.AgentExecStreamService_OpenExecStreamClient, sessionID string) {
	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			if sendErr := stream.Send(&agentpb.OpenExecStreamRequest{
				SessionId: sessionID,
				Payload: &agentpb.OpenExecStreamRequest_Stdout{
					Stdout: buf[:n],
				},
			}); sendErr != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}

// forwardMuxExecOutput reads Docker multiplexed frames from a non-TTY exec
// session and sends them to the ExecStream.
func (c *Client) forwardMuxExecOutput(reader io.Reader, stream agentpb.AgentExecStreamService_OpenExecStreamClient, sessionID string) {
	header := make([]byte, 8)
	for {
		if _, err := io.ReadFull(reader, header); err != nil {
			return
		}

		size := binary.BigEndian.Uint32(header[4:8])
		if size == 0 {
			continue
		}

		payload := make([]byte, size)
		if _, err := io.ReadFull(reader, payload); err != nil {
			return
		}

		// For ExecStream, all output goes as stdout (the manager/WebSocket
		// bridge handles display). Stderr vs stdout distinction is not
		// critical here since both go to the terminal.
		if sendErr := stream.Send(&agentpb.OpenExecStreamRequest{
			SessionId: sessionID,
			Payload: &agentpb.OpenExecStreamRequest_Stdout{
				Stdout: payload,
			},
		}); sendErr != nil {
			return
		}
	}
}

// forwardExecInput reads messages from the ExecStream (stdin, resize, close)
// and writes them to the Docker exec session.
func (c *Client) forwardExecInput(ctx context.Context, session *executor.ExecSession, stream agentpb.AgentExecStreamService_OpenExecStreamClient) {
	for {
		msg, err := stream.Recv()
		if err != nil {
			return
		}

		switch payload := msg.Payload.(type) {
		case *agentpb.OpenExecStreamResponse_Stdin:
			if _, writeErr := session.Write(payload.Stdin); writeErr != nil {
				return
			}

		case *agentpb.OpenExecStreamResponse_Resize:
			if resizeErr := session.Resize(ctx, payload.Resize.Rows, payload.Resize.Cols); resizeErr != nil {
				c.logger.Warn("exec resize failed", "error", resizeErr)
			}

		case *agentpb.OpenExecStreamResponse_Close:
			return
		}
	}
}

// sendExecExit sends the exit code message on the exec stream.
func (c *Client) sendExecExit(stream agentpb.AgentExecStreamService_OpenExecStreamClient, sessionID string, exitCode int32) {
	_ = stream.Send(&agentpb.OpenExecStreamRequest{
		SessionId: sessionID,
		Payload: &agentpb.OpenExecStreamRequest_Exit{
			Exit: &agentpb.ExecStreamExit{ExitCode: exitCode},
		},
	})
}
