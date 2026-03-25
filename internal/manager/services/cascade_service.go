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

// ── Preview Methods ────────────────────────────────

// PreviewContainerRemove returns what would be affected by removing a container.
func (s *CascadeService) PreviewContainerRemove(containerID string) (*ContainerCascadePreview, error) {
	// 1. Fetch container record
	containerRecord, err := s.app.FindRecordById(constants.CollContainers, containerID)
	if err != nil {
		return nil, fmt.Errorf("find container: %w", err)
	}

	containerName := containerRecord.GetString("name")
	serverID := containerRecord.GetString("server")

	// 2. Count alert_rules for this container
	alertFilter := fmt.Sprintf("target_type = 'container' AND target_id = '%s'", escapeFilterValue(containerID))
	alertRules, err := s.app.FindRecordsByFilter(constants.CollAlertRules, alertFilter, "", 999, 0)
	if err != nil {
		return nil, fmt.Errorf("count alert rules: %w", err)
	}

	// 3. Analyze volumes for this container
	volumesJSON := containerRecord.GetString("volumes")
	var volumeItems []ResourceItem
	var volumeMounts []MountPoint

	if volumesJSON != "" {
		if err := json.Unmarshal([]byte(volumesJSON), &volumeMounts); err != nil {
			slog.Warn("failed to parse volumes JSON", "container_id", containerID, "error", err)
		}
	}

	// For each volume mount, check if it's exclusive or shared
	for _, mount := range volumeMounts {
		if mount.Name == "" && mount.Source == "" {
			continue
		}

		volumeName := mount.Name
		if volumeName == "" {
			volumeName = mount.Source
		}

		// Find volume record
		volFilter := fmt.Sprintf("name = '%s' AND server = '%s'", escapeFilterValue(volumeName), escapeFilterValue(serverID))
		volRecords, err := s.app.FindRecordsByFilter(constants.CollVolumes, volFilter, "", 1, 0)
		if err != nil || len(volRecords) == 0 {
			continue
		}

		volRecord := volRecords[0]
		isExclusive, otherContainers, err := s.IsVolumeExclusive(volRecord.Id, []string{containerID})
		reason := ""
		if !isExclusive && err == nil {
			reason = fmt.Sprintf("in use by %d other container(s)", len(otherContainers))
		}

		volumeItems = append(volumeItems, ResourceItem{
			ID:       volRecord.Id,
			Name:     volumeName,
			Eligible: isExclusive,
			Reason:   reason,
		})
	}

	deletableVolumes := int64(0)
	protectedVolumes := int64(0)
	for _, item := range volumeItems {
		if item.Eligible {
			deletableVolumes++
		} else {
			protectedVolumes++
		}
	}

	// 4. Count notification_logs for these alerts
	var alertIDs []string
	for _, alert := range alertRules {
		alertIDs = append(alertIDs, alert.Id)
	}

	notifCount := int64(0)
	if len(alertIDs) > 0 {
		notifFilter := fmt.Sprintf("alert IN ('%s')", strings.Join(alertIDs, "','"))
		notifLogs, err := s.app.FindRecordsByFilter(constants.CollNotificationLogs, notifFilter, "", 999, 0)
		if err == nil {
			notifCount = int64(len(notifLogs))
		}
	}

	// 5. Count backup_policies for container's volumes
	backupPoliciesCount := int64(0)
	if len(volumeItems) > 0 {
		for _, item := range volumeItems {
			polFilter := fmt.Sprintf("volume = '%s'", escapeFilterValue(item.ID))
			policies, err := s.app.FindRecordsByFilter(constants.CollBackupPolicies, polFilter, "", 999, 0)
			if err == nil {
				backupPoliciesCount += int64(len(policies))
			}
		}
	}

	// 6. Count terminal_sessions for this container
	terminalFilter := fmt.Sprintf("container = '%s'", escapeFilterValue(containerID))
	terminalSessions, err := s.app.FindRecordsByFilter(constants.CollTerminalSessions, terminalFilter, "", 999, 0)
	terminalCount := int64(0)
	if err == nil {
		terminalCount = int64(len(terminalSessions))
	}

	// 7. Count metrics_records for this container
	metricsFilter := fmt.Sprintf("container = '%s'", escapeFilterValue(containerID))
	metricsRecords, err := s.app.FindRecordsByFilter(constants.CollServerMetricsRaw, metricsFilter, "", 999, 0)
	metricsCount := int64(0)
	if err == nil {
		metricsCount = int64(len(metricsRecords))
	}

	preview := &ContainerCascadePreview{
		Entity: Entity{
			ID:   containerID,
			Name: containerName,
			Type: "container",
		},
		Impact: CascadeImpact{
			AlertRules: ResourceCount{
				Deletable: int64(len(alertRules)),
				Protected: 0,
			},
			Volumes: ResourceCountWithItems{
				Deletable: deletableVolumes,
				Protected: protectedVolumes,
				Items:     volumeItems,
			},
			NotificationLogs: ResourceCount{
				Deletable: notifCount,
				Protected: 0,
			},
			BackupPolicies: ResourceCount{
				Deletable: backupPoliciesCount,
				Protected: 0,
			},
			TerminalSessions: ResourceCount{
				Deletable: terminalCount,
				Protected: 0,
			},
			MetricsRecords: ResourceCount{
				Deletable: metricsCount,
				Protected: 0,
			},
		},
	}

	return preview, nil
}

