package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/store"
)

type contextKey string

const (
	ctxKeyCaps   contextKey = "dk_capabilities"
	ctxKeyUserID contextKey = "dk_user_id"
)

// RBAC holds the stores needed for capability-based access control.
type RBAC struct {
	rbacStore   *store.RBACStore
	apiKeyStore *store.APIKeyStore
}

// NewRBAC creates an RBAC middleware provider.
func NewRBAC(rbacStore *store.RBACStore, apiKeyStore *store.APIKeyStore) *RBAC {
	return &RBAC{rbacStore: rbacStore, apiKeyStore: apiKeyStore}
}

// RequireCapability returns a middleware that checks for a specific capability.
// Check order: PB superuser → API key → PB auth token → 401.
func (r *RBAC) RequireCapability(capability string) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// 1. PB superuser bypass.
		if e.Auth != nil && e.Auth.IsSuperuser() {
			return e.Next()
		}

		// 2. API key authentication via X-API-Key header.
		if apiKey := e.Request.Header.Get("X-API-Key"); apiKey != "" {
			rec, caps, err := r.apiKeyStore.ValidateKey(apiKey)
			if err != nil {
				return e.JSON(http.StatusUnauthorized, map[string]string{
					"error": "invalid API key",
				})
			}

			if !hasCap(caps, capability) {
				return e.JSON(http.StatusForbidden, map[string]string{
					"error":    "insufficient permissions",
					"required": capability,
				})
			}

			// Update last_used asynchronously.
			go r.apiKeyStore.UpdateLastUsed(rec.Id)

			// Attach to context.
			ctx := context.WithValue(e.Request.Context(), ctxKeyCaps, caps)
			ctx = context.WithValue(ctx, ctxKeyUserID, rec.GetString("user"))
			e.Request = e.Request.WithContext(ctx)

			return e.Next()
		}

		// 3. PB auth token (non-superuser).
		if e.Auth != nil {
			caps, err := r.rbacStore.GetUserCapabilities(e.Auth.Id)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{
					"error": "failed to load capabilities",
				})
			}

			if !hasCap(caps, capability) {
				return e.JSON(http.StatusForbidden, map[string]string{
					"error":    "insufficient permissions",
					"required": capability,
				})
			}

			ctx := context.WithValue(e.Request.Context(), ctxKeyCaps, caps)
			e.Request = e.Request.WithContext(ctx)

			return e.Next()
		}

		// 4. No authentication.
		return e.JSON(http.StatusUnauthorized, map[string]string{
			"error": "authentication required",
		})
	}
}

// hasCap checks if a capability exists in a list.
func hasCap(caps []string, required string) bool {
	for _, c := range caps {
		if c == required {
			return true
		}
	}

	// Check domain wildcards: "containers.manage" implies "containers.view"
	if strings.HasSuffix(required, ".view") {
		manageCap := strings.TrimSuffix(required, ".view") + ".manage"
		for _, c := range caps {
			if c == manageCap {
				return true
			}
		}
	}

	return false
}

// HasCapability checks if a specific user has a capability (for non-middleware use like WebSocket).
func (r *RBAC) HasCapability(userID string, capability string) bool {
	caps, err := r.rbacStore.GetUserCapabilities(userID)
	if err != nil {
		return false
	}
	return hasCap(caps, capability)
}

// GetCapsFromContext returns capabilities stored in request context.
func GetCapsFromContext(ctx context.Context) []string {
	caps, _ := ctx.Value(ctxKeyCaps).([]string)
	return caps
}

// GetUserIDFromContext returns the user ID from request context (for API key auth).
func GetUserIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(ctxKeyUserID).(string)
	return id
}
