package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/channel"
	"github.com/onflux-tech/dockeeper/internal/manager/store"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// NetworkService handles network operations by resolving the target server
// and dispatching commands to the agent via the reverse tunnel command channel.
type NetworkService struct {
	app      core.App
	audit    *store.AuditStore
	registry *channel.Registry
}

// NewNetworkService creates a NetworkService.
func NewNetworkService(app core.App, audit *store.AuditStore, registry *channel.Registry) *NetworkService {
	return &NetworkService{app: app, audit: audit, registry: registry}
}

// NetworkInfo holds the resolved network information needed for agent operations.
type NetworkInfo struct {
	DockerID string
	ServerID string
	Name     string
}

// ResolveNetwork looks up a network PocketBase record and returns agent info.
func (s *NetworkService) ResolveNetwork(networkRecordID string) (*NetworkInfo, error) {
	net, err := s.app.FindRecordById(constants.CollNetworks, networkRecordID)
	if err != nil {
		return nil, fmt.Errorf("network not found")
	}

	serverID := net.GetString("server")
	server, err := s.app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		return nil, ErrServerNotFound
	}

	if server.GetString("status") != "online" {
		return nil, ErrServerOffline
	}

	return &NetworkInfo{
		DockerID: net.GetString("network_id"),
		ServerID: serverID,
		Name:     net.GetString("name"),
	}, nil
}

// CreateNetwork creates a network on an agent via the command channel.
func (s *NetworkService) CreateNetwork(
	ctx context.Context, serverID string,
	name, driver, subnet, gateway string, internal bool,
	userID, ip string,
) (*agentpb.CreateNetworkResponse, error) {
	if _, err := s.ResolveServer(serverID); err != nil {
		return nil, err
	}

	ch, ok := s.registry.Get(serverID)
	if !ok {
		return nil, fmt.Errorf("no active channel for server %s", serverID)
	}

	cmdCtx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	result, err := ch.SendCommand(cmdCtx, &agentpb.ChannelCommand{
		RequestId: channel.NewRequestID(),
		Priority:  1,
		Command: &agentpb.ChannelCommand_CreateNetwork{CreateNetwork: &agentpb.CreateNetworkRequest{
			Name:     name,
			Driver:   driver,
			Subnet:   subnet,
			Gateway:  gateway,
			Internal: internal,
		}},
	})

	details := map[string]any{
		"server_id": serverID,
		"name":      name,
		"driver":    driver,
	}
	if err != nil {
		details["error"] = err.Error()
		s.audit.Log("network.create.failed", userID, "network", "", ip, details)
		return nil, fmt.Errorf("create network: %w", err)
	}

	if errResult := result.GetError(); errResult != nil {
		details["error"] = errResult.GetMessage()
		s.audit.Log("network.create.failed", userID, "network", "", ip, details)
		return nil, fmt.Errorf("create network: %s", errResult.GetMessage())
	}

	resp := result.GetCreateNetwork()
	if resp == nil {
		return nil, fmt.Errorf("unexpected result type from agent")
	}

	if resp.GetSuccess() {
		s.upsertCreatedNetwork(serverID, resp.GetNetworkId(), name, driver, subnet, gateway, internal)
	}

	details["network_id"] = resp.GetNetworkId()
	details["success"] = resp.GetSuccess()
	s.audit.Log("network.create", userID, "network", resp.GetNetworkId(), ip, details)
	return resp, nil
}

// RemoveNetwork removes a network on the agent via the command channel.
func (s *NetworkService) RemoveNetwork(ctx context.Context, networkRecordID, userID, ip string) (*agentpb.RemoveNetworkResponse, error) {
	info, err := s.ResolveNetwork(networkRecordID)
	if err != nil {
		return nil, err
	}

	if isProtectedDockerNetwork(info.Name) {
		return nil, ErrProtectedNetwork
	}

	ch, ok := s.registry.Get(info.ServerID)
	if !ok {
		return nil, fmt.Errorf("no active channel for server %s", info.ServerID)
	}

	cmdCtx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	result, err := ch.SendCommand(cmdCtx, &agentpb.ChannelCommand{
		RequestId: channel.NewRequestID(),
		Priority:  1,
		Command: &agentpb.ChannelCommand_RemoveNetwork{RemoveNetwork: &agentpb.RemoveNetworkRequest{
			NetworkId: info.DockerID,
		}},
	})

	details := map[string]any{
		"network_id": info.DockerID,
		"server_id":  info.ServerID,
		"name":       info.Name,
	}
	if err != nil {
		details["error"] = err.Error()
		s.audit.Log("network.remove.failed", userID, "network", networkRecordID, ip, details)
		return nil, fmt.Errorf("remove network: %w", err)
	}

	if errResult := result.GetError(); errResult != nil {
		details["error"] = errResult.GetMessage()
		s.audit.Log("network.remove.failed", userID, "network", networkRecordID, ip, details)
		return nil, fmt.Errorf("remove network: %s", errResult.GetMessage())
	}

	resp := result.GetRemoveNetwork()
	if resp == nil {
		return nil, fmt.Errorf("unexpected result type from agent")
	}

	if resp.GetSuccess() {
		if rec, findErr := s.app.FindRecordById(constants.CollNetworks, networkRecordID); findErr == nil {
			if delErr := s.app.Delete(rec); delErr != nil {
				slog.Warn("failed to delete network record after remove", "network_record_id", networkRecordID, "error", delErr)
			}
		}
	}

	details["success"] = resp.GetSuccess()
	s.audit.Log("network.remove", userID, "network", networkRecordID, ip, details)
	return resp, nil
}

