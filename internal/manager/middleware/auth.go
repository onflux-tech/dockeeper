package middleware

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"
)

// RequireSuperuser returns a handler that rejects requests without valid superuser auth.
func RequireSuperuser(e *core.RequestEvent) error {
	if e.Auth == nil || !e.Auth.IsSuperuser() {
		return e.JSON(http.StatusUnauthorized, map[string]string{
			"error": "authentication required",
		})
	}
	return e.Next()
}
