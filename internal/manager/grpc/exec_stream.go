package grpc

import (
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/onflux-tech/dockeeper/internal/manager/channel"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// ExecStreamHandler implements the AgentExecStreamService.
type ExecStreamHandler struct {
	agentpb.UnimplementedAgentExecStreamServiceServer
	registry *channel.Registry
}

// NewExecStreamHandler creates the handler.
func NewExecStreamHandler(registry *channel.Registry) *ExecStreamHandler {
	return &ExecStreamHandler{registry: registry}
}

// OpenExecStream handles a per-session exec stream opened by the agent
// in response to an OpenExecStreamSignal from the command channel.
func (h *ExecStreamHandler) OpenExecStream(stream agentpb.AgentExecStreamService_OpenExecStreamServer) error {
	serverID, ok := ServerIDFromContext(stream.Context())
	if !ok {
		return status.Error(codes.Unauthenticated, "missing server_id from mTLS")
	}

	// First message must carry the session_id for matching.
	msg, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to receive first exec message: %v", err)
	}

	sessionID := msg.SessionId
	if sessionID == "" {
		return status.Error(codes.InvalidArgument, "first message must include session_id")
	}

	// Find the agent channel and deliver this stream to the waiter.
	ch, chOk := h.registry.Get(serverID)
	if !chOk {
		return status.Error(codes.NotFound, "no active channel for server")
	}

	if !ch.DeliverExecStream(sessionID, stream) {
		return status.Error(codes.NotFound, "no waiter for session_id")
	}

	slog.Info("exec stream connected", "server_id", serverID, "session_id", sessionID)

	// Block until the stream ends — the WebSocket bridge drives Send/Recv.
	<-stream.Context().Done()

	slog.Info("exec stream closed", "server_id", serverID, "session_id", sessionID)
	return nil
}
