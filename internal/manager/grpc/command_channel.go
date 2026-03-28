package grpc

import (
	"io"
	"log/slog"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/onflux-tech/dockeeper/internal/manager/channel"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

const heartbeatInterval = 15 * time.Second

// CommandChannelHandler implements the AgentCommandChannelService.
type CommandChannelHandler struct {
	agentpb.UnimplementedAgentCommandChannelServiceServer
	registry *channel.Registry
}

// NewCommandChannelHandler creates the handler.
func NewCommandChannelHandler(registry *channel.Registry) *CommandChannelHandler {
	return &CommandChannelHandler{registry: registry}
}

// OpenChannel handles the persistent bidirectional command stream.
// The agent connects on startup and keeps the stream alive.
func (h *CommandChannelHandler) OpenChannel(stream agentpb.AgentCommandChannelService_OpenChannelServer) error {
	// The mTLS interceptor has already set server_id in context.
	serverID, ok := ServerIDFromContext(stream.Context())
	if !ok {
		return status.Error(codes.Unauthenticated, "missing server_id from mTLS")
	}

	// Wait for ChannelInit as the first message.
	initMsg, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to receive init: %v", err)
	}

	init := initMsg.GetInit()
	if init == nil {
		return status.Error(codes.InvalidArgument, "first message must be ChannelInit")
	}

	// Verify the init server_id matches the mTLS CN.
	if init.ServerId != serverID {
		slog.Warn("channel init server_id mismatch",
			"mtls_cn", serverID,
			"init_server_id", init.ServerId,
		)
		return status.Error(codes.PermissionDenied, "server_id mismatch")
	}

	// Create and register the agent channel.
	agentCh := channel.NewAgentChannel(serverID, stream)
	h.registry.Register(serverID, agentCh)
	defer h.registry.Unregister(serverID)

	slog.Info("command channel opened", "server_id", serverID)

	// Start heartbeat sender.
	go h.heartbeatLoop(agentCh, stream.Context().Done())

	// Receive loop: routes heartbeats and command results.
	for {
		msg, recvErr := stream.Recv()
		if recvErr != nil {
			if recvErr == io.EOF {
				slog.Info("command channel closed by agent", "server_id", serverID)
				return nil
			}
			slog.Warn("command channel recv error", "server_id", serverID, "error", recvErr)
			return recvErr
		}

		switch payload := msg.Payload.(type) {
		case *agentpb.OpenChannelRequest_Heartbeat:
			agentCh.RecordHeartbeat()

		case *agentpb.OpenChannelRequest_CommandResult:
			agentCh.DeliverCommandResult(payload.CommandResult)

		default:
			slog.Warn("unknown command channel message", "server_id", serverID)
		}
	}
}

// heartbeatLoop sends periodic heartbeats to the agent.
func (h *CommandChannelHandler) heartbeatLoop(ch *channel.AgentChannel, done <-chan struct{}) {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ch.Done():
			return
		case <-ticker.C:
			if err := ch.SendHeartbeat(); err != nil {
				slog.Warn("heartbeat send failed", "server_id", ch.ServerID, "error", err)
				return
			}
		}
	}
}
