package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/pocketbase/pocketbase/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/onflux-tech/dockeeper/internal/manager/store"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// NetworkService handles network operations by resolving the target server
// and dispatching commands to the agent via gRPC.
type NetworkService struct {
	app   core.App
	audit *store.AuditStore
}

// NewNetworkService creates a NetworkService.
func NewNetworkService(app core.App, audit *store.AuditStore) *NetworkService {
	return &NetworkService{app: app, audit: audit}
}

// NetworkInfo holds the resolved network information needed for agent operations.
type NetworkInfo struct {
	AgentAddr string
	DockerID  string
	ServerID  string
	Name      string
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
		AgentAddr: server.GetString("ip_address"),
		DockerID:  net.GetString("network_id"),
		ServerID:  serverID,
		Name:      net.GetString("name"),
	}, nil
}

// CreateNetwork creates a network on an agent via gRPC.
func (s *NetworkService) CreateNetwork(
	ctx context.Context, serverID string,
	name, driver, subnet, gateway string, internal bool,
	userID, ip string,
) (*agentpb.CreateNetworkResponse, error) {
	agentIP, err := s.ResolveServer(serverID)
	if err != nil {
		return nil, err
	}

	conn, client, err := dialNetworkService(agentIP)
	if err != nil {
		return nil, err
	}
	defer closeConn(conn, agentIP)

	cmdCtx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	resp, err := client.CreateNetwork(cmdCtx, &agentpb.CreateNetworkRequest{
		Name:     name,
		Driver:   driver,
		Subnet:   subnet,
		Gateway:  gateway,
		Internal: internal,
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

	if resp.GetSuccess() {
		s.upsertCreatedNetwork(serverID, resp.GetNetworkId(), name, driver, subnet, gateway, internal)
	}

	details["network_id"] = resp.GetNetworkId()
	details["success"] = resp.GetSuccess()
	s.audit.Log("network.create", userID, "network", resp.GetNetworkId(), ip, details)
	return resp, nil
}

// RemoveNetwork removes a network on the agent via gRPC.
func (s *NetworkService) RemoveNetwork(ctx context.Context, networkRecordID, userID, ip string) (*agentpb.RemoveNetworkResponse, error) {
	info, err := s.ResolveNetwork(networkRecordID)
	if err != nil {
		return nil, err
	}

	conn, client, err := dialNetworkService(info.AgentAddr)
	if err != nil {
		return nil, err
	}
	defer closeConn(conn, info.AgentAddr)

	cmdCtx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	resp, err := client.RemoveNetwork(cmdCtx, &agentpb.RemoveNetworkRequest{
		NetworkId: info.DockerID,
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

// ConnectContainer connects a container to a network via the agent.
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

	conn, client, err := dialNetworkService(info.AgentAddr)
	if err != nil {
		return nil, err
	}
	defer closeConn(conn, info.AgentAddr)

	cmdCtx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	resp, err := client.ConnectContainer(cmdCtx, &agentpb.NetworkConnectRequest{
		NetworkId:   info.DockerID,
		ContainerId: dockerContainerID,
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

	details["success"] = resp.GetSuccess()
	s.audit.Log("network.connect", userID, "network", networkRecordID, ip, details)
	return resp, nil
}

// DisconnectContainer disconnects a container from a network via the agent.
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

	conn, client, err := dialNetworkService(info.AgentAddr)
	if err != nil {
		return nil, err
	}
	defer closeConn(conn, info.AgentAddr)

	cmdCtx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	resp, err := client.DisconnectContainer(cmdCtx, &agentpb.NetworkDisconnectRequest{
		NetworkId:   info.DockerID,
		ContainerId: dockerContainerID,
		Force:       force,
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

	details["success"] = resp.GetSuccess()
	s.audit.Log("network.disconnect", userID, "network", networkRecordID, ip, details)
	return resp, nil
}

// ResolveServer validates a server is online and returns its IP address.
// This is a duplicate-safe helper; the canonical one is in image_service.go.
func (s *NetworkService) ResolveServer(serverID string) (string, error) {
	server, err := s.app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		return "", ErrServerNotFound
	}
	if server.GetString("status") != "online" {
		return "", ErrServerOffline
	}
	return server.GetString("ip_address"), nil
}

func dialNetworkService(agentIP string) (*grpc.ClientConn, agentpb.AgentNetworkServiceClient, error) {
	addr := agentIP + ":" + agentCommandPort
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("dial agent %s: %w", addr, err)
	}
	return conn, agentpb.NewAgentNetworkServiceClient(conn), nil
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
