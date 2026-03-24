package jobs

import (
	"log/slog"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/services"
)

// RegisterAlertJobs registers the alert evaluation cron job (every 30s).
func RegisterAlertJobs(app core.App, engine *services.AlertEngine) {
	if err := app.Cron().Add("alert_evaluator", "*/1 * * * *", func() {
		engine.EvaluateRules()
	}); err != nil {
		slog.Error("failed to register alert evaluator job", "error", err)
	}

	slog.Info("alert jobs registered", "evaluator", "*/1min")
}
