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

// ImageService handles image operations by resolving the target server
// and dispatching commands to the agent via gRPC.
type ImageService struct {
	app   core.App
	audit *store.AuditStore
}

// NewImageService creates an ImageService.
func NewImageService(app core.App, audit *store.AuditStore) *ImageService {
	return &ImageService{app: app, audit: audit}
}

// ImageInfo holds the resolved image information needed for agent operations.
type ImageInfo struct {
	AgentAddr    string
	DockerID     string
	ServerID     string
	ImageName    string
}

// ResolveImage looks up an image PocketBase record and returns agent info.
func (s *ImageService) ResolveImage(imageRecordID string) (*ImageInfo, error) {
	img, err := s.app.FindRecordById(constants.CollImages, imageRecordID)
	if err != nil {
		return nil, fmt.Errorf("image not found")
	}

	serverID := img.GetString("server")
	server, err := s.app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		return nil, ErrServerNotFound
	}

	if server.GetString("status") != "online" {
		return nil, ErrServerOffline
	}

	return &ImageInfo{
		AgentAddr: server.GetString("ip_address"),
		DockerID:  img.GetString("image_id"),
		ServerID:  serverID,
		ImageName: img.GetString("tags"),
	}, nil
}

// ResolveServer validates a server is online and returns its IP address.
func (s *ImageService) ResolveServer(serverID string) (string, error) {
	server, err := s.app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		return "", ErrServerNotFound
	}
	if server.GetString("status") != "online" {
		return "", ErrServerOffline
	}
	return server.GetString("ip_address"), nil
}

// RemoveImage removes an image on the agent via gRPC.
func (s *ImageService) RemoveImage(ctx context.Context, imageRecordID string, force bool, userID, ip string) (*agentpb.ImageRemoveResponse, error) {
	info, err := s.ResolveImage(imageRecordID)
	if err != nil {
		return nil, err
	}

	conn, client, err := dialImageService(info.AgentAddr)
	if err != nil {
		return nil, err
	}
	defer closeConn(conn, info.AgentAddr)

	cmdCtx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	resp, err := client.RemoveImage(cmdCtx, &agentpb.ImageRemoveRequest{
		ImageId:       info.DockerID,
		Force:         force,
		PruneChildren: true,
	})

	details := map[string]any{
		"image_id":  info.DockerID,
		"server_id": info.ServerID,
		"force":     force,
	}
	if err != nil {
		details["error"] = err.Error()
		s.audit.Log("image.remove.failed", userID, "image", imageRecordID, ip, details)
		return nil, fmt.Errorf("remove image: %w", err)
	}

	details["success"] = resp.GetSuccess()
	s.audit.Log("image.remove", userID, "image", imageRecordID, ip, details)
	if resp.GetSuccess() {
		if img, findErr := s.app.FindRecordById(constants.CollImages, imageRecordID); findErr == nil {
			if delErr := s.app.Delete(img); delErr != nil {
				slog.Warn("failed to delete image record after remove", "image_record_id", imageRecordID, "error", delErr)
			}
		}
	}
	return resp, nil
}

// PruneImages removes unused images on a server via gRPC.
func (s *ImageService) PruneImages(ctx context.Context, serverID string, all bool, userID, ip string) (*agentpb.ImagePruneResponse, error) {
	agentIP, err := s.ResolveServer(serverID)
	if err != nil {
		return nil, err
	}

	conn, client, err := dialImageService(agentIP)
	if err != nil {
		return nil, err
	}
	defer closeConn(conn, agentIP)

	cmdCtx, cancel := context.WithTimeout(ctx, 2*commandTimeout)
	defer cancel()

	resp, err := client.PruneImages(cmdCtx, &agentpb.ImagePruneRequest{All: all})

	details := map[string]any{
		"server_id": serverID,
		"all":       all,
	}
	if err != nil {
		details["error"] = err.Error()
		s.audit.Log("image.prune.failed", userID, "image", "", ip, details)
		return nil, fmt.Errorf("prune images: %w", err)
	}

	details["success"] = resp.GetSuccess()
	details["space_reclaimed"] = resp.GetSpaceReclaimed()
	s.audit.Log("image.prune", userID, "image", "", ip, details)
	return resp, nil
}

func dialImageService(agentIP string) (*grpc.ClientConn, agentpb.AgentImageServiceClient, error) {
	addr := agentIP + ":" + agentCommandPort
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("dial agent %s: %w", addr, err)
	}
	return conn, agentpb.NewAgentImageServiceClient(conn), nil
}

func closeConn(conn *grpc.ClientConn, addr string) {
	if err := conn.Close(); err != nil {
		slog.Warn("failed to close agent connection", "addr", addr, "error", err)
	}
}
