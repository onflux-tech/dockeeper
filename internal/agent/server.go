package agent

import (
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc"

	"github.com/onflux-tech/dockeeper/internal/agent/collector"
	"github.com/onflux-tech/dockeeper/internal/agent/executor"
	grpchandlers "github.com/onflux-tech/dockeeper/internal/agent/grpc_handlers"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

const agentGRPCPort = ":50052"

// AgentServer wraps the gRPC server that the agent exposes so the manager
// can send commands and request log streams.
type AgentServer struct {
	gs   *grpc.Server
	addr string
}

// NewAgentServer creates a gRPC server on the agent side.
func NewAgentServer(exec executor.Executor, orch collector.OrchestratorClient, logger *slog.Logger) *AgentServer {
	gs := grpc.NewServer()

	cmds := grpchandlers.NewCommandsHandler(exec, logger)
	agentpb.RegisterAgentCommandsServiceServer(gs, cmds)

	logs := grpchandlers.NewLogsHandler(orch, logger)
	agentpb.RegisterAgentLogsServiceServer(gs, logs)

	imgs := grpchandlers.NewImageHandler(exec, logger)
	agentpb.RegisterAgentImageServiceServer(gs, imgs)

	compose := grpchandlers.NewComposeHandler(logger)
	agentpb.RegisterAgentComposeServiceServer(gs, compose)

	nets := grpchandlers.NewNetworkHandler(exec, logger)
	agentpb.RegisterAgentNetworkServiceServer(gs, nets)

	vols := grpchandlers.NewVolumeHandler(exec, logger)
	agentpb.RegisterAgentVolumeServiceServer(gs, vols)

	backups := grpchandlers.NewBackupHandler(exec, logger)
	agentpb.RegisterAgentBackupServiceServer(gs, backups)

	execH, err := grpchandlers.NewExecHandler(logger)
	if err != nil {
		logger.Warn("exec handler unavailable, terminal disabled", "error", err)
	} else {
		agentpb.RegisterAgentExecServiceServer(gs, execH)
	}

	return &AgentServer{gs: gs, addr: agentGRPCPort}
}

// Start begins listening and serving. This method blocks.
func (s *AgentServer) Start() error {
	lis, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("agent gRPC listen %s: %w", s.addr, err)
	}
	slog.Info("agent gRPC server listening", "addr", s.addr)
	return s.gs.Serve(lis)
}

// Stop gracefully stops the agent gRPC server.
func (s *AgentServer) Stop() {
	slog.Info("stopping agent gRPC server")
	s.gs.GracefulStop()
}
