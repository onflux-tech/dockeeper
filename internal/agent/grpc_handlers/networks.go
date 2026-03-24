package grpc_handlers

import (
	"context"
	"log/slog"

	"github.com/jkdesenvolvedor/dockeeper/internal/agent/executor"
	agentpb "github.com/jkdesenvolvedor/dockeeper/proto/gen/dockeeper/agent/v1"
)

// NetworkHandler implements AgentNetworkServiceServer on the agent side.
type NetworkHandler struct {
	agentpb.UnimplementedAgentNetworkServiceServer
	exec   executor.Executor
	logger *slog.Logger
}

// NewNetworkHandler creates a network handler with the given executor.
func NewNetworkHandler(exec executor.Executor, logger *slog.Logger) *NetworkHandler {
	return &NetworkHandler{exec: exec, logger: logger}
}

// CreateNetwork creates a Docker network.
func (h *NetworkHandler) CreateNetwork(ctx context.Context, req *agentpb.CreateNetworkRequest) (*agentpb.CreateNetworkResponse, error) {
	name := req.GetName()
	h.logger.Info("creating network", "name", name, "driver", req.GetDriver())

	opts := executor.NetworkCreateOpts{
		Name:     name,
		Driver:   req.GetDriver(),
		Subnet:   req.GetSubnet(),
		Gateway:  req.GetGateway(),
		Internal: req.GetInternal(),
		Labels:   req.GetLabels(),
	}

	networkID, err := h.exec.CreateNetwork(ctx, opts)
	if err != nil {
		h.logger.Warn("network create failed", "name", name, "error", err)
		return &agentpb.CreateNetworkResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	h.logger.Info("network created", "name", name, "network_id", networkID)
	return &agentpb.CreateNetworkResponse{
		NetworkId: networkID,
		Success:   true,
	}, nil
}

// RemoveNetwork removes a Docker network.
func (h *NetworkHandler) RemoveNetwork(ctx context.Context, req *agentpb.RemoveNetworkRequest) (*agentpb.RemoveNetworkResponse, error) {
	networkID := req.GetNetworkId()
	h.logger.Info("removing network", "network_id", networkID)

	if err := h.exec.RemoveNetwork(ctx, networkID); err != nil {
		h.logger.Warn("network remove failed", "network_id", networkID, "error", err)
		return &agentpb.RemoveNetworkResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	h.logger.Info("network removed", "network_id", networkID)
	return &agentpb.RemoveNetworkResponse{Success: true}, nil
}

// ConnectContainer connects a container to a network.
func (h *NetworkHandler) ConnectContainer(ctx context.Context, req *agentpb.NetworkConnectRequest) (*agentpb.NetworkConnectResponse, error) {
	h.logger.Info("connecting container to network", "network_id", req.GetNetworkId(), "container_id", req.GetContainerId())

	if err := h.exec.ConnectContainer(ctx, req.GetNetworkId(), req.GetContainerId()); err != nil {
		h.logger.Warn("network connect failed", "error", err)
		return &agentpb.NetworkConnectResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	h.logger.Info("container connected to network", "network_id", req.GetNetworkId(), "container_id", req.GetContainerId())
	return &agentpb.NetworkConnectResponse{Success: true}, nil
}

// DisconnectContainer disconnects a container from a network.
func (h *NetworkHandler) DisconnectContainer(ctx context.Context, req *agentpb.NetworkDisconnectRequest) (*agentpb.NetworkDisconnectResponse, error) {
	h.logger.Info("disconnecting container from network", "network_id", req.GetNetworkId(), "container_id", req.GetContainerId())

	if err := h.exec.DisconnectContainer(ctx, req.GetNetworkId(), req.GetContainerId(), req.GetForce()); err != nil {
		h.logger.Warn("network disconnect failed", "error", err)
		return &agentpb.NetworkDisconnectResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	h.logger.Info("container disconnected from network", "network_id", req.GetNetworkId(), "container_id", req.GetContainerId())
	return &agentpb.NetworkDisconnectResponse{Success: true}, nil
}
