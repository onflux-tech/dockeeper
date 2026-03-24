package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/notify"
	"github.com/onflux-tech/dockeeper/internal/manager/store"
	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// AlertEngine evaluates alert rules against stored metrics and triggers notifications.
type AlertEngine struct {
	app       core.App
	store     *store.AlertStore
	notifySvc *NotificationService
}

// NewAlertEngine creates an AlertEngine.
func NewAlertEngine(app core.App, alertStore *store.AlertStore, notifySvc *NotificationService) *AlertEngine {
	return &AlertEngine{app: app, store: alertStore, notifySvc: notifySvc}
}

// EvaluateRules checks all enabled alert rules and transitions state as needed.
func (e *AlertEngine) EvaluateRules() {
	rules, err := e.store.GetEnabledRules()
	if err != nil {
		slog.Error("alert engine: failed to load rules", "error", err)
		return
	}

	ctx := context.Background()

	for _, rule := range rules {
		e.evaluateRule(ctx, rule)
	}
}

func (e *AlertEngine) evaluateRule(ctx context.Context, rule *core.Record) {
	metric := rule.GetString("metric")
	operator := rule.GetString("operator")
	threshold := rule.GetFloat("threshold")
	duration := rule.GetInt("duration") // seconds
	targetType := rule.GetString("target_type")
	targetID := rule.GetString("target_id")
	currentState := rule.GetString("state")

	if metric == "" || targetID == "" {
		return
	}

	// Query metrics for the last `duration` seconds.
	samples, err := e.queryMetrics(targetType, targetID, metric, duration)
	if err != nil {
		slog.Debug("alert engine: metrics query failed", "rule", rule.Id, "error", err)
		return
	}

	// Need at least 1 sample to evaluate.
	if len(samples) == 0 {
		return
	}

	// Check if ALL samples satisfy the condition.
	conditionMet := true
	var latestValue float64
	for _, val := range samples {
		latestValue = val
		if !compare(val, operator, threshold) {
			conditionMet = false
			break
		}
	}

	// State transitions.
	switch {
	case conditionMet && currentState == "ok":
		// OK → FIRING
		slog.Info("alert engine: rule firing", "rule", rule.Id, "metric", metric, "value", latestValue)
		if err := e.store.UpdateRuleState(rule.Id, "firing"); err != nil {
			slog.Error("alert engine: update state failed", "rule", rule.Id, "error", err)
			return
		}

		targetName := e.resolveTargetName(targetType, targetID)
		channels := e.getChannelIDs(rule)
		e.notifySvc.Dispatch(ctx, channels, rule.Id, notify.AlertMessage{
			RuleName:   rule.GetString("name"),
			Metric:     metric,
			Operator:   operator,
			Threshold:  threshold,
			Value:      latestValue,
			TargetType: targetType,
			TargetName: targetName,
			State:      "firing",
		})

	case !conditionMet && currentState == "firing":
		// FIRING → RESOLVED
		slog.Info("alert engine: rule resolved", "rule", rule.Id, "metric", metric, "value", latestValue)
		if err := e.store.UpdateRuleState(rule.Id, "resolved"); err != nil {
			slog.Error("alert engine: update state failed", "rule", rule.Id, "error", err)
			return
		}

		targetName := e.resolveTargetName(targetType, targetID)
		channels := e.getChannelIDs(rule)
		e.notifySvc.Dispatch(ctx, channels, rule.Id, notify.AlertMessage{
			RuleName:   rule.GetString("name"),
			Metric:     metric,
			Operator:   operator,
			Threshold:  threshold,
			Value:      latestValue,
			TargetType: targetType,
			TargetName: targetName,
			State:      "resolved",
		})

		// Transition back to OK after resolved notification.
		_ = e.store.UpdateRuleState(rule.Id, "ok")
	}
}

// queryMetrics fetches metric values for the given target within the duration window.
func (e *AlertEngine) queryMetrics(targetType, targetID, metric string, durationSec int) ([]float64, error) {
	cutoff := time.Now().UTC().Add(-time.Duration(durationSec) * time.Second).Format(time.RFC3339)

	var collection string
	var filterField string
	switch targetType {
	case "server":
		collection = constants.CollServerMetricsRaw
		filterField = "server"
	case "container":
		collection = constants.CollContainerMetricsRaw
		filterField = "container"
	default:
		return nil, fmt.Errorf("unknown target type: %s", targetType)
	}

	filter := fmt.Sprintf("%s = '%s' && timestamp >= '%s'", filterField, targetID, cutoff)
	records, err := e.app.FindRecordsByFilter(collection, filter, "-timestamp", 200, 0)
	if err != nil {
		return nil, err
	}

	values := make([]float64, 0, len(records))
	for _, rec := range records {
		val := rec.GetFloat(metric)
		values = append(values, val)
	}

	return values, nil
}

func compare(value float64, operator string, threshold float64) bool {
	switch operator {
	case "gt":
		return value > threshold
	case "lt":
		return value < threshold
	case "gte":
		return value >= threshold
	case "lte":
		return value <= threshold
	case "eq":
		return value == threshold
	default:
		return false
	}
}

func (e *AlertEngine) resolveTargetName(targetType, targetID string) string {
	switch targetType {
	case "server":
		rec, err := e.app.FindRecordById(constants.CollServers, targetID)
		if err != nil {
			return targetID
		}
		return rec.GetString("hostname")
	case "container":
		rec, err := e.app.FindRecordById(constants.CollContainers, targetID)
		if err != nil {
			return targetID
		}
		return rec.GetString("name")
	default:
		return targetID
	}
}

func (e *AlertEngine) getChannelIDs(rule *core.Record) []string {
	raw := rule.Get("channels")
	switch v := raw.(type) {
	case []string:
		return v
	case []any:
		ids := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				ids = append(ids, s)
			}
		}
		return ids
	default:
		return nil
	}
}
