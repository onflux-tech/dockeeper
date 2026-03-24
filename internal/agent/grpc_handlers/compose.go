package grpc_handlers

import (
	"log/slog"

	"github.com/jkdesenvolvedor/dockeeper/internal/agent/executor"
	agentpb "github.com/jkdesenvolvedor/dockeeper/proto/gen/dockeeper/agent/v1"
)

// ComposeHandler implements AgentComposeServiceServer on the agent side.
type ComposeHandler struct {
	agentpb.UnimplementedAgentComposeServiceServer
	runner *executor.ComposeRunner
	logger *slog.Logger
}

// NewComposeHandler creates a compose handler.
func NewComposeHandler(logger *slog.Logger) *ComposeHandler {
	return &ComposeHandler{
		runner: executor.NewComposeRunner(),
		logger: logger,
	}
}

// composeSender abstracts the Send method shared by all compose stream types.
type composeSender interface {
	Send(*agentpb.ComposeOutput) error
}

// ComposeUp runs `docker compose up -d` and streams output.
func (h *ComposeHandler) ComposeUp(req *agentpb.ComposeRequest, stream agentpb.AgentComposeService_ComposeUpServer) error {
	h.logger.Info("compose up", "project", req.GetProjectName(), "dir", req.GetWorkingDir())
	outCh, doneCh := h.runner.Up(stream.Context(), req.GetProjectName(), req.GetWorkingDir(), req.GetFiles())
	return h.forward(stream, outCh, doneCh)
}

// ComposeDown runs `docker compose down` and streams output.
func (h *ComposeHandler) ComposeDown(req *agentpb.ComposeRequest, stream agentpb.AgentComposeService_ComposeDownServer) error {
	h.logger.Info("compose down", "project", req.GetProjectName(), "dir", req.GetWorkingDir())
	outCh, doneCh := h.runner.Down(stream.Context(), req.GetProjectName(), req.GetWorkingDir(), req.GetFiles())
	return h.forward(stream, outCh, doneCh)
}

// ComposePull runs `docker compose pull` and streams output.
func (h *ComposeHandler) ComposePull(req *agentpb.ComposeRequest, stream agentpb.AgentComposeService_ComposePullServer) error {
	h.logger.Info("compose pull", "project", req.GetProjectName(), "dir", req.GetWorkingDir())
	outCh, doneCh := h.runner.Pull(stream.Context(), req.GetProjectName(), req.GetWorkingDir(), req.GetFiles())
	return h.forward(stream, outCh, doneCh)
}

// forward bridges compose output channels to a gRPC server stream.
func (h *ComposeHandler) forward(stream composeSender, outCh <-chan executor.ComposeOutputLine, doneCh <-chan executor.ComposeResult) error {
	for line := range outCh {
		if sendErr := stream.Send(&agentpb.ComposeOutput{
			Line:   line.Line,
			Stream: line.Stream,
		}); sendErr != nil {
			return sendErr
		}
	}

	result := <-doneCh

	return stream.Send(&agentpb.ComposeOutput{
		Done:     true,
		ExitCode: int32(result.ExitCode),
		Error:    result.Error,
	})
}
