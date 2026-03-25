package services

import (
	"context"
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
	app       core.App
	audit     *store.AuditStore
	volumeSvc *VolumeService
}

// NewCascadeService creates a CascadeService.
func NewCascadeService(app core.App, audit *store.AuditStore, volumeSvc *VolumeService) *CascadeService {
	return &CascadeService{
		app:       app,
		audit:     audit,
		volumeSvc: volumeSvc,
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

// ── Cascade Execution ────────────────────────────────

// ExecuteServerCascade deletes ALL database records related to a server.
// This is a DB-only operation — NO gRPC communication with any agent.
// Order: alert_rules → notification_logs → backup_records → backup_policies →
// terminal_sessions → metrics → containers → volumes → images → networks →
// compose_stacks → traefik_routes → server.
func (s *CascadeService) ExecuteServerCascade(serverID, userID, ip string) (*CascadeResult, error) {
	// 1. Validate server exists.
	serverRecord, err := s.app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		return nil, fmt.Errorf("find server: %w", err)
	}

	serverName := serverRecord.GetString("hostname")

	result := &CascadeResult{
		ServerID:           serverID,
		EntityType:         "server",
		EntityID:           serverID,
		EntityName:         serverName,
		DeletedResources:   make(map[string]int64),
		ProtectedResources: make(map[string]int64),
	}

	serverFilter := "server = {:sid}"
	serverParams := map[string]any{"sid": serverID}

	// 2. Collect container IDs for alert_rules lookup.
	containers, _ := s.app.FindRecordsByFilter(constants.CollContainers, serverFilter, "", 0, 0, serverParams)
	var containerIDs []string
	for _, c := range containers {
		containerIDs = append(containerIDs, c.Id)
	}

	// 3. Delete alert_rules (server-targeted).
	serverAlertFilter := "target_type = 'server' && target_id = {:tid}"
	serverAlertParams := map[string]any{"tid": serverID}
	serverAlerts, _ := s.app.FindRecordsByFilter(constants.CollAlertRules, serverAlertFilter, "", 0, 0, serverAlertParams)

	// Collect all alert IDs for notification_logs cleanup.
	var allAlertIDs []string
	for _, a := range serverAlerts {
		allAlertIDs = append(allAlertIDs, a.Id)
	}

	// Delete container-targeted alert_rules.
	for _, cID := range containerIDs {
		containerAlertFilter := "target_type = 'container' && target_id = {:tid}"
		containerAlertParams := map[string]any{"tid": cID}
		containerAlerts, err := s.app.FindRecordsByFilter(constants.CollAlertRules, containerAlertFilter, "", 0, 0, containerAlertParams)
		if err == nil {
			for _, a := range containerAlerts {
				allAlertIDs = append(allAlertIDs, a.Id)
			}
		}
	}

	// Delete notification_logs for all collected alerts.
	for _, alertID := range allAlertIDs {
		notifFilter := "alert = {:aid}"
		notifParams := map[string]any{"aid": alertID}
		result.DeletedResources["notification_logs"] += s.deleteByFilter(constants.CollNotificationLogs, notifFilter, notifParams)
	}

	// Delete all collected alert_rules.
	for _, alertID := range allAlertIDs {
		alertRecord, err := s.app.FindRecordById(constants.CollAlertRules, alertID)
		if err != nil {
			continue
		}
		if err := s.app.Delete(alertRecord); err != nil {
			slog.Warn("cascade delete alert_rule failed", "id", alertID, "error", err)
		} else {
			result.DeletedResources["alert_rules"]++
		}
	}

	// 4. Collect volume IDs for backup cleanup.
	volumes, _ := s.app.FindRecordsByFilter(constants.CollVolumes, serverFilter, "", 0, 0, serverParams)
	var volumeIDs []string
	for _, v := range volumes {
		volumeIDs = append(volumeIDs, v.Id)
	}

	// 5. Delete backup_records and backup_policies for each volume.
	for _, volID := range volumeIDs {
		polFilter := "volume = {:vid}"
		polParams := map[string]any{"vid": volID}
		policies, err := s.app.FindRecordsByFilter(constants.CollBackupPolicies, polFilter, "", 0, 0, polParams)
		if err != nil {
			continue
		}
		for _, policy := range policies {
			recFilter := "policy = {:pid}"
			recParams := map[string]any{"pid": policy.Id}
			result.DeletedResources["backup_records"] += s.deleteByFilter(constants.CollBackupRecords, recFilter, recParams)
		}
		result.DeletedResources["backup_policies"] += s.deleteByFilter(constants.CollBackupPolicies, polFilter, polParams)
	}

	// 6. Delete terminal_sessions for all containers.
	for _, cID := range containerIDs {
		termFilter := "container = {:cid}"
		termParams := map[string]any{"cid": cID}
		result.DeletedResources["terminal_sessions"] += s.deleteByFilter(constants.CollTerminalSessions, termFilter, termParams)
	}

	// 7. Delete all metrics (container + server).
	metricsCollections := []struct {
		name string
		key  string
	}{
		{constants.CollContainerMetricsRaw, "container_metrics_raw"},
		{constants.CollContainerMetricsHourly, "container_metrics_hourly"},
		{constants.CollContainerMetricsDaily, "container_metrics_daily"},
		{constants.CollServerMetricsRaw, "server_metrics_raw"},
		{constants.CollServerMetricsHourly, "server_metrics_hourly"},
		{constants.CollServerMetricsDaily, "server_metrics_daily"},
	}

	totalMetrics := int64(0)
	for _, mc := range metricsCollections {
		count := s.deleteByFilter(mc.name, serverFilter, serverParams)
		totalMetrics += count
	}
	result.DeletedResources["metrics_records"] = totalMetrics

	// 8. Delete resource records (containers, volumes, images, networks, compose_stacks, traefik_routes).
	resourceCollections := []struct {
		name string
		key  string
	}{
		{constants.CollContainers, "containers"},
		{constants.CollVolumes, "volumes"},
		{constants.CollImages, "images"},
		{constants.CollNetworks, "networks"},
		{constants.CollComposeStacks, "compose_stacks"},
		{constants.CollTraefikRoutes, "traefik_routes"},
	}

	for _, rc := range resourceCollections {
		result.DeletedResources[rc.key] = s.deleteByFilter(rc.name, serverFilter, serverParams)
	}

	// 9. Delete the server record itself (last).
	if err := s.app.Delete(serverRecord); err != nil {
		return result, fmt.Errorf("delete server record: %w", err)
	}

	// 10. Create audit log entry.
	s.audit.Log("server.cascade_delete", userID, "server", serverID, ip, map[string]any{
		"server_name":        serverName,
		"deleted_resources":  result.DeletedResources,
	})

	return result, nil
}

// ExecuteContainerCascade handles optional cleanup when a container is removed.
// Called AFTER the container is successfully removed from the Docker host.
// Handles: alert cleanup, exclusive volume removal (Docker host + DB), backup cleanup.
func (s *CascadeService) ExecuteContainerCascade(
	ctx context.Context,
	containerID, serverID string,
	opts ContainerCascadeOptions,
	userID, ip string,
) (*CascadeResult, error) {
	// 1. Fetch container record for metadata.
	containerRecord, err := s.app.FindRecordById(constants.CollContainers, containerID)
	if err != nil {
		return nil, fmt.Errorf("find container: %w", err)
	}

	containerName := containerRecord.GetString("name")

	result := &CascadeResult{
		ServerID:           serverID,
		EntityType:         "container",
		EntityID:           containerID,
		EntityName:         containerName,
		DeletedResources:   make(map[string]int64),
		ProtectedResources: make(map[string]int64),
		Options: map[string]bool{
			"remove_alerts":  opts.RemoveAlerts,
			"remove_volumes": opts.RemoveVolumes,
			"remove_backups": opts.RemoveBackups,
		},
	}

	// 2. If remove_alerts: delete alert_rules for this container and their notification_logs.
	if opts.RemoveAlerts {
		alertFilter := "target_type = 'container' && target_id = {:cid}"
		alertParams := map[string]any{"cid": containerID}
		alertRules, err := s.app.FindRecordsByFilter(constants.CollAlertRules, alertFilter, "", 0, 0, alertParams)
		if err == nil {
			// Delete notification_logs for each alert before deleting the alert.
			for _, alert := range alertRules {
				notifFilter := "alert = {:aid}"
				notifParams := map[string]any{"aid": alert.Id}
				result.DeletedResources["notification_logs"] += s.deleteByFilter(constants.CollNotificationLogs, notifFilter, notifParams)
			}
			// Delete the alert rules.
			for _, alert := range alertRules {
				if err := s.app.Delete(alert); err != nil {
					slog.Warn("cascade delete alert_rule failed", "id", alert.Id, "error", err)
					continue
				}
				result.DeletedResources["alert_rules"]++
			}
		}
	}

	// 3. If remove_volumes: parse volumes JSON, check exclusivity, remove exclusive ones.
	var removedVolumeIDs []string
	if opts.RemoveVolumes {
		volumesJSON := containerRecord.GetString("volumes")
		var mounts []MountPoint
		if volumesJSON != "" {
			if err := json.Unmarshal([]byte(volumesJSON), &mounts); err != nil {
				slog.Warn("failed to parse volumes JSON for cascade", "container_id", containerID, "error", err)
			}
		}

		for _, mount := range mounts {
			volumeName := mount.Name
			if volumeName == "" {
				volumeName = mount.Source
			}
			if volumeName == "" {
				continue
			}

			// Find volume PB record.
			volFilter := "name = {:vname} && server = {:sid}"
			volParams := map[string]any{"vname": volumeName, "sid": serverID}
			volRecords, err := s.app.FindRecordsByFilter(constants.CollVolumes, volFilter, "", 1, 0, volParams)
			if err != nil || len(volRecords) == 0 {
				continue
			}

			volRecord := volRecords[0]

			// Check exclusivity — exclude the container being removed.
			isExclusive, otherContainers, err := s.IsVolumeExclusive(volRecord.Id, []string{containerID})
			if err != nil {
				slog.Warn("cascade volume exclusivity check failed", "volume", volumeName, "error", err)
				continue
			}

			if !isExclusive {
				slog.Info("cascade skip shared volume",
					"volume", volumeName,
					"shared_with", otherContainers,
					"container", containerName,
				)
				result.ProtectedResources["volumes"]++
				continue
			}

			// Remove volume from Docker host via agent gRPC.
			if s.volumeSvc != nil {
				_, removeErr := s.volumeSvc.RemoveVolume(ctx, volRecord.Id, true, userID, ip)
				if removeErr != nil {
					slog.Warn("cascade remove volume from host failed",
						"volume", volumeName,
						"error", removeErr,
					)
					// Even if Docker removal fails, try to clean up DB record.
				}
			}

			// Delete volume PB record (if not already deleted by VolumeService.RemoveVolume).
			if rec, findErr := s.app.FindRecordById(constants.CollVolumes, volRecord.Id); findErr == nil {
				if delErr := s.app.Delete(rec); delErr != nil {
					slog.Warn("cascade delete volume record failed", "volume_id", volRecord.Id, "error", delErr)
				} else {
					result.DeletedResources["volumes"]++
					removedVolumeIDs = append(removedVolumeIDs, volRecord.Id)
				}
			} else {
				// Already deleted by VolumeService.RemoveVolume.
				result.DeletedResources["volumes"]++
				removedVolumeIDs = append(removedVolumeIDs, volRecord.Id)
			}
		}
	}

	// 4. If remove_backups: delete backup_policies and backup_records for removed volumes.
	if opts.RemoveBackups && len(removedVolumeIDs) > 0 {
		for _, volID := range removedVolumeIDs {
			polFilter := "volume = {:vid}"
			polParams := map[string]any{"vid": volID}
			policies, err := s.app.FindRecordsByFilter(constants.CollBackupPolicies, polFilter, "", 0, 0, polParams)
			if err != nil {
				continue
			}
			for _, policy := range policies {
				recFilter := "policy = {:pid}"
				recParams := map[string]any{"pid": policy.Id}
				result.DeletedResources["backup_records"] += s.deleteByFilter(constants.CollBackupRecords, recFilter, recParams)
			}
			result.DeletedResources["backup_policies"] += s.deleteByFilter(constants.CollBackupPolicies, polFilter, polParams)
		}
	}

	// 5. If remove_notification_logs: delete notification_logs not yet covered by alert removal.
	if opts.RemoveNotificationLogs && !opts.RemoveAlerts {
		// If alerts were not removed, still clean up notification logs for this container's alerts.
		alertFilter := "target_type = 'container' && target_id = {:cid}"
		alertParams := map[string]any{"cid": containerID}
		alertRules, err := s.app.FindRecordsByFilter(constants.CollAlertRules, alertFilter, "", 0, 0, alertParams)
		if err == nil {
			for _, alert := range alertRules {
				notifFilter := "alert = {:aid}"
				notifParams := map[string]any{"aid": alert.Id}
				result.DeletedResources["notification_logs"] += s.deleteByFilter(constants.CollNotificationLogs, notifFilter, notifParams)
			}
		}
	}

	// 6. Delete terminal_sessions for this container.
	termFilter := "container = {:cid}"
	termParams := map[string]any{"cid": containerID}
	result.DeletedResources["terminal_sessions"] = s.deleteByFilter(constants.CollTerminalSessions, termFilter, termParams)

	// 7. Delete metrics records for this container.
	metricsCollections := []string{
		constants.CollContainerMetricsRaw,
		constants.CollContainerMetricsHourly,
		constants.CollContainerMetricsDaily,
	}
	totalMetrics := int64(0)
	for _, mc := range metricsCollections {
		containerFilter := "container = {:cid}"
		containerParams := map[string]any{"cid": containerID}
		totalMetrics += s.deleteByFilter(mc, containerFilter, containerParams)
	}
	result.DeletedResources["metrics_records"] = totalMetrics

	// 8. Create audit log entry.
	s.audit.Log("container.cascade_delete", userID, "container", containerID, ip, map[string]any{
		"container_name":     containerName,
		"server_id":          serverID,
		"options":            result.Options,
		"deleted_resources":  result.DeletedResources,
		"protected_resources": result.ProtectedResources,
	})

	return result, nil
}

// ExecuteVolumeCascade handles optional backup record preservation when a volume is removed.
// Called BEFORE the volume PB record is deleted so that backup_records can be detached.
func (s *CascadeService) ExecuteVolumeCascade(volumeID string, opts VolumeCascadeOptions, userID, ip string) (*CascadeResult, error) {
	// 1. Fetch volume record for metadata.
	volumeRecord, err := s.app.FindRecordById(constants.CollVolumes, volumeID)
	if err != nil {
		return nil, fmt.Errorf("find volume: %w", err)
	}

	volumeName := volumeRecord.GetString("name")
	serverID := volumeRecord.GetString("server")

	result := &CascadeResult{
		ServerID:           serverID,
		EntityType:         "volume",
		EntityID:           volumeID,
		EntityName:         volumeName,
		DeletedResources:   make(map[string]int64),
		ProtectedResources: make(map[string]int64),
		Options: map[string]bool{
			"keep_backup_records": opts.KeepBackupRecords,
		},
	}

	// 2. Find backup_policies for this volume.
	polFilter := "volume = {:vid}"
	polParams := map[string]any{"vid": volumeID}
	policies, err := s.app.FindRecordsByFilter(constants.CollBackupPolicies, polFilter, "", 0, 0, polParams)
	if err != nil {
		policies = nil
	}

	// 3. Handle backup_records for each policy.
	for _, policy := range policies {
		recFilter := "policy = {:pid}"
		recParams := map[string]any{"pid": policy.Id}
		records, err := s.app.FindRecordsByFilter(constants.CollBackupRecords, recFilter, "", 0, 0, recParams)
		if err != nil {
			continue
		}

		if opts.KeepBackupRecords {
			// Detach records from policy so PB cascade won't delete them.
			for _, rec := range records {
				rec.Set("policy", "")
				if saveErr := s.app.Save(rec); saveErr != nil {
					slog.Warn("cascade detach backup_record failed", "record_id", rec.Id, "error", saveErr)
				} else {
					result.ProtectedResources["backup_records"]++
				}
			}
		} else {
			// Delete backup_records normally.
			for _, rec := range records {
				if delErr := s.app.Delete(rec); delErr != nil {
					slog.Warn("cascade delete backup_record failed", "record_id", rec.Id, "error", delErr)
				} else {
					result.DeletedResources["backup_records"]++
				}
			}
		}
	}

	// 4. Delete backup_policies.
	result.DeletedResources["backup_policies"] = s.deleteByFilter(constants.CollBackupPolicies, polFilter, polParams)

	// 5. Create audit log entry.
	s.audit.Log("volume.cascade_delete", userID, "volume", volumeID, ip, map[string]any{
		"volume_name":         volumeName,
		"server_id":           serverID,
		"options":             result.Options,
		"deleted_resources":   result.DeletedResources,
		"protected_resources": result.ProtectedResources,
	})

	return result, nil
}

// deleteByFilter finds and deletes all records matching a filter.
// Returns the count of successfully deleted records. Logs warnings on failure.
func (s *CascadeService) deleteByFilter(collection, filter string, params map[string]any) int64 {
	records, err := s.app.FindRecordsByFilter(collection, filter, "", 0, 0, params)
	if err != nil {
		return 0
	}

	deleted := int64(0)
	for _, r := range records {
		if err := s.app.Delete(r); err != nil {
			slog.Warn("cascade delete failed", "collection", collection, "record", r.Id, "error", err)
			continue
		}
		deleted++
	}
	return deleted
}

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
	alertFilter := fmt.Sprintf("target_type = 'container' && target_id = '%s'", escapeFilterValue(containerID))
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
		volFilter := fmt.Sprintf("name = '%s' && server = '%s'", escapeFilterValue(volumeName), escapeFilterValue(serverID))
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
	alertFilter := fmt.Sprintf("target_type = 'server' && target_id = '%s'", escapeFilterValue(serverID))
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
		containerAlertFilter := fmt.Sprintf("target_type = 'container' && target_id IN ('%s')", strings.Join(containerIDs, "','"))
		containerAlerts, err := s.app.FindRecordsByFilter(constants.CollAlertRules, containerAlertFilter, "", 999, 0)
		if err == nil {
			containerAlertsCount = int64(len(containerAlerts))
		}
	}
	totalAlertsCount := serverAlertsCount + containerAlertsCount

	// 8. Count backup_policies
	polFilter := fmt.Sprintf("volume IN ('%s')", strings.Join(getRecordIDs(volumes), "','"))
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
		containerAlertFilter := fmt.Sprintf("target_type = 'container' && target_id IN ('%s')", strings.Join(containerIDs, "','"))
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

// getRecordIDs extracts IDs from a slice of records.
func getRecordIDs(records []*core.Record) []string {
	ids := make([]string, 0, len(records))
	for _, r := range records {
		ids = append(ids, r.Id)
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
