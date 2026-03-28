package services

import (
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/store"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// TagService manages tag operations with denormalization sync.
type TagService struct {
	app   core.App
	store *store.TagStore
}

// NewTagService creates a new TagService.
func NewTagService(app core.App, tagStore *store.TagStore) *TagService {
	return &TagService{app: app, store: tagStore}
}

// ApplyTag adds a tag to a resource and syncs the denormalized JSON field.
func (s *TagService) ApplyTag(name, color, resourceType, resourceID string) (*core.Record, error) {
	rec, err := s.store.ApplyTag(name, color, resourceType, resourceID)
	if err != nil {
		return nil, err
	}
	s.syncDenormalizedTags(resourceType, resourceID)
	return rec, nil
}

// RemoveTag removes a tag by ID and syncs the denormalized JSON field.
func (s *TagService) RemoveTag(id string) error {
	// Get tag info before deletion for denorm sync.
	tagRec, err := s.app.FindRecordById(constants.CollTags, id)
	if err != nil {
		return err
	}
	rt := tagRec.GetString("resource_type")
	rid := tagRec.GetString("resource_id")

	if err := s.store.RemoveTag(id); err != nil {
		return err
	}
	s.syncDenormalizedTags(rt, rid)
	return nil
}

// BulkApply applies a tag to multiple resources at once.
func (s *TagService) BulkApply(name, color, resourceType string, resourceIDs []string) (int, error) {
	applied := 0
	for _, rid := range resourceIDs {
		if _, err := s.store.ApplyTag(name, color, resourceType, rid); err != nil {
			slog.Warn("bulk tag apply failed for resource", "name", name, "resource_id", rid, "error", err)
			continue
		}
		s.syncDenormalizedTags(resourceType, rid)
		applied++
	}
	return applied, nil
}

// BulkRemove removes a tag from multiple resources at once.
func (s *TagService) BulkRemove(name, resourceType string, resourceIDs []string) (int, error) {
	removed := 0
	for _, rid := range resourceIDs {
		if err := s.store.RemoveTagByName(name, resourceType, rid); err != nil {
			continue
		}
		s.syncDenormalizedTags(resourceType, rid)
		removed++
	}
	return removed, nil
}

// GetTagsForResource returns tags for a specific resource.
func (s *TagService) GetTagsForResource(resourceType, resourceID string) ([]*core.Record, error) {
	return s.store.GetTagsForResource(resourceType, resourceID)
}

// GetResourcesByTag returns resource IDs matching a tag.
func (s *TagService) GetResourcesByTag(name, resourceType string) ([]string, error) {
	return s.store.GetResourcesByTag(name, resourceType)
}

// GetUniqueTagNames returns distinct tag names with counts.
func (s *TagService) GetUniqueTagNames(resourceType string) ([]store.TagNameCount, error) {
	return s.store.GetUniqueTagNames(resourceType)
}

// GetAllTags returns all tag records.
func (s *TagService) GetAllTags(resourceType string) ([]*core.Record, error) {
	return s.store.GetAllTags(resourceType)
}

// syncDenormalizedTags updates the tags JSON field on the resource record.
func (s *TagService) syncDenormalizedTags(resourceType, resourceID string) {
	collName := constants.CollContainers
	if resourceType == "server" {
		collName = constants.CollServers
	}

	rec, err := s.app.FindRecordById(collName, resourceID)
	if err != nil {
		slog.Warn("failed to find resource for tag sync", "type", resourceType, "id", resourceID, "error", err)
		return
	}

	tagRecords, err := s.store.GetTagsForResource(resourceType, resourceID)
	if err != nil {
		slog.Warn("failed to get tags for sync", "type", resourceType, "id", resourceID, "error", err)
		return
	}

	names := make([]string, 0, len(tagRecords))
	for _, t := range tagRecords {
		names = append(names, strings.ToLower(t.GetString("name")))
	}

	tagsJSON, _ := json.Marshal(names)
	rec.Set("tags", string(tagsJSON))
	if err := s.app.Save(rec); err != nil {
		slog.Warn("failed to sync denormalized tags", "type", resourceType, "id", resourceID, "error", err)
	}
}
