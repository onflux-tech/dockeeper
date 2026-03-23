package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/jkdesenvolvedor/dockeeper/internal/manager/store"
	"github.com/jkdesenvolvedor/dockeeper/internal/shared/constants"
	agentpb "github.com/jkdesenvolvedor/dockeeper/proto/gen/dockeeper/agent/v1"
)

// Sentinel errors for container operations.
var (
	ErrContainerNotFound = errors.New("container not found")
	ErrServerNotFound    = errors.New("server not found")
	ErrServerOffline     = errors.New("server is offline")
)

const (
	agentCommandPort = "50052"
	commandTimeout   = 30 * time.Second
)

// CommandDispatcher sends a command to an agent and returns the result.
type CommandDispatcher interface {
	Dispatch(ctx context.Context, agentAddr string, req *agentpb.ExecuteCommandRequest) (*agentpb.ExecuteCommandResponse, error)
}

// GRPCCommandDispatcher connects to agents via gRPC to execute commands.
type GRPCCommandDispatcher struct{}

// NewGRPCCommandDispatcher creates a GRPCCommandDispatcher.
func NewGRPCCommandDispatcher() *GRPCCommandDispatcher {
	return &GRPCCommandDispatcher{}
}

// Dispatch dials the agent's command gRPC server and executes the command.
func (d *GRPCCommandDispatcher) Dispatch(ctx context.Context, agentAddr string, req *agentpb.ExecuteCommandRequest) (*agentpb.ExecuteCommandResponse, error) {
	addr := agentAddr + ":" + agentCommandPort
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial agent %s: %w", addr, err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			slog.Warn("failed to close agent connection", "addr", addr, "error", closeErr)
		}
	}()

	client := agentpb.NewAgentCommandsServiceClient(conn)

	cmdCtx, cmdCancel := context.WithTimeout(ctx, commandTimeout)
	defer cmdCancel()

	return client.ExecuteCommand(cmdCtx, req)
}

// ContainerService handles container action requests by resolving the
// container's server, dispatching the command via gRPC, and logging the action.
type ContainerService struct {
	app        core.App
	dispatcher CommandDispatcher
	audit      *store.AuditStore
}

// NewContainerService creates a ContainerService.
func NewContainerService(app core.App, dispatcher CommandDispatcher, audit *store.AuditStore) *ContainerService {
	return &ContainerService{
		app:        app,
		dispatcher: dispatcher,
		audit:      audit,
	}
}

// ActionResult is the outcome of a container action.
type ActionResult struct {
	Success bool   `json:"success"`
	Output  string `json:"output,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ExecuteAction resolves the container's server, checks connectivity,
// dispatches the command, and creates an audit log entry.
func (s *ContainerService) ExecuteAction(
	ctx context.Context,
	containerID string,
	cmdType agentpb.CommandType,
	params map[string]string,
	userID, ip string,
) (*ActionResult, error) {
	// 1. Find container record.
	container, err := s.app.FindRecordById(constants.CollContainers, containerID)
	if err != nil {
		return nil, ErrContainerNotFound
	}

	// 2. Resolve server.
	serverID := container.GetString("server")
	server, err := s.app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		return nil, ErrServerNotFound
	}

	// 3. Check server is online.
	if server.GetString("status") != "online" {
		return nil, ErrServerOffline
	}

	// 4. Build gRPC request.
	agentAddr := server.GetString("ip_address")
	dockerContainerID := container.GetString("container_id")

	req := &agentpb.ExecuteCommandRequest{
		ServerId:    serverID,
		ContainerId: dockerContainerID,
		Type:        cmdType,
		Params:      params,
	}

	// 5. Dispatch to agent.
	resp, err := s.dispatcher.Dispatch(ctx, agentAddr, req)

	// 6. Create audit log.
	details := map[string]any{
		"command":       cmdType.String(),
		"container_id":  dockerContainerID,
		"container_name": container.GetString("name"),
		"server_id":    serverID,
		"params":       params,
	}

	if err != nil {
		details["error"] = err.Error()
		s.audit.Log("container.action.failed", userID, "container", containerID, ip, details)
		return nil, fmt.Errorf("dispatch command: %w", err)
	}

	details["success"] = resp.GetSuccess()
	if resp.GetError() != "" {
		details["agent_error"] = resp.GetError()
	}

	s.audit.Log("container.action", userID, "container", containerID, ip, details)

	return &ActionResult{
		Success: resp.GetSuccess(),
		Output:  resp.GetOutput(),
		Error:   resp.GetError(),
	}, nil
}

// InspectContainer returns a container record by ID.
func (s *ContainerService) InspectContainer(containerID string) (*core.Record, error) {
	record, err := s.app.FindRecordById(constants.CollContainers, containerID)
	if err != nil {
		return nil, ErrContainerNotFound
	}
	return record, nil
}
