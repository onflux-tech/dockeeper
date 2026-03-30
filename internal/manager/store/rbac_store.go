package store

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// RBACStore provides CRUD for groups, members, and capability queries.
type RBACStore struct {
	app   core.App
	cache sync.Map // userID -> cachedCaps
}

type cachedCaps struct {
	caps      []string
	expiresAt time.Time
}

const capCacheTTL = 60 * time.Second

// NewRBACStore creates an RBACStore.
func NewRBACStore(app core.App) *RBACStore {
	return &RBACStore{app: app}
}

// GetGroups returns all groups.
func (s *RBACStore) GetGroups() ([]*core.Record, error) {
	return s.app.FindRecordsByFilter(constants.CollGroups, "", "-created", 0, 0)
}

// GetGroupByID returns a single group by ID.
func (s *RBACStore) GetGroupByID(id string) (*core.Record, error) {
	return s.app.FindRecordById(constants.CollGroups, id)
}

// CreateGroup creates a new group.
func (s *RBACStore) CreateGroup(name, description string, capabilities []string, isPreset bool) (*core.Record, error) {
	col, err := s.app.FindCachedCollectionByNameOrId(constants.CollGroups)
	if err != nil {
		return nil, err
	}

	rec := core.NewRecord(col)
	rec.Set("name", name)
	rec.Set("description", description)
	rec.Set("capabilities", capabilities)
	rec.Set("is_preset", isPreset)

	if err := s.app.Save(rec); err != nil {
		return nil, err
	}
	return rec, nil
}

// UpdateGroup updates a group's capabilities and description.
func (s *RBACStore) UpdateGroup(id string, name, description string, capabilities []string) (*core.Record, error) {
	rec, err := s.app.FindRecordById(constants.CollGroups, id)
	if err != nil {
		return nil, err
	}

	rec.Set("name", name)
	rec.Set("description", description)
	rec.Set("capabilities", capabilities)

	if err := s.app.Save(rec); err != nil {
		return nil, err
	}

	s.invalidateGroupCache(id)
	return rec, nil
}

// DeleteGroup deletes a non-preset group.
func (s *RBACStore) DeleteGroup(id string) error {
	rec, err := s.app.FindRecordById(constants.CollGroups, id)
	if err != nil {
		return err
	}

	s.invalidateGroupCache(id)
	return s.app.Delete(rec)
}

// GetGroupMembers returns all members of a group.
func (s *RBACStore) GetGroupMembers(groupID string) ([]*core.Record, error) {
	return s.app.FindRecordsByFilter(
		constants.CollGroupMembers,
		"group = {:groupId}",
		"-created", 0, 0,
		dbx.Params{"groupId": groupID},
	)
}

// AddMember adds a user to a group.
func (s *RBACStore) AddMember(groupID, userID string) (*core.Record, error) {
	col, err := s.app.FindCachedCollectionByNameOrId(constants.CollGroupMembers)
	if err != nil {
		return nil, err
	}

	rec := core.NewRecord(col)
	rec.Set("group", groupID)
	rec.Set("user", userID)

	if err := s.app.Save(rec); err != nil {
		return nil, err
	}

	s.cache.Delete(userID)
	return rec, nil
}

// RemoveMember removes a user from a group.
func (s *RBACStore) RemoveMember(groupID, userID string) error {
	rec, err := s.app.FindFirstRecordByFilter(
		constants.CollGroupMembers,
		"group = {:groupId} && user = {:userId}",
		dbx.Params{"groupId": groupID, "userId": userID},
	)
	if err != nil {
		return err
	}

	s.cache.Delete(userID)
	return s.app.Delete(rec)
}

// GetUserCapabilities returns the merged capabilities for a user from all their groups.
func (s *RBACStore) GetUserCapabilities(userID string) ([]string, error) {
	// Check cache first (lazy expiry).
	if val, ok := s.cache.Load(userID); ok {
		cached := val.(cachedCaps)
		if time.Now().Before(cached.expiresAt) {
			return cached.caps, nil
		}
		s.cache.Delete(userID)
	}

	// Load memberships.
	members, err := s.app.FindRecordsByFilter(
		constants.CollGroupMembers,
		"user = {:userId}",
		"", 0, 0,
		dbx.Params{"userId": userID},
	)
	if err != nil {
		return nil, err
	}

	capSet := make(map[string]struct{})
	for _, m := range members {
		groupID := m.GetString("group")
		group, err := s.app.FindRecordById(constants.CollGroups, groupID)
		if err != nil {
			continue
		}

		var caps []string
		raw := group.Get("capabilities")
		switch v := raw.(type) {
		case string:
			_ = json.Unmarshal([]byte(v), &caps)
		case []string:
			caps = v
		case []any:
			for _, item := range v {
				if str, ok := item.(string); ok {
					caps = append(caps, str)
				}
			}
		default:
			// PB JSONField returns types.JSONRaw ([]byte) — try JSON unmarshal.
			if b, err := json.Marshal(raw); err == nil {
				_ = json.Unmarshal(b, &caps)
			}
		}

		for _, cap := range caps {
			capSet[cap] = struct{}{}
		}
	}

	result := make([]string, 0, len(capSet))
	for cap := range capSet {
		result = append(result, cap)
	}

	s.cache.Store(userID, cachedCaps{
		caps:      result,
		expiresAt: time.Now().Add(capCacheTTL),
	})

	return result, nil
}

// CountAdminGroupMembers returns the number of users in the Administrators group.
func (s *RBACStore) CountAdminGroupMembers() (int, error) {
	adminGroup, err := s.app.FindFirstRecordByFilter(constants.CollGroups, "name = 'Administrators'")
	if err != nil {
		return 0, err
	}

	members, err := s.GetGroupMembers(adminGroup.Id)
	if err != nil {
		return 0, err
	}
	return len(members), nil
}

// InvalidateUserCache removes a specific user from the cache.
func (s *RBACStore) InvalidateUserCache(userID string) {
	s.cache.Delete(userID)
}

// invalidateGroupCache removes all users from cache who belong to a group.
func (s *RBACStore) invalidateGroupCache(groupID string) {
	members, err := s.GetGroupMembers(groupID)
	if err != nil {
		// Flush entire cache on error.
		s.cache = sync.Map{}
		return
	}
	for _, m := range members {
		s.cache.Delete(m.GetString("user"))
	}
}
