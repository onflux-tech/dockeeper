package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/store"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// CascadeService handles cascade delete operations for servers, containers, and volumes.
type CascadeService struct {
	app   core.App
	audit *store.AuditStore
}

// NewCascadeService creates a CascadeService.
func NewCascadeService(app core.App, audit *store.AuditStore) *CascadeService {
	return &CascadeService{
		app:   app,
		audit: audit,
	}
}

// ── In-Use Detection ────────────────────────────────

// IsVolumeExclusive checks whether a volume is exclusively used by the specified containers.
// Returns: (isExclusive bool, otherContainerNames []string, error)
// If isExclusive is false, otherContainerNames lists who else is using the volume.
func (s *CascadeService) IsVolumeExclusive(volumeRecordID string, excludeContainerIDs []string) (bool, []string, error) {
	// 1. Find the volume record to get server info.
	volumeRecord, err := s.app.FindRecordById(constants.CollVolumes, volumeRecordID)
	if err != nil {
		return false, nil, fmt.Errorf("find volume: %w", err)
	}

	volumeName := volumeRecord.GetString("name")
	serverID := volumeRecord.GetString("server")
	if volumeName == "" || serverID == "" {
		return false, nil, errors.New("volume has no name or server")
	}

	// 2. Query all containers on this server.
	filter := fmt.Sprintf("server = '%s'", escapeFilterValue(serverID))
	containerRecords, err := s.app.FindRecordsByFilter(
		constants.CollContainers,
		filter,
		"-created", // sort descending by created
		999,        // limit high number
		0,          // offset
	)
	if err != nil {
		return false, nil, fmt.Errorf("list containers for server: %w", err)
	}

	// Convert exclude list to map for O(1) lookup
	excludeMap := make(map[string]bool, len(excludeContainerIDs))
	for _, id := range excludeContainerIDs {
		excludeMap[id] = true
	}

	// 3. Check which other containers use this volume.
	var otherContainers []string
	for _, container := range containerRecords {
		containerID := container.Id
		// Skip containers marked for exclusion.
		if excludeMap[containerID] {
			continue
		}

		volumesJSON := container.GetString("volumes")
		if volumesJSON == "" {
			continue
		}

		// Parse volumes JSON and check if this volume is referenced.
		var mounts []MountPoint
		if err := json.Unmarshal([]byte(volumesJSON), &mounts); err != nil {
			slog.Warn("failed to parse volumes JSON", "container_id", containerID, "error", err)
			continue
		}

		for _, mount := range mounts {
			// Match by volume name (for named volumes) or source path (for bind mounts).
			if mount.Name == volumeName || mount.Source == volumeName {
				otherContainers = append(otherContainers, container.GetString("name"))
				break
			}
		}
	}

	// 4. If no other containers use it, the volume is exclusive.
	if len(otherContainers) == 0 {
		return true, nil, nil
	}

	return false, otherContainers, nil
}

// IsImageExclusive checks whether an image is exclusively used by the specified containers.
// Returns: (isExclusive bool, otherContainerNames []string, error)
// If isExclusive is false, otherContainerNames lists who else is using the image.
func (s *CascadeService) IsImageExclusive(imageTag string, serverID string, excludeContainerIDs []string) (bool, []string, error) {
	if imageTag == "" || serverID == "" {
		return false, nil, errors.New("imageTag and serverID are required")
	}

	// 1. Query all containers on this server that use the image.
	filter := fmt.Sprintf("server = '%s'", escapeFilterValue(serverID))
	containerRecords, err := s.app.FindRecordsByFilter(
		constants.CollContainers,
		filter,
		"-created",
		999,
		0,
	)
	if err != nil {
		return false, nil, fmt.Errorf("list containers for server: %w", err)
	}

	// Convert exclude list to map for O(1) lookup
	excludeMap := make(map[string]bool, len(excludeContainerIDs))
	for _, id := range excludeContainerIDs {
		excludeMap[id] = true
	}

	// 2. Check which other containers use this image.
	var otherContainers []string
	for _, container := range containerRecords {
		containerID := container.Id
		// Skip containers marked for exclusion.
		if excludeMap[containerID] {
			continue
		}

		containerImage := container.GetString("image")
		// Simple string match: check if containerImage contains the imageTag.
		// This is a basic match; production code might need more sophisticated logic.
		if strings.Contains(containerImage, imageTag) || containerImage == imageTag {
			otherContainers = append(otherContainers, container.GetString("name"))
		}
	}

	// 3. If no other containers use it, the image is exclusive.
	if len(otherContainers) == 0 {
		return true, nil, nil
	}

	return false, otherContainers, nil
}

// IsNetworkExclusive checks whether a network is exclusively used by the specified containers.
// Returns: (isExclusive bool, otherContainerNames []string, error)
// If isExclusive is false, otherContainerNames lists who else is using the network.
func (s *CascadeService) IsNetworkExclusive(networkName string, serverID string, excludeContainerIDs []string) (bool, []string, error) {
	if networkName == "" || serverID == "" {
		return false, nil, errors.New("networkName and serverID are required")
	}

	// 1. Query all containers on this server.
	filter := fmt.Sprintf("server = '%s'", escapeFilterValue(serverID))
	containerRecords, err := s.app.FindRecordsByFilter(
		constants.CollContainers,
		filter,
		"-created",
		999,
		0,
	)
	if err != nil {
		return false, nil, fmt.Errorf("list containers for server: %w", err)
	}

	// Convert exclude list to map for O(1) lookup
	excludeMap := make(map[string]bool, len(excludeContainerIDs))
	for _, id := range excludeContainerIDs {
		excludeMap[id] = true
	}

	// 2. Check which other containers use this network.
	var otherContainers []string
	for _, container := range containerRecords {
		containerID := container.Id
		// Skip containers marked for exclusion.
		if excludeMap[containerID] {
			continue
		}

		networksJSON := container.GetString("networks")
		if networksJSON == "" {
			continue
		}

		// Parse networks JSON (it's a map of network names to EndpointSettings).
		var networks EndpointSettings
		if err := json.Unmarshal([]byte(networksJSON), &networks); err != nil {
			slog.Warn("failed to parse networks JSON", "container_id", containerID, "error", err)
			continue
		}

		// Check if the network name is a key in the map.
		if _, exists := networks[networkName]; exists {
			otherContainers = append(otherContainers, container.GetString("name"))
		}
	}

	// 3. If no other containers use it, the network is exclusive.
	if len(otherContainers) == 0 {
		return true, nil, nil
	}

	return false, otherContainers, nil
}

// escapeFilterValue escapes a value for use in PocketBase filter strings.
// This is a simple escape that handles single quotes.
func escapeFilterValue(value string) string {
	return strings.ReplaceAll(value, "'", "\\'")
}
