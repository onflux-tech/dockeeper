package store

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/shared/constants"
	agentpb "github.com/onflux-tech/dockeeper/proto/gen/dockeeper/agent/v1"
)

// ResourceStore handles upsert and sync operations for containers, images,
// networks and volumes in PocketBase.
type ResourceStore struct {
	app core.App
}

// NewResourceStore creates a new ResourceStore.
func NewResourceStore(app core.App) *ResourceStore {
	return &ResourceStore{app: app}
}

// UpsertContainers creates or updates container records and marks containers
// that are no longer reported as "removed".
func (s *ResourceStore) UpsertContainers(serverID string, containers []*agentpb.ContainerInfo) (int, error) {
	col, err := s.app.FindCachedCollectionByNameOrId(constants.CollContainers)
	if err != nil {
		return 0, fmt.Errorf("find collection %s: %w", constants.CollContainers, err)
	}

	reported := make(map[string]bool, len(containers))
	count := 0

	for _, c := range containers {
		reported[c.GetContainerId()] = true

		rec, err := s.findContainer(serverID, c.GetContainerId())
		if err != nil {
			// Fallback: look up by server + name to handle container recreate
			// (same name, new Docker ID after update/recreate operations).
			rec, err = s.findContainerByName(serverID, c.GetName())
			if err != nil {
				rec = core.NewRecord(col)
				rec.Set("server", serverID)
			}
			rec.Set("container_id", c.GetContainerId())
		}

		rec.Set("name", c.GetName())
		rec.Set("image", c.GetImage())
		rec.Set("status", c.GetStatus())
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
	}

	// Mark containers not in report as "removed".
	s.markRemovedContainers(serverID, reported)

	return count, nil
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
func (s *ResourceStore) markRemovedContainers(serverID string, reported map[string]bool) {
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
			if err := s.app.Delete(r); err != nil {
				slog.Error("failed to delete removed container", "container_id", cid, "error", err)
			} else {
				slog.Info("container deleted (no longer on host)", "server_id", serverID, "container_id", cid)
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
