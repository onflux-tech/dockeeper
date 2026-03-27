package store

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/events"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// ResourceStore handles upsert and sync operations for containers, images,
// networks and volumes in PocketBase.
type ResourceStore struct {
	app       core.App
	bus       *events.Bus
	debounce  sync.Map // key: serverID+containerID → time.Time
	debounceTTL time.Duration
}

// NewResourceStore creates a new ResourceStore.
func NewResourceStore(app core.App, bus *events.Bus) *ResourceStore {
	return &ResourceStore{
		app:         app,
		bus:         bus,
		debounceTTL: 5 * time.Second,
	}
}

// UpsertContainers creates or updates container records and marks containers
// that are no longer reported as "removed".
func (s *ResourceStore) UpsertContainers(serverID string, containers []*agentpb.ContainerInfo) (int, error) {
	col, err := s.app.FindCachedCollectionByNameOrId(constants.CollContainers)
	if err != nil {
		return 0, fmt.Errorf("find collection %s: %w", constants.CollContainers, err)
	}

	// Resolve server name for events.
	serverName := s.resolveServerName(serverID)

	reported := make(map[string]bool, len(containers))
	count := 0

	for _, c := range containers {
		reported[c.GetContainerId()] = true

		rec, err := s.findContainer(serverID, c.GetContainerId())
		isNew := false
		if err != nil {
			// Fallback: look up by server + name to handle container recreate
			// (same name, new Docker ID after update/recreate operations).
			rec, err = s.findContainerByName(serverID, c.GetName())
			if err != nil {
				rec = core.NewRecord(col)
				rec.Set("server", serverID)
				isNew = true
			}
			rec.Set("container_id", c.GetContainerId())
		}

		// Capture old status before update for state diff detection.
		oldStatus := rec.GetString("status")
		newStatus := c.GetStatus()

		rec.Set("name", c.GetName())
		rec.Set("image", c.GetImage())
		rec.Set("status", newStatus)
		rec.Set("ports", parseJSONField(c.GetPortsJson()))
		rec.Set("labels", parseJSONField(c.GetLabelsJson()))
		rec.Set("volumes", parseJSONField(c.GetVolumesJson()))
		rec.Set("networks", parseJSONField(c.GetNetworksJson()))
		rec.Set("restart_policy", c.GetRestartPolicy())
		rec.Set("command", c.GetCommand())
		rec.Set("entrypoint", c.GetEntrypoint())

		if c.GetResourceLimits() != nil {
			limits := map[string]any{
				"cpu_limit":  c.GetResourceLimits().GetCpuLimit(),
				"mem_limit":  c.GetResourceLimits().GetMemLimit(),
				"pids_limit": c.GetResourceLimits().GetPidsLimit(),
			}
			rec.Set("resource_limits", limits)
		}

		if c.GetCreatedAt() != nil {
			rec.Set("container_created_at", c.GetCreatedAt().AsTime().UTC())
		}
		if c.GetStartedAt() != nil {
			rec.Set("started_at", c.GetStartedAt().AsTime().UTC())
		}

		if err := s.app.Save(rec); err != nil {
			slog.Error("upsert container failed", "container_id", c.GetContainerId(), "error", err)
			continue
		}
		count++

		// Link to compose stack if container has compose labels.
		s.linkComposeStack(serverID, rec, c.GetLabelsJson())

		// Publish state transition events.
		if s.bus != nil {
			if isNew {
				s.publishContainerEvent(events.ContainerCreated, serverID, serverName, rec.Id, c.GetName(), nil)
			} else if oldStatus != newStatus && oldStatus != "" {
				s.publishStateTransition(oldStatus, newStatus, serverID, serverName, rec.Id, c.GetName())
			}
		}
	}

	// Mark containers not in report as "removed".
	s.markRemovedContainers(serverID, serverName, reported)

	return count, nil
}

// publishStateTransition determines the event type for a container state change.
func (s *ResourceStore) publishStateTransition(oldStatus, newStatus, serverID, serverName, recordID, name string) {
	var eventType events.EventType
	var details map[string]any

	switch {
	case newStatus == "dead":
		// "dead" status indicates a crash or abnormal exit.
		eventType = events.ContainerCrashed
		details = map[string]any{"old_status": oldStatus}
	case newStatus == "exited":
		eventType = events.ContainerStopped
		details = map[string]any{"old_status": oldStatus}
	case newStatus == "running" && (oldStatus == "exited" || oldStatus == "created" || oldStatus == "dead"):
		eventType = events.ContainerStarted
		details = map[string]any{"old_status": oldStatus}
	case newStatus == "restarting":
		eventType = events.ContainerRestarting
		details = map[string]any{"old_status": oldStatus}
	default:
		return // no relevant transition
	}

	s.publishContainerEvent(eventType, serverID, serverName, recordID, name, details)
}

