package events

import "log/slog"

// AlertHandler is a placeholder for event-based alerting (activated in Sprint 16).
type AlertHandler struct{}

// NewAlertHandler creates the stub alert handler.
func NewAlertHandler() *AlertHandler {
	return &AlertHandler{}
}

// Handle logs critical events for future alert evaluation.
func (h *AlertHandler) Handle(e Event) {
	if e.Severity == SeverityCritical {
		slog.Debug("alert handler: critical event received (stub)", "type", string(e.Type), "server", e.ServerID)
	}
}
