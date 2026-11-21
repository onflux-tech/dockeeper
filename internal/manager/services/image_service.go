package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/channel"
	"github.com/onflux-tech/dockeeper/internal/manager/store"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// ImageService handles image operations by resolving the target server
// and dispatching commands to the agent via the reverse tunnel command channel.
type ImageService struct {
	app      core.App
	audit    *store.AuditStore
	registry *channel.Registry
}

// NewImageService creates an ImageService.
func NewImageService(app core.App, audit *store.AuditStore, registry *channel.Registry) *ImageService {
	return &ImageService{app: app, audit: audit, registry: registry}
}

// ImageInfo holds the resolved image information needed for agent operations.
type ImageInfo struct {
	DockerID  string
	ServerID  string
	ImageName string
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
		DockerID:  img.GetString("image_id"),
		ServerID:  serverID,
		ImageName: img.GetString("tags"),
	}, nil
}

// ResolveServer validates a server is online and returns its ID.
func (s *ImageService) ResolveServer(serverID string) (string, error) {
	server, err := s.app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		return "", ErrServerNotFound
	}
	if server.GetString("status") != "online" {
		return "", ErrServerOffline
	}
	return serverID, nil
}

// RemoveImage removes an image on the agent via the command channel.
func (s *ImageService) RemoveImage(ctx context.Context, imageRecordID string, force bool, userID, ip string) (*agentpb.ImageRemoveResponse, error) {
	info, err := s.ResolveImage(imageRecordID)
	if err != nil {
		return nil, err
	}

	ch, ok := s.registry.Get(info.ServerID)
	if !ok {
		return nil, fmt.Errorf("no active channel for server %s", info.ServerID)
	}

	cmdCtx, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	result, err := ch.SendCommand(cmdCtx, &agentpb.ChannelCommand{
		RequestId: channel.NewRequestID(),
		Priority:  2, // NORMAL
		Command: &agentpb.ChannelCommand_RemoveImage{RemoveImage: &agentpb.ImageRemoveRequest{
			ImageId:       info.DockerID,
			Force:         force,
			PruneChildren: true,
		}},
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

	if errResult := result.GetError(); errResult != nil {
		details["error"] = errResult.GetMessage()
		s.audit.Log("image.remove.failed", userID, "image", imageRecordID, ip, details)
		return nil, fmt.Errorf("agent error: %s", errResult.GetMessage())
	}

	resp := result.GetRemoveImage()
	if resp == nil {
		return nil, fmt.Errorf("unexpected result type from agent")
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

// PruneImages removes unused images on a server via the command channel.
func (s *ImageService) PruneImages(ctx context.Context, serverID string, all bool, userID, ip string) (*agentpb.ImagePruneResponse, error) {
	_, err := s.ResolveServer(serverID)
	if err != nil {
		return nil, err
	}

	ch, ok := s.registry.Get(serverID)
	if !ok {
		return nil, fmt.Errorf("no active channel for server %s", serverID)
	}

	cmdCtx, cancel := context.WithTimeout(ctx, 2*commandTimeout)
	defer cancel()

	result, err := ch.SendCommand(cmdCtx, &agentpb.ChannelCommand{
		RequestId: channel.NewRequestID(),
		Priority:  2, // NORMAL
		Command:   &agentpb.ChannelCommand_PruneImages{PruneImages: &agentpb.ImagePruneRequest{All: all}},
	})

	details := map[string]any{
		"server_id": serverID,
		"all":       all,
	}
	if err != nil {
		details["error"] = err.Error()
		s.audit.Log("image.prune.failed", userID, "image", "", ip, details)
		return nil, fmt.Errorf("prune images: %w", err)
	}

	if errResult := result.GetError(); errResult != nil {
		details["error"] = errResult.GetMessage()
		s.audit.Log("image.prune.failed", userID, "image", "", ip, details)
		return nil, fmt.Errorf("agent error: %s", errResult.GetMessage())
	}

	resp := result.GetPruneImages()
	if resp == nil {
		return nil, fmt.Errorf("unexpected result type from agent")
	}

	details["success"] = resp.GetSuccess()
	details["space_reclaimed"] = resp.GetSpaceReclaimed()
	details["records_removed"] = s.cleanupPrunedImages(serverID, resp)
	s.audit.Log("image.prune", userID, "image", "", ip, details)
	return resp, nil
}

func (s *ImageService) cleanupPrunedImages(serverID string, resp *agentpb.ImagePruneResponse) int {
	if !resp.GetSuccess() {
		return 0
	}
	return s.deletePrunedImages(serverID, resp.GetDeleted())
}

func (s *ImageService) deletePrunedImages(serverID string, deleted []string) int {
	removed := 0
	for _, value := range deleted {
		imageID := shortImageID(value)
		records, err := s.app.FindRecordsByFilter(
			constants.CollImages,
			"server = {:server} && image_id = {:image}",
			"",
			0,
			0,
			map[string]any{"server": serverID, "image": imageID},
		)
		if err != nil {
			slog.Warn("prune images: find image records failed", "server_id", serverID, "image_id", imageID, "error", err)
			continue
		}
		for _, rec := range records {
			if err := s.app.Delete(rec); err != nil {
				slog.Warn("prune images: delete image record failed", "server_id", serverID, "image_id", imageID, "error", err)
				continue
			}
			removed++
		}
	}
	return removed
}

func shortImageID(value string) string {
	id := strings.TrimPrefix(value, "sha256:")
	if len(id) > constants.ImageShortIDLength {
		return id[:constants.ImageShortIDLength]
	}
	return id
}