// publishContainerEvent publishes a container event with debounce.
func (s *ResourceStore) publishContainerEvent(t events.EventType, serverID, serverName, containerID, containerName string, details map[string]any) {
	key := fmt.Sprintf("%s:%s:%s", serverID, containerID, string(t))
	if last, ok := s.debounce.Load(key); ok {
		if time.Since(last.(time.Time)) < s.debounceTTL {
			return // debounced
		}
	}
	s.debounce.Store(key, time.Now())

	s.bus.Publish(events.Event{
		Type:          t,
		Severity:      events.SeverityFor(t),
		ServerID:      serverID,
		ServerName:    serverName,
		ContainerID:   containerID,
		ContainerName: containerName,
		Details:       details,
		Timestamp:     time.Now().UTC(),
	})
}

// CleanupDebounce removes stale debounce entries. Call periodically.
func (s *ResourceStore) CleanupDebounce() {
	s.debounce.Range(func(key, value any) bool {
		if time.Since(value.(time.Time)) > 5*time.Minute {
			s.debounce.Delete(key)
		}
		return true
	})
}

// resolveServerName looks up the server hostname.
func (s *ResourceStore) resolveServerName(serverID string) string {
	rec, err := s.app.FindRecordById(constants.CollServers, serverID)
	if err != nil {
		return ""
	}
	return rec.GetString("hostname")
}

// UpsertImages creates or updates image records for a server.
func (s *ResourceStore) UpsertImages(serverID string, images []*agentpb.ImageInfo) (int, error) {
	col, err := s.app.FindCachedCollectionByNameOrId(constants.CollImages)
	if err != nil {
		return 0, fmt.Errorf("find collection %s: %w", constants.CollImages, err)
	}

	count := 0
	for _, img := range images {
		rec, err := s.findByKey(constants.CollImages, serverID, "image_id", img.GetImageId())
		if err != nil {
			rec = core.NewRecord(col)
			rec.Set("server", serverID)
			rec.Set("image_id", img.GetImageId())
		}

		rec.Set("tags", img.GetTags())
		rec.Set("size", img.GetSize())
		rec.Set("in_use", img.GetInUse())

		if img.GetCreatedAt() != nil {
			rec.Set("image_created_at", img.GetCreatedAt().AsTime().UTC())
		}

		if err := s.app.Save(rec); err != nil {
			slog.Error("upsert image failed", "image_id", img.GetImageId(), "error", err)
			continue
		}
		count++
	}
	return count, nil
}

// UpsertNetworks creates or updates network records for a server.
func (s *ResourceStore) UpsertNetworks(serverID string, networks []*agentpb.NetworkInfo) (int, error) {
	col, err := s.app.FindCachedCollectionByNameOrId(constants.CollNetworks)
	if err != nil {
		return 0, fmt.Errorf("find collection %s: %w", constants.CollNetworks, err)
	}

	count := 0
	for _, n := range networks {
		rec, err := s.findByKey(constants.CollNetworks, serverID, "network_id", n.GetNetworkId())
		if err != nil {
			rec = core.NewRecord(col)
			rec.Set("server", serverID)
			rec.Set("network_id", n.GetNetworkId())
		}

		rec.Set("name", n.GetName())
		rec.Set("driver", n.GetDriver())
		rec.Set("scope", n.GetScope())
		rec.Set("subnet", n.GetSubnet())
		rec.Set("gateway", n.GetGateway())
		rec.Set("internal", n.GetInternal())
		rec.Set("labels", parseJSONField(n.GetLabelsJson()))

		if err := s.app.Save(rec); err != nil {
			slog.Error("upsert network failed", "network_id", n.GetNetworkId(), "error", err)
			continue
		}
		count++
	}
	return count, nil
}

