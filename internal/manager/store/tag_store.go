package store

import (
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// TagStore handles persistence for tag records.
type TagStore struct {
	app core.App
}

// NewTagStore creates a new TagStore.
func NewTagStore(app core.App) *TagStore {
	return &TagStore{app: app}
}

// ApplyTag creates a tag assignment. Name is case-insensitive (stored lowercase).
func (s *TagStore) ApplyTag(name, color, resourceType, resourceID string) (*core.Record, error) {
	name = strings.ToLower(strings.TrimSpace(name))

	// Check for existing tag assignment.
	existing, err := s.app.FindFirstRecordByFilter(
		constants.CollTags,
		"name = {:name} && resource_type = {:rt} && resource_id = {:rid}",
		dbx.Params{"name": name, "rt": resourceType, "rid": resourceID},
	)
	if err == nil && existing != nil {
		// Already exists — update color if provided.
		if color != "" {
			existing.Set("color", color)
			if err := s.app.Save(existing); err != nil {
				return nil, err
			}
		}
		return existing, nil
	}

	col, err := s.app.FindCachedCollectionByNameOrId(constants.CollTags)
	if err != nil {
		return nil, err
	}
	rec := core.NewRecord(col)
	rec.Set("name", name)
	rec.Set("color", color)
	rec.Set("resource_type", resourceType)
	rec.Set("resource_id", resourceID)
	if err := s.app.Save(rec); err != nil {
		return nil, err
	}
	return rec, nil
}

// RemoveTag deletes a tag assignment by its PocketBase ID.
func (s *TagStore) RemoveTag(id string) error {
	rec, err := s.app.FindRecordById(constants.CollTags, id)
	if err != nil {
		return err
	}
	return s.app.Delete(rec)
}

// RemoveTagByName removes a specific tag from a specific resource.
func (s *TagStore) RemoveTagByName(name, resourceType, resourceID string) error {
	name = strings.ToLower(strings.TrimSpace(name))
	rec, err := s.app.FindFirstRecordByFilter(
		constants.CollTags,
		"name = {:name} && resource_type = {:rt} && resource_id = {:rid}",
		dbx.Params{"name": name, "rt": resourceType, "rid": resourceID},
	)
	if err != nil {
		return err
	}
	return s.app.Delete(rec)
}

// GetTagsForResource returns all tags assigned to a specific resource.
func (s *TagStore) GetTagsForResource(resourceType, resourceID string) ([]*core.Record, error) {
	return s.app.FindRecordsByFilter(
		constants.CollTags,
		"resource_type = {:rt} && resource_id = {:rid}",
		"name",
		0, 0,
		dbx.Params{"rt": resourceType, "rid": resourceID},
	)
}

// GetResourcesByTag returns resource IDs that have a given tag name.
func (s *TagStore) GetResourcesByTag(name, resourceType string) ([]string, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	records, err := s.app.FindRecordsByFilter(
		constants.CollTags,
		"name = {:name} && resource_type = {:rt}",
		"",
		0, 0,
		dbx.Params{"name": name, "rt": resourceType},
	)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(records))
	for _, r := range records {
		ids = append(ids, r.GetString("resource_id"))
	}
	return ids, nil
}

// GetUniqueTagNames returns distinct tag names for a resource type, with counts.
func (s *TagStore) GetUniqueTagNames(resourceType string) ([]TagNameCount, error) {
	var results []TagNameCount

	records, err := s.app.FindRecordsByFilter(
		constants.CollTags,
		"resource_type = {:rt}",
		"name",
		0, 0,
		dbx.Params{"rt": resourceType},
	)
	if err != nil {
		return nil, err
	}

	counts := make(map[string]*TagNameCount)
	for _, r := range records {
		name := r.GetString("name")
		color := r.GetString("color")
		if tc, ok := counts[name]; ok {
			tc.Count++
		} else {
			counts[name] = &TagNameCount{Name: name, Color: color, Count: 1}
		}
	}
	for _, tc := range counts {
		results = append(results, *tc)
	}
	return results, nil
}

// GetAllTags returns all tag records, optionally filtered by resource type.
func (s *TagStore) GetAllTags(resourceType string) ([]*core.Record, error) {
	if resourceType != "" {
		return s.app.FindRecordsByFilter(
			constants.CollTags,
			"resource_type = {:rt}",
			"name",
			0, 0,
			dbx.Params{"rt": resourceType},
		)
	}
	return s.app.FindRecordsByFilter(constants.CollTags, "", "name", 0, 0)
}

// TagNameCount represents a unique tag name with its usage count.
type TagNameCount struct {
	Name  string `json:"name"`
	Color string `json:"color"`
	Count int    `json:"count"`
}
