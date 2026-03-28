package routes

import (
	"fmt"
	"net/http"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/services"
	"github.com/onflux-tech/dockeeper/internal/manager/store"
)

// RegisterActionTokenRoutes registers the public action token execution endpoint.
func RegisterActionTokenRoutes(se *core.ServeEvent, tokenSvc *services.ActionTokenService, auditStore *store.AuditStore) {
	// No auth — token IS the auth.
	se.Router.GET("/api/dk/actions/{token}", func(e *core.RequestEvent) error {
		token := e.Request.PathValue("token")
		if token == "" {
			return e.HTML(http.StatusBadRequest, actionHTML("Error", "Missing action token."))
		}

		result, err := tokenSvc.Execute(token)
		if err != nil {
			msg := err.Error()
			status := http.StatusBadRequest

			switch msg {
			case "token not found":
				status = http.StatusNotFound
				msg = "This action link is invalid or has already been used."
			case "token already used":
				status = http.StatusGone
				msg = "This action has already been executed."
			case "token expired":
				status = http.StatusGone
				msg = "This action link has expired."
			default:
				if len(msg) > 100 {
					msg = msg[:100]
				}
			}

			return e.HTML(status, actionHTML("Action Failed", msg))
		}

		// Audit log
		if auditStore != nil {
			auditStore.Log("action_token_executed", "", "action_token", token[:8]+"...", e.RealIP(), map[string]any{
				"action":  result.Action,
				"message": result.Message,
			})
		}

		return e.HTML(http.StatusOK, actionHTML("Action Executed", result.Message))
	})
}

// actionHTML generates a simple action result HTML page.
func actionHTML(title, message string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Dockeeper — %s</title>
<style>
  body { font-family: system-ui, sans-serif; background: #0a0a0b; color: #e4e4e7; display: flex; justify-content: center; align-items: center; min-height: 100vh; margin: 0; }
  .card { background: #18181b; border: 1px solid #27272a; border-radius: 12px; padding: 2rem; max-width: 400px; text-align: center; }
  h1 { font-size: 1.25rem; margin: 0 0 0.75rem; }
  p { color: #a1a1aa; font-size: 0.875rem; margin: 0 0 1.5rem; }
  a { color: #22c55e; text-decoration: none; font-size: 0.875rem; }
  a:hover { text-decoration: underline; }
</style>
</head>
<body>
  <div class="card">
    <h1>%s</h1>
    <p>%s</p>
    <a href="/">Go to Dashboard</a>
  </div>
</body>
</html>`, title, title, message)
}
