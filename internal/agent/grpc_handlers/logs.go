package grpc_handlers

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/jkdesenvolvedor/dockeeper/internal/agent/collector"
	agentpb "github.com/jkdesenvolvedor/dockeeper/proto/gen/dockeeper/agent/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// LogsHandler implements AgentLogsServiceServer on the agent side.
type LogsHandler struct {
	agentpb.UnimplementedAgentLogsServiceServer
	orch   collector.OrchestratorClient
	logger *slog.Logger
}

// NewLogsHandler creates a new logs handler.
func NewLogsHandler(orch collector.OrchestratorClient, logger *slog.Logger) *LogsHandler {
	return &LogsHandler{orch: orch, logger: logger}
}

// StreamLogs streams container logs back to the caller.
func (h *LogsHandler) StreamLogs(req *agentpb.StreamLogsRequest, stream agentpb.AgentLogsService_StreamLogsServer) error {
	containerID := req.GetContainerId()
	tail := "100"
	if req.GetTailLines() > 0 {
		tail = fmt.Sprintf("%d", req.GetTailLines())
	}

	h.logger.Info("streaming logs", "container_id", containerID, "follow", req.GetFollow(), "tail", tail)

	reader, tty, err := h.orch.ContainerLogs(
		stream.Context(),
		containerID,
		req.GetFollow(),
		tail,
		req.GetSince(),
	)
	if err != nil {
		return fmt.Errorf("open container logs: %w", err)
	}
	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			h.logger.Warn("failed to close log reader", "error", closeErr)
		}
	}()

	if tty {
		return h.streamTTYLogs(reader, stream)
	}
	return h.streamMuxLogs(reader, stream)
}

// streamTTYLogs reads line-by-line from a TTY log stream (no multiplexed header).
func (h *LogsHandler) streamTTYLogs(reader io.Reader, stream agentpb.AgentLogsService_StreamLogsServer) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		ts, msg := parseTimestampedLine(line)

		if err := stream.Send(&agentpb.StreamLogsResponse{
			Timestamp: ts,
			Stream:    "stdout",
			Message:   msg,
		}); err != nil {
			return err
		}
	}
	return scanner.Err()
}

// streamMuxLogs reads Docker multiplexed log frames (8-byte header + payload).
func (h *LogsHandler) streamMuxLogs(reader io.Reader, stream agentpb.AgentLogsService_StreamLogsServer) error {
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

		// Payload may contain multiple lines.
		lines := strings.Split(strings.TrimRight(string(payload), "\n"), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			ts, msg := parseTimestampedLine(line)

			if err := stream.Send(&agentpb.StreamLogsResponse{
				Timestamp: ts,
				Stream:    streamType,
				Message:   msg,
			}); err != nil {
				return err
			}
		}
	}
}

// parseTimestampedLine splits a Docker log line with timestamp prefix.
// Docker format: "2024-01-01T00:00:00.000000000Z message content"
func parseTimestampedLine(line string) (*timestamppb.Timestamp, string) {
	// Docker timestamps are at least 30 chars: "2006-01-02T15:04:05.000000000Z"
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
