package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"golang.org/x/crypto/bcrypt"

	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// APIKeyStore provides CRUD for API keys with bcrypt hashing.
type APIKeyStore struct {
	app core.App
}

// NewAPIKeyStore creates an APIKeyStore.
func NewAPIKeyStore(app core.App) *APIKeyStore {
	return &APIKeyStore{app: app}
}

// Create generates a new API key, stores bcrypt hash, and returns the plaintext key once.
func (s *APIKeyStore) Create(name, userID string, capabilities []string, expiresAt *time.Time) (string, *core.Record, error) {
	// Generate 32 random bytes → hex → prefix with "dk_" = 67 chars total.
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return "", nil, fmt.Errorf("generate key: %w", err)
	}
	plaintext := "dk_" + hex.EncodeToString(rawBytes)
	prefix := plaintext[:8] // "dk_XXXXX"

	hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if err != nil {
		return "", nil, fmt.Errorf("hash key: %w", err)
	}

	col, err := s.app.FindCachedCollectionByNameOrId(constants.CollAPIKeys)
	if err != nil {
		return "", nil, err
	}

	rec := core.NewRecord(col)
	rec.Set("name", name)
	rec.Set("key_prefix", prefix)
	rec.Set("key_hash", string(hash))
	rec.Set("user", userID)
	rec.Set("capabilities", capabilities)
	rec.Set("enabled", true)
	if expiresAt != nil {
		rec.Set("expires_at", expiresAt.UTC().Format("2006-01-02 15:04:05.000Z"))
	}

	if err := s.app.Save(rec); err != nil {
		return "", nil, err
	}

	return plaintext, rec, nil
}

// dummyHash is a pre-computed bcrypt hash used to ensure constant-time behavior
// when no matching key prefix is found (prevents timing oracle on prefix existence).
var dummyHash = func() []byte {
	h, _ := bcrypt.GenerateFromPassword([]byte("dummy-key-for-timing"), bcrypt.DefaultCost)
	return h
}()

// ValidateKey finds a key by prefix and validates the full plaintext.
// Returns the record and capabilities if valid.
func (s *APIKeyStore) ValidateKey(plaintext string) (*core.Record, []string, error) {
	if len(plaintext) < 8 {
		return nil, nil, fmt.Errorf("invalid key format")
	}
	prefix := plaintext[:8]

	records, err := s.app.FindRecordsByFilter(
		constants.CollAPIKeys,
		"key_prefix = {:prefix} && enabled = true",
		"", 0, 0,
		dbx.Params{"prefix": prefix},
	)
	if err != nil || len(records) == 0 {
		// Perform a dummy bcrypt comparison to prevent timing oracle on prefix existence.
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(plaintext))
		return nil, nil, fmt.Errorf("key not found")
	}

	for _, rec := range records {
		hash := rec.GetString("key_hash")
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintext)); err == nil {
			// Check expiry.
			expiresStr := rec.GetString("expires_at")
			if expiresStr != "" {
				expires, err := time.Parse("2006-01-02 15:04:05.000Z", expiresStr)
				if err == nil && time.Now().After(expires) {
					return nil, nil, fmt.Errorf("key expired")
				}
			}

			var caps []string
			raw := rec.Get("capabilities")
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
				// PocketBase may return types.JsonArray or similar;
				// marshal+unmarshal as a fallback.
				if b, err := json.Marshal(raw); err == nil {
					_ = json.Unmarshal(b, &caps)
				}
			}

			return rec, caps, nil
		}
	}

	return nil, nil, fmt.Errorf("invalid key")
}

// UpdateLastUsed updates the last_used_at timestamp.
func (s *APIKeyStore) UpdateLastUsed(id string) {
	rec, err := s.app.FindRecordById(constants.CollAPIKeys, id)
	if err != nil {
		return
	}
	rec.Set("last_used_at", time.Now().UTC().Format("2006-01-02 15:04:05.000Z"))
	_ = s.app.Save(rec)
}

// ListByUser returns all API keys for a user (or all if userID is empty).
func (s *APIKeyStore) ListByUser(userID string) ([]*core.Record, error) {
	filter := ""
	var params []dbx.Params
	if userID != "" {
		filter = "user = {:userId}"
		params = append(params, dbx.Params{"userId": userID})
	}
	return s.app.FindRecordsByFilter(constants.CollAPIKeys, filter, "-created", 0, 0, params...)
}

// Delete revokes an API key.
func (s *APIKeyStore) Delete(id string) error {
	rec, err := s.app.FindRecordById(constants.CollAPIKeys, id)
	if err != nil {
		return err
	}
	return s.app.Delete(rec)
}
