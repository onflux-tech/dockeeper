package grpc

import (
	"io"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/onflux-tech/dockeeper/internal/manager/channel"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// DataStreamHandler implements the AgentDataStreamService.
type DataStreamHandler struct {
	agentpb.UnimplementedAgentDataStreamServiceServer
	registry *channel.Registry
}

// NewDataStreamHandler creates the handler.
func NewDataStreamHandler(registry *channel.Registry) *DataStreamHandler {
	return &DataStreamHandler{registry: registry}
}

// OpenDataStream handles an on-demand data stream opened by the agent
// in response to an OpenDataStreamSignal from the command channel.
func (h *DataStreamHandler) OpenDataStream(stream agentpb.AgentDataStreamService_OpenDataStreamServer) error {
	serverID, ok := ServerIDFromContext(stream.Context())
	if !ok {
		return status.Error(codes.Unauthenticated, "missing server_id from mTLS")
	}

	// First message must carry the stream_id for matching.
	msg, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to receive first data message: %v", err)
	}

	streamID := msg.StreamId
	if streamID == "" {
		return status.Error(codes.InvalidArgument, "first message must include stream_id")
	}

	// Find the agent channel and deliver this stream to the waiter.
	ch, chOk := h.registry.Get(serverID)
	if !chOk {
		return status.Error(codes.NotFound, "no active channel for server")
	}

	if !ch.DeliverDataStream(streamID, stream) {
		return status.Error(codes.NotFound, "no waiter for stream_id")
	}

	slog.Info("data stream connected", "server_id", serverID, "stream_id", streamID)

	// Block until the stream ends — the consumer drives Send/Recv from here.
	// We wait for the stream context to be done (closed by consumer or agent).
	<-stream.Context().Done()

	// Drain any remaining messages to avoid gRPC resource leaks.
	for {
		if _, drainErr := stream.Recv(); drainErr != nil {
			break
		}
	}

	slog.Info("data stream closed", "server_id", serverID, "stream_id", streamID)
	return nil
}

// ConsumeDataStream is a helper that reads all data from a DataStream and calls
// the handler for each message. Used by manager services to process streaming data.
func ConsumeDataStream(
	stream agentpb.AgentDataStreamService_OpenDataStreamServer,
	onMessage func(*agentpb.OpenDataStreamRequest) error,
) error {
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := onMessage(msg); err != nil {
			return err
		}
	}
}
