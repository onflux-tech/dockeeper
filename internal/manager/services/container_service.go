package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/onflux-tech/dockeeper/internal/manager/store"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// Sentinel errors for container operations.
var (
	ErrContainerNotFound = errors.New("container not found")
	ErrServerNotFound    = errors.New("server not found")
	ErrServerOffline     = errors.New("server is offline")
)

const (
	agentCommandPort     = "50052"
	commandTimeout       = 30 * time.Second
	updateCommandTimeout = 150 * time.Second // pull + recreate needs more time
	internalRestartDelay = 750 * time.Millisecond
)

// CommandDispatcher sends a command to an agent and returns the result.
type CommandDispatcher interface {
	Dispatch(ctx context.Context, agentAddr string, req *agentpb.ExecuteCommandRequest) (*agentpb.ExecuteCommandResponse, error)
}

func timeoutForCommand(cmdType agentpb.CommandType) time.Duration {
	if cmdType == agentpb.CommandType_COMMAND_TYPE_UPDATE_IMAGE {
		return updateCommandTimeout
	}
	return commandTimeout
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

	cmdCtx, cmdCancel := context.WithTimeout(ctx, timeoutForCommand(req.GetType()))
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
	if cmdType == agentpb.CommandType_COMMAND_TYPE_RESTART && isDockeeperRuntimeContainer(container) {
		s.applyContainerActionState(container, cmdType)
		s.scheduleInternalRestart(agentAddr, req, container, userID, ip)

		details := map[string]any{
			"command":        cmdType.String(),
			"container_id":   dockerContainerID,
			"container_name": container.GetString("name"),
			"server_id":      serverID,
			"params":         params,
			"scheduled":      true,
		}
		s.logAudit("container.action", userID, "container", containerID, ip, details)

		return &ActionResult{
			Success: true,
			Output:  fmt.Sprintf("container %s restart scheduled", dockerContainerID),
		}, nil
	}

	resp, err := s.dispatcher.Dispatch(ctx, agentAddr, req)

	// 6. Create audit log.
	details := map[string]any{
		"command":        cmdType.String(),
		"container_id":   dockerContainerID,
		"container_name": container.GetString("name"),
		"server_id":      serverID,
		"params":         params,
	}

	if err != nil {
		details["error"] = err.Error()
		s.logAudit("container.action.failed", userID, "container", containerID, ip, details)
		return nil, fmt.Errorf("dispatch command: %w", err)
	}

	if resp.GetSuccess() {
		s.applyContainerActionState(container, cmdType)
	}

	details["success"] = resp.GetSuccess()
	if resp.GetError() != "" {
		details["agent_error"] = resp.GetError()
	}

	s.logAudit("container.action", userID, "container", containerID, ip, details)

	return &ActionResult{
		Success: resp.GetSuccess(),
		Output:  resp.GetOutput(),
		Error:   resp.GetError(),
	}, nil
}

func isDockeeperRuntimeContainer(container *core.Record) bool {
	name := strings.ToLower(container.GetString("name"))
	image := strings.ToLower(container.GetString("image"))

	return name == "dockeeper-agent" ||
		name == "dockeeper-manager" ||
		strings.Contains(image, "dockeeper-agent") ||
		strings.Contains(image, "dockeeper-manager")
}

func (s *ContainerService) scheduleInternalRestart(
	agentAddr string,
	req *agentpb.ExecuteCommandRequest,
	container *core.Record,
	userID, ip string,
) {
	containerID := container.Id
	containerName := container.GetString("name")
	dockerContainerID := container.GetString("container_id")
	serverID := container.GetString("server")
	paramsCopy := make(map[string]string, len(req.GetParams()))
	for key, value := range req.GetParams() {
		paramsCopy[key] = value
	}

	go func() {
		time.Sleep(internalRestartDelay)

		bgCtx, cancel := context.WithTimeout(context.Background(), timeoutForCommand(req.GetType()))
		defer cancel()

		bgReq := &agentpb.ExecuteCommandRequest{
			ServerId:    req.GetServerId(),
			ContainerId: req.GetContainerId(),
			Type:        req.GetType(),
			Params:      paramsCopy,
		}

		resp, err := s.dispatcher.Dispatch(bgCtx, agentAddr, bgReq)
		if err != nil {
			slog.Warn("scheduled internal restart failed",
				"container_id", dockerContainerID,
				"container_name", containerName,
				"server_id", serverID,
				"error", err,
			)
			s.logAudit("container.action.failed", userID, "container", containerID, ip, map[string]any{
				"command":        bgReq.GetType().String(),
				"container_id":   dockerContainerID,
				"container_name": containerName,
				"server_id":      serverID,
				"params":         paramsCopy,
				"scheduled":      true,
				"error":          err.Error(),
			})
			return
		}

		if !resp.GetSuccess() {
			slog.Warn("scheduled internal restart reported failure",
				"container_id", dockerContainerID,
				"container_name", containerName,
				"server_id", serverID,
				"error", resp.GetError(),
			)
			s.logAudit("container.action.failed", userID, "container", containerID, ip, map[string]any{
				"command":        bgReq.GetType().String(),
				"container_id":   dockerContainerID,
				"container_name": containerName,
				"server_id":      serverID,
				"params":         paramsCopy,
				"scheduled":      true,
				"agent_error":    resp.GetError(),
			})
		}
	}()
}

func (s *ContainerService) logAudit(action, userID, targetType, targetID, ip string, details map[string]any) {
	if s.audit == nil {
		return
	}
	s.audit.Log(action, userID, targetType, targetID, ip, details)
}

func (s *ContainerService) applyContainerActionState(container *core.Record, cmdType agentpb.CommandType) {
	if s.app == nil || container == nil {
		return
	}

	status, touchStartedAt, deleteRecord, ok := containerStateForAction(cmdType)
	if !ok {
		return
	}

	if deleteRecord {
		_ = s.app.Delete(container)
		return
	}

	container.Set("status", status)
	if touchStartedAt {
		container.Set("started_at", time.Now().UTC())
	}
	_ = s.app.Save(container)
}

func containerStateForAction(cmdType agentpb.CommandType) (status string, touchStartedAt bool, deleteRecord bool, ok bool) {
	switch cmdType {
	case agentpb.CommandType_COMMAND_TYPE_START,
		agentpb.CommandType_COMMAND_TYPE_UNPAUSE,
		agentpb.CommandType_COMMAND_TYPE_UPDATE_IMAGE,
		agentpb.CommandType_COMMAND_TYPE_RESTART:
		return "running", true, false, true
	case agentpb.CommandType_COMMAND_TYPE_STOP:
		return "exited", false, false, true
	case agentpb.CommandType_COMMAND_TYPE_PAUSE:
		return "paused", false, false, true
	case agentpb.CommandType_COMMAND_TYPE_REMOVE:
		return "", false, true, true
	default:
		return "", false, false, false
	}
}

// InspectContainer returns a container record by ID.
func (s *ContainerService) InspectContainer(containerID string) (*core.Record, error) {
	record, err := s.app.FindRecordById(constants.CollContainers, containerID)
	if err != nil {
		return nil, ErrContainerNotFound
	}
	return record, nil
}