// UpsertVolumes creates or updates volume records for a server.
func (s *ResourceStore) UpsertVolumes(serverID string, volumes []*agentpb.VolumeInfo) (int, error) {
	col, err := s.app.FindCachedCollectionByNameOrId(constants.CollVolumes)
	if err != nil {
		return 0, fmt.Errorf("find collection %s: %w", constants.CollVolumes, err)
	}

	count := 0
	for _, v := range volumes {
		rec, err := s.findByKey(constants.CollVolumes, serverID, "name", v.GetName())
		if err != nil {
			rec = core.NewRecord(col)
			rec.Set("server", serverID)
		}

		rec.Set("name", v.GetName())
		rec.Set("driver", v.GetDriver())
		rec.Set("mountpoint", v.GetMountpoint())
		rec.Set("size", v.GetSize())
		rec.Set("labels", parseJSONField(v.GetLabelsJson()))
		rec.Set("in_use", v.GetInUse())

		if err := s.app.Save(rec); err != nil {
			slog.Error("upsert volume failed", "name", v.GetName(), "error", err)
			continue
		}
		count++
	}
	return count, nil
}

// findContainer looks up a container record by server and container_id.
func (s *ResourceStore) findContainer(serverID, containerID string) (*core.Record, error) {
	return s.findByKey(constants.CollContainers, serverID, "container_id", containerID)
}

// findContainerByName looks up a container record by server and name.
func (s *ResourceStore) findContainerByName(serverID, name string) (*core.Record, error) {
	return s.findByKey(constants.CollContainers, serverID, "name", name)
}

// findByKey looks up a record by server relation and a unique key field.
func (s *ResourceStore) findByKey(collection, serverID, keyField, keyValue string) (*core.Record, error) {
	filter := fmt.Sprintf("server = {:sid} && %s = {:val}", keyField)
	records, err := s.app.FindRecordsByFilter(
		collection, filter, "", 1, 0,
		map[string]any{"sid": serverID, "val": keyValue},
	)
	if err != nil || len(records) == 0 {
		return nil, fmt.Errorf("record not found")
	}
	return records[0], nil
}

// markRemovedContainers deletes container records not present in the reported set.
func (s *ResourceStore) markRemovedContainers(serverID, serverName string, reported map[string]bool) {
	records, err := s.app.FindRecordsByFilter(
		constants.CollContainers,
		"server = {:sid}",
		"", 0, 0,
		map[string]any{"sid": serverID},
	)
	if err != nil {
		slog.Error("failed to list containers for removal check", "server_id", serverID, "error", err)
		return
	}

	for _, r := range records {
		cid := r.GetString("container_id")
		if !reported[cid] {
			containerName := r.GetString("name")
			recordID := r.Id
			if err := s.app.Delete(r); err != nil {
				slog.Error("failed to delete removed container", "container_id", cid, "error", err)
			} else {
				slog.Info("container deleted (no longer on host)", "server_id", serverID, "container_id", cid)
				if s.bus != nil {
					s.publishContainerEvent(events.ContainerRemoved, serverID, serverName, recordID, containerName, nil)
				}
			}
		}
	}
}

// parseJSONField parses a JSON string into a generic value suitable for PB JSON fields.
// Returns nil on empty/invalid input so PB stores null.
func parseJSONField(raw string) any {
	if raw == "" || raw == "{}" || raw == "[]" || raw == "null" {
		return nil
	}
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		return nil
	}
	return v
}

// linkComposeStack extracts compose labels and links the container to its compose stack.
func (s *ResourceStore) linkComposeStack(serverID string, rec *core.Record, labelsJSON string) {
	if labelsJSON == "" {
		return
	}

	var labels map[string]string
	if err := json.Unmarshal([]byte(labelsJSON), &labels); err != nil {
		return
	}

	project := labels["com.docker.compose.project"]
	service := labels["com.docker.compose.service"]

	if project == "" {
		return
	}

	changed := false

	if rec.GetString("compose_project") != project {
		rec.Set("compose_project", project)
		changed = true
	}
	if service != "" && rec.GetString("compose_service") != service {
		rec.Set("compose_service", service)
		changed = true
	}

	// Look up compose_stacks for this server + project name.
	stacks, err := s.app.FindRecordsByFilter(
		constants.CollComposeStacks,
		"server = {:sid} && name = {:name}",
		"", 1, 0,
		map[string]any{"sid": serverID, "name": project},
	)
	if err == nil && len(stacks) > 0 {
		if rec.GetString("compose_stack") != stacks[0].Id {
			rec.Set("compose_stack", stacks[0].Id)
			changed = true
		}
	}

	if changed {
		if err := s.app.Save(rec); err != nil {
			slog.Error("link compose stack failed", "container", rec.Id, "error", err)
		}
	}
}
