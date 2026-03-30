package routes

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/hooks"
	"github.com/onflux-tech/dockeeper/internal/manager/middleware"
	"github.com/onflux-tech/dockeeper/internal/manager/store"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

const (
	accessTokenDuration  = 15 * time.Minute
	refreshTokenDuration = 7 * 24 * time.Hour
)

// RegisterAuthRoutes registers all /api/dk/auth/* routes.
func RegisterAuthRoutes(se *core.ServeEvent, authLimiter *middleware.RateLimiter, rbacStore *store.RBACStore) {
	auth := se.Router.Group("/api/dk/auth")

	auth.POST("/setup", handleSetup).BindFunc(authLimiter.Limit)
	auth.POST("/login", handleLogin(rbacStore)).BindFunc(authLimiter.Limit)
	auth.POST("/refresh", handleRefresh(rbacStore)).BindFunc(middleware.RequireAuth)
	auth.POST("/logout", handleLogout).BindFunc(middleware.RequireAuth)
	auth.POST("/password", handleChangePassword).BindFunc(middleware.RequireAuth)
	auth.GET("/me", handleMe(rbacStore)).BindFunc(middleware.RequireAuth)
	auth.GET("/status", handleStatus)
}

// POST /api/dk/auth/setup — create the first admin account.
func handleSetup(e *core.RequestEvent) error {
	if !hooks.NeedsSetup(e.App) {
		return e.JSON(http.StatusForbidden, map[string]string{
			"error": "admin already configured",
		})
	}

	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := e.BindBody(&body); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
	}

	if body.Email == "" || body.Password == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "email and password are required",
		})
	}

	col, err := e.App.FindCachedCollectionByNameOrId(core.CollectionNameSuperusers)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to find superusers collection",
		})
	}

	admin := core.NewRecord(col)
	admin.SetEmail(body.Email)
	admin.SetPassword(body.Password)

	if err := e.App.Save(admin); err != nil {
		slog.Error("failed to create admin", "error", err)
		return e.JSON(http.StatusBadRequest, map[string]string{
			"error": "failed to create admin: " + err.Error(),
		})
	}

	slog.Info("admin account created", "email", body.Email)

	return respondWithTokens(e, admin, true, nil)
}

// POST /api/dk/auth/login — authenticate with email and password.
func handleLogin(rbacStore *store.RBACStore) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var body struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := e.BindBody(&body); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid request body",
			})
		}

		if body.Email == "" || body.Password == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{
				"error": "email and password are required",
			})
		}

		// Try dk_users first (regular users).
		dkUser, err := e.App.FindAuthRecordByEmail(constants.CollDkUsers, body.Email)
		if err == nil {
			if !dkUser.ValidatePassword(body.Password) {
				slog.Warn("login attempt with wrong password", "email", body.Email, "ip", e.RealIP())
				return e.JSON(http.StatusUnauthorized, map[string]string{
					"error": "invalid credentials",
				})
			}

			// Check if user is enabled.
			if !dkUser.GetBool("enabled") {
				slog.Warn("login attempt for disabled user", "email", body.Email, "ip", e.RealIP())
				return e.JSON(http.StatusForbidden, map[string]string{
					"error": "account is disabled",
				})
			}

			caps, _ := rbacStore.GetUserCapabilities(dkUser.Id)

			slog.Info("user logged in", "email", body.Email, "ip", e.RealIP())
			return respondWithTokens(e, dkUser, false, caps)
		}

		// Fallback: try _superusers (admin users).
		admin, err := e.App.FindAuthRecordByEmail(core.CollectionNameSuperusers, body.Email)
		if err != nil {
			slog.Warn("login attempt with unknown email", "email", body.Email, "ip", e.RealIP())
			return e.JSON(http.StatusUnauthorized, map[string]string{
				"error": "invalid credentials",
			})
		}

		if !admin.ValidatePassword(body.Password) {
			slog.Warn("login attempt with wrong password", "email", body.Email, "ip", e.RealIP())
			return e.JSON(http.StatusUnauthorized, map[string]string{
				"error": "invalid credentials",
			})
		}

		slog.Info("admin logged in", "email", body.Email, "ip", e.RealIP())

		return respondWithTokens(e, admin, true, nil)
	}
}