// PreviewVolumeRemove returns what would be affected by removing a volume.
func (s *CascadeService) PreviewVolumeRemove(volumeID string) (*VolumeCascadePreview, error) {
	// 1. Fetch volume record
	volumeRecord, err := s.app.FindRecordById(constants.CollVolumes, volumeID)
	if err != nil {
		return nil, fmt.Errorf("find volume: %w", err)
	}

	volumeName := volumeRecord.GetString("name")
	serverID := volumeRecord.GetString("server")

	// 2. Find containers using this volume
	inUseBy, err := s.findContainersUsingVolume(volumeName, serverID)
	if err != nil {
		slog.Warn("failed to find containers using volume", "volume_id", volumeID, "error", err)
	}

	// 3. Count backup_policies for this volume
	polFilter := fmt.Sprintf("volume = '%s'", escapeFilterValue(volumeID))
	policies, err := s.app.FindRecordsByFilter(constants.CollBackupPolicies, polFilter, "", 999, 0)
	policiesCount := int64(0)
	if err == nil {
		policiesCount = int64(len(policies))
	}

	// 4. Count backup_records for these policies
	backupRecordsCount := int64(0)
	if policiesCount > 0 {
		var policyIDs []string
		for _, policy := range policies {
			policyIDs = append(policyIDs, policy.Id)
		}
		recordFilter := fmt.Sprintf("policy IN ('%s')", strings.Join(policyIDs, "','"))
		records, err := s.app.FindRecordsByFilter(constants.CollBackupRecords, recordFilter, "", 999, 0)
		if err == nil {
			backupRecordsCount = int64(len(records))
		}
	}

	preview := &VolumeCascadePreview{
		Entity: Entity{
			ID:   volumeID,
			Name: volumeName,
			Type: "volume",
		},
		Impact: VolumeCascadeImpact{
			BackupPolicies: ResourceCount{
				Deletable: policiesCount,
				Protected: 0,
			},
			BackupRecords: ResourceCount{
				Deletable: backupRecordsCount,
				Protected: 0,
			},
		},
		InUseBy: inUseBy,
	}

	return preview, nil
}

