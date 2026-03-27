package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/grpc_pool"
	"github.com/onflux-tech/dockeeper/internal/manager/store"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// VolumeService handles volume operations by resolving the target server
// and dispatching commands to the agent via gRPC.
type VolumeService struct {
	app   core.App
	audit *store.AuditStore
	pool  *grpc_pool.Pool
}

// NewVolumeService creates a VolumeService.
func NewVolumeService(app core.App, audit *store.AuditStore, pool *grpc_pool.Pool) *VolumeService {
	return &VolumeService{app: app, audit: audit, pool: pool}
}

// VolumeInfo holds the resolved volume information needed for agent operations.
type VolumeInfo struct {
	AgentAddr string
	Name      string
	ServerID  string
	InUse     bool
}

// ResolveVolume looks up a volume PocketBase record and returns agent info.
func (s *VolumeService) ResolveVolume(volumeRecordID string) (*VolumeInfo, error) {
	vol, err := s.app.FindRecordById(constants.CollVolumes, volumeRecordID)
	if err != nil {
		return nil, fmt.Errorf("volume not found")
	}

	serverID := vol.GetString("server")
	server, err := s.app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		return nil, ErrServerNotFound
	}

	if server.GetString("status") != "online" {
		return nil, ErrServerOffline
	}

	return &VolumeInfo{
		AgentAddr: server.GetString("ip_address"),
		Name:      vol.GetString("name"),
		ServerID:  serverID,
		InUse:     vol.GetBool("in_use"),
	}, nil
}

// CreateVolume creates a volume on an agent via gRPC.
func (s *VolumeService) CreateVolume(
	ctx context.Context, serverID, name, driver, userID, ip string,
) (*agentpb.CreateVolumeResponse, error) {
	agentIP, err := s.resolveServer(serverID)
	if err != nil {
		return nil, err
	}

	client, err := s.volumeClient(agentIP)
	if err != nil {
		return nil, err
	}

	cmdCtx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	resp, err := client.CreateVolume(cmdCtx, &agentpb.CreateVolumeRequest{
		Name:   name,
		Driver: driver,
	})

	details := map[string]any{
		"server_id": serverID,
		"name":      name,
		"driver":    driver,
	}
	if err != nil {
		details["error"] = err.Error()
		s.audit.Log("volume.create.failed", userID, "volume", "", ip, details)
		return nil, fmt.Errorf("create volume: %w", err)
	}

	if resp.GetSuccess() {
		s.upsertCreatedVolume(serverID, resp.GetName(), driver)
	}

	details["success"] = resp.GetSuccess()
	s.audit.Log("volume.create", userID, "volume", resp.GetName(), ip, details)
	return resp, nil
}

// RemoveVolume removes a volume on the agent via gRPC.
func (s *VolumeService) RemoveVolume(ctx context.Context, volumeRecordID string, force bool, userID, ip string) (*agentpb.RemoveVolumeResponse, error) {
	info, err := s.ResolveVolume(volumeRecordID)
	if err != nil {
		return nil, err
	}

	client, err := s.volumeClient(info.AgentAddr)
	if err != nil {
		return nil, err
	}

	cmdCtx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	resp, err := client.RemoveVolume(cmdCtx, &agentpb.RemoveVolumeRequest{
		Name:  info.Name,
		Force: force,
	})

	details := map[string]any{
		"name":      info.Name,
		"server_id": info.ServerID,
		"force":     force,
	}
	if err != nil {
		details["error"] = err.Error()
		s.audit.Log("volume.remove.failed", userID, "volume", volumeRecordID, ip, details)
		return nil, fmt.Errorf("remove volume: %w", err)
	}

	if resp.GetSuccess() {
		if rec, findErr := s.app.FindRecordById(constants.CollVolumes, volumeRecordID); findErr == nil {
			if delErr := s.app.Delete(rec); delErr != nil {
				slog.Warn("failed to delete volume record after remove", "volume_record_id", volumeRecordID, "error", delErr)
			}
		}
	}

	details["success"] = resp.GetSuccess()
	s.audit.Log("volume.remove", userID, "volume", volumeRecordID, ip, details)
	return resp, nil
}

// InspectVolume inspects a volume on the agent via gRPC.
func (s *VolumeService) InspectVolume(ctx context.Context, volumeRecordID string) (*agentpb.InspectVolumeResponse, error) {
	info, err := s.ResolveVolume(volumeRecordID)
	if err != nil {
		return nil, err
	}

	client, err := s.volumeClient(info.AgentAddr)
	if err != nil {
		return nil, err
	}

	cmdCtx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	resp, err := client.InspectVolume(cmdCtx, &agentpb.InspectVolumeRequest{
		Name: info.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("inspect volume: %w", err)
	}

	return resp, nil
}

func (s *VolumeService) resolveServer(serverID string) (string, error) {
	server, err := s.app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		return "", ErrServerNotFound
	}
	if server.GetString("status") != "online" {
		return "", ErrServerOffline
	}
	return server.GetString("ip_address"), nil
}

func (s *VolumeService) volumeClient(agentIP string) (agentpb.AgentVolumeServiceClient, error) {
	conn, err := s.pool.Get(agentIP)
	if err != nil {
		return nil, err
	}
	return agentpb.NewAgentVolumeServiceClient(conn), nil
}

func (s *VolumeService) upsertCreatedVolume(serverID, name, driver string) {
	col, err := s.app.FindCachedCollectionByNameOrId(constants.CollVolumes)
	if err != nil {
		slog.Warn("failed to find volumes collection", "error", err)
		return
	}

	records, err := s.app.FindRecordsByFilter(constants.CollVolumes, "server = {:sid} && name = {:name}", "", 1, 0, map[string]any{
		"sid":  serverID,
		"name": name,
	})
	var rec *core.Record
	if err == nil && len(records) > 0 {
		rec = records[0]
	} else {
		rec = core.NewRecord(col)
		rec.Set("server", serverID)
	}

	rec.Set("name", name)
	rec.Set("driver", driver)
	rec.Set("in_use", false)

	if saveErr := s.app.Save(rec); saveErr != nil {
		slog.Warn("failed to save volume record after create", "volume_name", name, "error", saveErr)
	}
}