// ConnectContainer connects a container to a network via the command channel.
func (s *NetworkService) ConnectContainer(ctx context.Context, networkRecordID, containerRecordID, userID, ip string) (*agentpb.NetworkConnectResponse, error) {
	info, err := s.ResolveNetwork(networkRecordID)
	if err != nil {
		return nil, err
	}

	// Resolve container Docker ID.
	container, err := s.app.FindRecordById(constants.CollContainers, containerRecordID)
	if err != nil {
		return nil, ErrContainerNotFound
	}
	dockerContainerID := container.GetString("container_id")

	ch, ok := s.registry.Get(info.ServerID)
	if !ok {
		return nil, fmt.Errorf("no active channel for server %s", info.ServerID)
	}

	cmdCtx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	result, err := ch.SendCommand(cmdCtx, &agentpb.ChannelCommand{
		RequestId: channel.NewRequestID(),
		Priority:  1,
		Command: &agentpb.ChannelCommand_ConnectContainer{ConnectContainer: &agentpb.NetworkConnectRequest{
			NetworkId:   info.DockerID,
			ContainerId: dockerContainerID,
		}},
	})

	details := map[string]any{
		"network_id":   info.DockerID,
		"container_id": dockerContainerID,
		"server_id":    info.ServerID,
	}
	if err != nil {
		details["error"] = err.Error()
		s.audit.Log("network.connect.failed", userID, "network", networkRecordID, ip, details)
		return nil, fmt.Errorf("connect container: %w", err)
	}

	if errResult := result.GetError(); errResult != nil {
		details["error"] = errResult.GetMessage()
		s.audit.Log("network.connect.failed", userID, "network", networkRecordID, ip, details)
		return nil, fmt.Errorf("connect container: %s", errResult.GetMessage())
	}

	resp := result.GetConnectContainer()
	if resp == nil {
		return nil, fmt.Errorf("unexpected result type from agent")
	}

	details["success"] = resp.GetSuccess()
	s.audit.Log("network.connect", userID, "network", networkRecordID, ip, details)
	return resp, nil
}

// DisconnectContainer disconnects a container from a network via the command channel.
func (s *NetworkService) DisconnectContainer(ctx context.Context, networkRecordID, containerRecordID string, force bool, userID, ip string) (*agentpb.NetworkDisconnectResponse, error) {
	info, err := s.ResolveNetwork(networkRecordID)
	if err != nil {
		return nil, err
	}

	container, err := s.app.FindRecordById(constants.CollContainers, containerRecordID)
	if err != nil {
		return nil, ErrContainerNotFound
	}
	dockerContainerID := container.GetString("container_id")

	ch, ok := s.registry.Get(info.ServerID)
	if !ok {
		return nil, fmt.Errorf("no active channel for server %s", info.ServerID)
	}

	cmdCtx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	result, err := ch.SendCommand(cmdCtx, &agentpb.ChannelCommand{
		RequestId: channel.NewRequestID(),
		Priority:  1,
		Command: &agentpb.ChannelCommand_DisconnectContainer{DisconnectContainer: &agentpb.NetworkDisconnectRequest{
			NetworkId:   info.DockerID,
			ContainerId: dockerContainerID,
			Force:       force,
		}},
	})

	details := map[string]any{
		"network_id":   info.DockerID,
		"container_id": dockerContainerID,
		"server_id":    info.ServerID,
		"force":        force,
	}
	if err != nil {
		details["error"] = err.Error()
		s.audit.Log("network.disconnect.failed", userID, "network", networkRecordID, ip, details)
		return nil, fmt.Errorf("disconnect container: %w", err)
	}

	if errResult := result.GetError(); errResult != nil {
		details["error"] = errResult.GetMessage()
		s.audit.Log("network.disconnect.failed", userID, "network", networkRecordID, ip, details)
		return nil, fmt.Errorf("disconnect container: %s", errResult.GetMessage())
	}

	resp := result.GetDisconnectContainer()
	if resp == nil {
		return nil, fmt.Errorf("unexpected result type from agent")
	}

	details["success"] = resp.GetSuccess()
	s.audit.Log("network.disconnect", userID, "network", networkRecordID, ip, details)
	return resp, nil
}

// ResolveServer validates a server is online and returns its ID.
func (s *NetworkService) ResolveServer(serverID string) (string, error) {
	server, err := s.app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		return "", ErrServerNotFound
	}
	if server.GetString("status") != "online" {
		return "", ErrServerOffline
	}
	return serverID, nil
}

func (s *NetworkService) upsertCreatedNetwork(serverID, networkID, name, driver, subnet, gateway string, internal bool) {
	col, err := s.app.FindCachedCollectionByNameOrId(constants.CollNetworks)
	if err != nil {
		slog.Warn("failed to find networks collection", "error", err)
		return
	}

	records, err := s.app.FindRecordsByFilter(constants.CollNetworks, "server = {:sid} && network_id = {:nid}", "", 1, 0, map[string]any{
		"sid": serverID,
		"nid": networkID,
	})
	var rec *core.Record
	if err == nil && len(records) > 0 {
		rec = records[0]
	} else {
		rec = core.NewRecord(col)
		rec.Set("server", serverID)
		rec.Set("network_id", networkID)
	}

	rec.Set("name", name)
	rec.Set("driver", driver)
	rec.Set("subnet", subnet)
	rec.Set("gateway", gateway)
	rec.Set("internal", internal)

	if saveErr := s.app.Save(rec); saveErr != nil {
		slog.Warn("failed to save network record after create", "network_id", networkID, "error", saveErr)
	}
}
