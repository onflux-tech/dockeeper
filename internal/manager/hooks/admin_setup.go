package hooks

import (
	"log/slog"

	"github.com/pocketbase/pocketbase/core"
)

// RegisterAdminSetupHook logs whether the system needs initial admin setup.
func RegisterAdminSetupHook(app core.App) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		if NeedsSetup(app) {
			slog.Warn("no admin found — initial setup required via POST /api/dk/auth/setup")
		} else {
			slog.Info("admin account configured")
		}
		return se.Next()
	})
}

// NeedsSetup returns true if no superuser exists yet.
func NeedsSetup(app core.App) bool {
	records, err := app.FindAllRecords(core.CollectionNameSuperusers)
	if err != nil {
		return true
	}
	// Ignore the default installer account.
	for _, r := range records {
		if r.Email() != core.DefaultInstallerEmail {
			return false
		}
	}
	return true
}