// PreviewServerDelete returns what would be affected by removing a server.
func (s *CascadeService) PreviewServerDelete(serverID string) (*ServerCascadePreview, error) {
	// 1. Fetch server record
	serverRecord, err := s.app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		return nil, fmt.Errorf("find server: %w", err)
	}

	serverName := serverRecord.GetString("hostname")

	serverFilter := fmt.Sprintf("server = '%s'", escapeFilterValue(serverID))

	// 2. Count containers
	containers, err := s.app.FindRecordsByFilter(constants.CollContainers, serverFilter, "", 999, 0)
	containersCount := int64(0)
	if err == nil {
		containersCount = int64(len(containers))
	}

	// 3. Count volumes
	volumes, err := s.app.FindRecordsByFilter(constants.CollVolumes, serverFilter, "", 999, 0)
	volumesCount := int64(0)
	if err == nil {
		volumesCount = int64(len(volumes))
	}

	// 4. Count images
	images, err := s.app.FindRecordsByFilter(constants.CollImages, serverFilter, "", 999, 0)
	imagesCount := int64(0)
	if err == nil {
		imagesCount = int64(len(images))
	}

	// 5. Count networks
	networks, err := s.app.FindRecordsByFilter(constants.CollNetworks, serverFilter, "", 999, 0)
	networksCount := int64(0)
	if err == nil {
		networksCount = int64(len(networks))
	}

	// 6. Count compose_stacks
	stacksFilter := fmt.Sprintf("server = '%s'", escapeFilterValue(serverID))
	stacks, err := s.app.FindRecordsByFilter(constants.CollComposeStacks, stacksFilter, "", 999, 0)
	stacksCount := int64(0)
	if err == nil {
		stacksCount = int64(len(stacks))
	}

	// 7. Count alert_rules (both server and container targets)
	alertFilter := fmt.Sprintf("target_type = 'server' AND target_id = '%s'", escapeFilterValue(serverID))
	serverAlerts, err := s.app.FindRecordsByFilter(constants.CollAlertRules, alertFilter, "", 999, 0)
	serverAlertsCount := int64(0)
	if err == nil {
		serverAlertsCount = int64(len(serverAlerts))
	}

	// Also get container-targeted alerts
	var containerIDs []string
	for _, container := range containers {
		containerIDs = append(containerIDs, container.Id)
	}
	containerAlertsCount := int64(0)
	if len(containerIDs) > 0 {
		containerAlertFilter := fmt.Sprintf("target_type = 'container' AND target_id IN ('%s')", strings.Join(containerIDs, "','"))
		containerAlerts, err := s.app.FindRecordsByFilter(constants.CollAlertRules, containerAlertFilter, "", 999, 0)
		if err == nil {
			containerAlertsCount = int64(len(containerAlerts))
		}
	}
	totalAlertsCount := serverAlertsCount + containerAlertsCount

	// 8. Count backup_policies
	polFilter := fmt.Sprintf("volume IN ('%s')", strings.Join(getVolumeIDs(volumes), "','"))
	policies, err := s.app.FindRecordsByFilter(constants.CollBackupPolicies, polFilter, "", 999, 0)
	policiesCount := int64(0)
	if err == nil && len(volumes) > 0 {
		policiesCount = int64(len(policies))
	}

	// 9. Count backup_records for these policies
	backupRecordsCount := int64(0)
	if policiesCount > 0 {
		var policyIDs []string
		for _, policy := range policies {
			policyIDs = append(policyIDs, policy.Id)
		}
		recordFilter := fmt.Sprintf("policy IN ('%s')", strings.Join(policyIDs, "','"))
		records, err := s.app.FindRecordsByFilter(constants.CollBackupRecords, recordFilter, "", 999, 0)
		if err == nil {
			backupRecordsCount = int64(len(records))
		}
	}

	// 10. Count notification_logs (for all alerts)
	notifCount := int64(0)
	var allAlertIDs []string
	if serverAlertsCount > 0 {
		for _, alert := range serverAlerts {
			allAlertIDs = append(allAlertIDs, alert.Id)
		}
	}
	// Also add container alert IDs
	if containerAlertsCount > 0 && len(containerIDs) > 0 {
		containerAlertFilter := fmt.Sprintf("target_type = 'container' AND target_id IN ('%s')", strings.Join(containerIDs, "','"))
		containerAlerts, err := s.app.FindRecordsByFilter(constants.CollAlertRules, containerAlertFilter, "", 999, 0)
		if err == nil {
			for _, alert := range containerAlerts {
				allAlertIDs = append(allAlertIDs, alert.Id)
			}
		}
	}

	if len(allAlertIDs) > 0 {
		notifFilter := fmt.Sprintf("alert IN ('%s')", strings.Join(allAlertIDs, "','"))
		notifLogs, err := s.app.FindRecordsByFilter(constants.CollNotificationLogs, notifFilter, "", 999, 0)
		if err == nil {
			notifCount = int64(len(notifLogs))
		}
	}

	// 11. Count terminal_sessions
	var containerIDsStr []string
	for _, container := range containers {
		containerIDsStr = append(containerIDsStr, container.Id)
	}
	terminalCount := int64(0)
	if len(containerIDsStr) > 0 {
		terminalFilter := fmt.Sprintf("container IN ('%s')", strings.Join(containerIDsStr, "','"))
		terminalSessions, err := s.app.FindRecordsByFilter(constants.CollTerminalSessions, terminalFilter, "", 999, 0)
		if err == nil {
			terminalCount = int64(len(terminalSessions))
		}
	}

	// 12. Count traefik_routes
	routesFilter := fmt.Sprintf("server = '%s'", escapeFilterValue(serverID))
	routes, err := s.app.FindRecordsByFilter(constants.CollTraefikRoutes, routesFilter, "", 999, 0)
	routesCount := int64(0)
	if err == nil {
		routesCount = int64(len(routes))
	}

	// 13. Count metrics_records (server metrics)
	metricsFilter := fmt.Sprintf("server = '%s'", escapeFilterValue(serverID))
	metricsRecords, err := s.app.FindRecordsByFilter(constants.CollServerMetricsRaw, metricsFilter, "", 999, 0)
	metricsCount := int64(0)
	if err == nil {
		metricsCount = int64(len(metricsRecords))
	}

	preview := &ServerCascadePreview{
		Entity: Entity{
			ID:   serverID,
			Name: serverName,
			Type: "server",
		},
		Impact: ServerCascadeImpact{
			Containers: ResourceCount{
				Deletable: containersCount,
				Protected: 0,
			},
			Volumes: ResourceCount{
				Deletable: volumesCount,
				Protected: 0,
			},
			Images: ResourceCount{
				Deletable: imagesCount,
				Protected: 0,
			},
			Networks: ResourceCount{
				Deletable: networksCount,
				Protected: 0,
			},
			ComposeStacks: ResourceCount{
				Deletable: stacksCount,
				Protected: 0,
			},
			AlertRules: ResourceCount{
				Deletable: totalAlertsCount,
				Protected: 0,
			},
			BackupPolicies: ResourceCount{
				Deletable: policiesCount,
				Protected: 0,
			},
			BackupRecords: ResourceCount{
				Deletable: backupRecordsCount,
				Protected: 0,
			},
			NotificationLogs: ResourceCount{
				Deletable: notifCount,
				Protected: 0,
			},
			TerminalSessions: ResourceCount{
				Deletable: terminalCount,
				Protected: 0,
			},
			TraefikRoutes: ResourceCount{
				Deletable: routesCount,
				Protected: 0,
			},
			MetricsRecords: ResourceCount{
				Deletable: metricsCount,
				Protected: 0,
			},
		},
	}

	return preview, nil
}

