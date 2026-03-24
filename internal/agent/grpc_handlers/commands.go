package grpc_handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/onflux-tech/dockeeper/internal/agent/executor"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// CommandsHandler implements AgentCommandsServiceServer on the agent side.
// It receives commands from the manager and executes them via the Docker executor.
type CommandsHandler struct {
	agentpb.UnimplementedAgentCommandsServiceServer
	exec   executor.Executor
	logger *slog.Logger
}

// NewCommandsHandler creates a CommandsHandler with the given executor.
func NewCommandsHandler(exec executor.Executor, logger *slog.Logger) *CommandsHandler {
	return &CommandsHandler{
		exec:   exec,
		logger: logger,
	}
}

// ExecuteCommand dispatches a command to the appropriate executor method based
// on the CommandType.
func (h *CommandsHandler) ExecuteCommand(ctx context.Context, req *agentpb.ExecuteCommandRequest) (*agentpb.ExecuteCommandResponse, error) {
	containerID := req.GetContainerId()
	params := req.GetParams()

	h.logger.Info("executing command",
		"type", req.GetType().String(),
		"container_id", containerID,
	)

	var result executor.Result

	switch req.GetType() {
	case agentpb.CommandType_COMMAND_TYPE_START:
		result = h.exec.Start(ctx, containerID)

	case agentpb.CommandType_COMMAND_TYPE_STOP:
		timeout := parseIntParam(params, "timeout")
		result = h.exec.Stop(ctx, containerID, timeout)

	case agentpb.CommandType_COMMAND_TYPE_RESTART:
		timeout := parseIntParam(params, "timeout")
		result = h.exec.Restart(ctx, containerID, timeout)

	case agentpb.CommandType_COMMAND_TYPE_REMOVE:
		force := params["force"] == "true"
		result = h.exec.Remove(ctx, containerID, force)

	case agentpb.CommandType_COMMAND_TYPE_PAUSE:
		result = h.exec.Pause(ctx, containerID)

	case agentpb.CommandType_COMMAND_TYPE_UNPAUSE:
		result = h.exec.Unpause(ctx, containerID)

	case agentpb.CommandType_COMMAND_TYPE_CREATE:
		specJSON, ok := params["spec"]
		if !ok || specJSON == "" {
			return &agentpb.ExecuteCommandResponse{
				Success: false,
				Error:   "missing 'spec' param for CREATE command",
			}, nil
		}
		spec, parseErr := executor.ParseContainerSpec(specJSON)
		if parseErr != nil {
			return &agentpb.ExecuteCommandResponse{
				Success: false,
				Error:   parseErr.Error(),
			}, nil
		}
		result = h.exec.Create(ctx, spec)

	case agentpb.CommandType_COMMAND_TYPE_PULL_IMAGE:
		imageRef, ok := params["image"]
		if !ok || imageRef == "" {
			return &agentpb.ExecuteCommandResponse{
				Success: false,
				Error:   "missing 'image' param for PULL_IMAGE command",
			}, nil
		}
		result = h.exec.PullImage(ctx, imageRef)

	case agentpb.CommandType_COMMAND_TYPE_UPDATE_RESOURCES:
		resJSON, ok := params["resources"]
		if !ok || resJSON == "" {
			return &agentpb.ExecuteCommandResponse{
				Success: false,
				Error:   "missing 'resources' param for UPDATE_RESOURCES command",
			}, nil
		}
		update, parseErr := executor.ParseResourceUpdate(resJSON)
		if parseErr != nil {
			return &agentpb.ExecuteCommandResponse{
				Success: false,
				Error:   parseErr.Error(),
			}, nil
		}
		result = h.exec.UpdateResources(ctx, containerID, update)

	case agentpb.CommandType_COMMAND_TYPE_RECREATE_ENV:
		envJSON, ok := params["env"]
		if !ok {
			return &agentpb.ExecuteCommandResponse{
				Success: false,
				Error:   "missing 'env' param for RECREATE_ENV command",
			}, nil
		}
		var envSlice []string
		if jsonErr := json.Unmarshal([]byte(envJSON), &envSlice); jsonErr != nil {
			return &agentpb.ExecuteCommandResponse{
				Success: false,
				Error:   "invalid 'env' JSON: " + jsonErr.Error(),
			}, nil
		}
		result = h.exec.RecreateWithEnv(ctx, containerID, envSlice)

	case agentpb.CommandType_COMMAND_TYPE_UPDATE_IMAGE:
		result = h.exec.UpdateImage(ctx, containerID)

	default:
		return &agentpb.ExecuteCommandResponse{
			Success: false,
			Error:   fmt.Sprintf("unsupported command type: %s", req.GetType().String()),
		}, nil
	}

	if result.Error != "" {
		h.logger.Warn("command failed",
			"type", req.GetType().String(),
			"container_id", containerID,
			"error", result.Error,
		)
	} else {
		h.logger.Info("command succeeded",
			"type", req.GetType().String(),
			"container_id", containerID,
		)
	}

	return &agentpb.ExecuteCommandResponse{
		Success: result.Success,
		Error:   result.Error,
		Output:  result.Output,
	}, nil
}

// parseIntParam extracts an integer from the params map, returning nil if
// the key is missing or not a valid integer.
func parseIntParam(params map[string]string, key string) *int {
	v, ok := params[key]
	if !ok {
		return nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return nil
	}
	return &n
}
