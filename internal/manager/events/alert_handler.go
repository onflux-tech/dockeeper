package events

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// AlertEngineInterface is the subset of AlertEngine methods needed by the handler.
type AlertEngineInterface interface {
	HandleEventRule(ctx context.Context, rule *core.Record, serverName, containerName string)
	ResolveEventRule(ctx context.Context, rule *core.Record, serverName, containerName string)
}

// RuleStore is the subset of store.AlertStore needed by AlertHandler.
// This avoids an import cycle between events and store.
type RuleStore interface {
	GetEnabledRules() ([]*core.Record, error)
}

// AlertHandler subscribes to EventBus and triggers event-based alert rules.
type AlertHandler struct {
	app      core.App
	rules    RuleStore
	engine   AlertEngineInterface
	blackout BlackoutChecker
}

// BlackoutChecker checks if a server is in a blackout window.
type BlackoutChecker interface {
	IsBlackedOut(serverID string) bool
}

// NewAlertHandler creates the event-based alert handler.
func NewAlertHandler(app core.App, alertStore RuleStore, engine AlertEngineInterface, blackout BlackoutChecker) *AlertHandler {
	return &AlertHandler{
		app:      app,
		rules:    alertStore,
		engine:   engine,
		blackout: blackout,
	}
}

// Handle processes an event and evaluates matching event-based alert rules.
func (h *AlertHandler) Handle(e Event) {
	// Check blackout for this server.
	if h.blackout != nil && e.ServerID != "" && h.blackout.IsBlackedOut(e.ServerID) {
		slog.Debug("alert handler: event suppressed by blackout", "type", string(e.Type), "server", e.ServerID)
		return
	}

	rules, err := h.rules.GetEnabledRules()
	if err != nil {
		slog.Error("alert handler: failed to load rules", "error", err)
		return
	}

	ctx := context.Background()

	for _, rule := range rules {
		if rule.GetString("rule_type") != "event" {
			continue
		}

		ruleEventType := rule.GetString("event_type")
		if ruleEventType != string(e.Type) {
			// Check for resolution mappings.
			if !isResolutionEvent(ruleEventType, e) {
				continue
			}
			// This is a resolution event — resolve the rule.
			if matchesScope(rule, e) {
				h.engine.ResolveEventRule(ctx, rule, e.ServerName, e.ContainerName)
			}
			continue
		}

		if !matchesScope(rule, e) {
			continue
		}

		h.engine.HandleEventRule(ctx, rule, e.ServerName, e.ContainerName)
	}
}

// matchesScope checks if an event matches the rule's scope.
func matchesScope(rule *core.Record, e Event) bool {
	scope := rule.GetString("scope")
	if scope == "" || scope == "all" {
		return true
	}

	parts := strings.SplitN(scope, ":", 2)
	if len(parts) != 2 {
		return false
	}

	switch parts[0] {
	case "server":
		return e.ServerID == parts[1]
	case "container":
		return e.ContainerID == parts[1]
	case "connectivity":
		return e.ServerID == parts[1]
	case "host_app":
		bits := strings.SplitN(parts[1], ":", 3)
		if len(bits) != 3 {
			return false
		}
		return e.ServerID == bits[0] &&
			fmt.Sprint(e.Details["source"]) == bits[1] &&
			fmt.Sprint(e.Details["name"]) == bits[2]
	default:
		return false
	}
}

// resolutionMap maps recovery events to the failure events they resolve.
var resolutionMap = map[string][]string{
	string(ContainerStarted):            {string(ContainerCrashed), string(ContainerStopped)},
	string(ServerOnline):                {string(ServerOffline)},
	string(HostAppHealthRecovered):      {string(HostAppHealthDegraded)},
	string(ServerConnectivityRecovered): {string(ServerConnectivityDegraded)},
}

func isResolutionEvent(ruleEventType string, e Event) bool {
	resolves, ok := resolutionMap[string(e.Type)]
	if !ok {
		return false
	}
	for _, r := range resolves {
		if r == ruleEventType {
			return true
		}
	}
	return false
}

// GetEventRulesForContainer returns enabled event rules targeting a specific container.
func (h *AlertHandler) GetEventRulesForContainer(containerID string) []*core.Record {
	rules, err := h.app.FindRecordsByFilter(
		constants.CollAlertRules,
		fmt.Sprintf("enabled = true && rule_type = 'event' && scope = 'container:%s'", containerID),
		"", 0, 0,
	)
	if err != nil {
		return nil
	}
	return rules
}