// POST /api/dk/auth/refresh — issue new tokens from a valid access token.
func handleRefresh(rbacStore *store.RBACStore) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		isSuperuser := e.Auth.IsSuperuser()
		var caps []string
		if !isSuperuser {
			caps, _ = rbacStore.GetUserCapabilities(e.Auth.Id)
		}
		return respondWithTokens(e, e.Auth, isSuperuser, caps)
	}
}

// POST /api/dk/auth/logout — invalidate all tokens by rotating the token key.
func handleLogout(e *core.RequestEvent) error {
	e.Auth.RefreshTokenKey()
	if err := e.App.Save(e.Auth); err != nil {
		slog.Error("failed to rotate token key on logout", "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "logout failed",
		})
	}

	slog.Info("user logged out", "email", e.Auth.Email(), "ip", e.RealIP())

	return e.NoContent(http.StatusNoContent)
}

// GET /api/dk/auth/me — return the authenticated user's data with capabilities.
func handleMe(rbacStore *store.RBACStore) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		isSuperuser := e.Auth.IsSuperuser()

		var caps []string
		if isSuperuser {
			caps = constants.AllCapabilities
		} else {
			var err error
			caps, err = rbacStore.GetUserCapabilities(e.Auth.Id)
			if err != nil {
				caps = []string{}
			}
		}

		return e.JSON(http.StatusOK, map[string]any{
			"id":           e.Auth.Id,
			"email":        e.Auth.Email(),
			"created":      e.Auth.GetDateTime("created"),
			"updated":      e.Auth.GetDateTime("updated"),
			"is_superuser": isSuperuser,
			"capabilities": caps,
		})
	}
}

// POST /api/dk/auth/password — change the authenticated user's password.
func handleChangePassword(e *core.RequestEvent) error {
	var body struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := e.BindBody(&body); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if body.CurrentPassword == "" || body.NewPassword == "" {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "current_password and new_password are required"})
	}

	if len(body.NewPassword) < 8 {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": "new password must be at least 8 characters"})
	}

	if !e.Auth.ValidatePassword(body.CurrentPassword) {
		return e.JSON(http.StatusUnauthorized, map[string]string{"error": "current password is incorrect"})
	}

	e.Auth.SetPassword(body.NewPassword)
	if err := e.App.Save(e.Auth); err != nil {
		slog.Error("failed to update password", "error", err)
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update password"})
	}

	slog.Info("user password changed", "email", e.Auth.Email(), "ip", e.RealIP())
	return e.JSON(http.StatusOK, map[string]string{"status": "password_changed"})
}

// GET /api/dk/auth/status — return whether initial setup is needed.
func handleStatus(e *core.RequestEvent) error {
	return e.JSON(http.StatusOK, map[string]any{
		"needs_setup": hooks.NeedsSetup(e.App),
	})
}

// respondWithTokens generates access + refresh tokens and returns the auth response.
func respondWithTokens(e *core.RequestEvent, user *core.Record, isSuperuser bool, caps []string) error {
	accessToken, err := user.NewStaticAuthToken(accessTokenDuration)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to generate access token",
		})
	}

	refreshToken, err := user.NewStaticAuthToken(refreshTokenDuration)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to generate refresh token",
		})
	}

	if isSuperuser {
		caps = constants.AllCapabilities
	}
	if caps == nil {
		caps = []string{}
	}

	return e.JSON(http.StatusOK, map[string]any{
		"token":   accessToken,
		"refresh": refreshToken,
		"admin": map[string]any{
			"id":           user.Id,
			"email":        user.Email(),
			"created":      user.GetDateTime("created"),
			"updated":      user.GetDateTime("updated"),
			"is_superuser": isSuperuser,
			"capabilities": caps,
		},
	})
}