// ── Helper Functions ────────────────────────────────

// getVolumeIDs extracts IDs from volume records
func getVolumeIDs(volumes []*core.Record) []string {
	var ids []string
	for _, v := range volumes {
		ids = append(ids, v.Id)
	}
	return ids
}

// escapeFilterValue escapes a value for use in PocketBase filter strings.
func escapeFilterValue(value string) string {
	return strings.ReplaceAll(value, "'", "\\'")
}

// findContainersUsingVolume returns container names that reference a volume.
func (s *CascadeService) findContainersUsingVolume(volumeName string, serverID string) ([]string, error) {
	filter := fmt.Sprintf("server = '%s'", escapeFilterValue(serverID))
	containerRecords, err := s.app.FindRecordsByFilter(
		constants.CollContainers,
		filter,
		"-created",
		999,
		0,
	)
	if err != nil {
		return nil, err
	}

	var containers []string
	for _, container := range containerRecords {
		volumesJSON := container.GetString("volumes")
		if volumesJSON == "" {
			continue
		}

		var mounts []MountPoint
		if err := json.Unmarshal([]byte(volumesJSON), &mounts); err != nil {
			continue
		}

		for _, mount := range mounts {
			if mount.Name == volumeName || mount.Source == volumeName {
				containers = append(containers, container.GetString("name"))
				break
			}
		}
	}

	return containers, nil
}
