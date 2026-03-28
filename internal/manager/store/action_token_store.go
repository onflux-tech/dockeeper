package store

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// ActionTokenStore manages CRUD for action_tokens collection.
type ActionTokenStore struct {
	app core.App
}

// NewActionTokenStore creates a new ActionTokenStore.
func NewActionTokenStore(app core.App) *ActionTokenStore {
	return &ActionTokenStore{app: app}
}

// CreateToken generates a cryptographically random token and persists it.
func (s *ActionTokenStore) CreateToken(action string, params map[string]string, ttl time.Duration, createdBy string) (string, error) {
	// Generate 32-byte random token
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)

	col, err := s.app.FindCollectionByNameOrId(constants.CollActionTokens)
	if err != nil {
		return "", fmt.Errorf("find collection: %w", err)
	}

	rec := core.NewRecord(col)
	rec.Set("token", token)
	rec.Set("action", action)
	rec.Set("params", params)
	rec.Set("expires_at", time.Now().UTC().Add(ttl).Format(time.RFC3339))
	rec.Set("used", false)
	rec.Set("created_by", createdBy)

	if err := s.app.Save(rec); err != nil {
		return "", fmt.Errorf("save token: %w", err)
	}

	return token, nil
}

// GetByToken looks up an action token record by its token string.
func (s *ActionTokenStore) GetByToken(token string) (*core.Record, error) {
	return s.app.FindFirstRecordByFilter(
		constants.CollActionTokens,
		"token = {:token}",
		dbx.Params{"token": token},
	)
}

// MarkUsed marks a token as used.
func (s *ActionTokenStore) MarkUsed(rec *core.Record) error {
	rec.Set("used", true)
	rec.Set("used_at", time.Now().UTC().Format(time.RFC3339))
	return s.app.Save(rec)
}

// DeleteExpired removes tokens that are expired and used, or expired > 24h.
func (s *ActionTokenStore) DeleteExpired() (int64, error) {
	now := time.Now().UTC().Format("2006-01-02 15:04:05.000Z")
	old := time.Now().UTC().Add(-24 * time.Hour).Format("2006-01-02 15:04:05.000Z")

	// Delete: (used AND expired) OR (expired > 24h ago regardless of use)
	sql := `DELETE FROM action_tokens WHERE id IN (
		SELECT id FROM action_tokens
		WHERE (used = true AND expires_at < {:now})
		   OR (expires_at < {:old})
		LIMIT 1000
	)`

	res, err := s.app.DB().NewQuery(sql).Bind(dbx.Params{
		"now": now,
		"old": old,
	}).Execute()
	if err != nil {
		return 0, err
	}

	affected, _ := res.RowsAffected()
	return affected, nil
}
