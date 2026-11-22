package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
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

	// In-memory timing trackers for active_delay / clear_delay.
	firstBreachAt sync.Map // ruleID -> time.Time
	firstClearAt  sync.Map // ruleID -> time.Time
}

// NewAlertEngine creates an AlertEngine.
func NewAlertEngine(app core.App, alertStore *store.AlertStore, notifySvc *NotificationService) *AlertEngine {
	return &AlertEngine{app: app, store: alertStore, notifySvc: notifySvc}
}

// EvaluateRules checks all enabled metric-type alert rules and transitions state as needed.
func (e *AlertEngine) EvaluateRules() {
	rules, err := e.store.GetEnabledRules()
	if err != nil {
		slog.Error("alert engine: failed to load rules", "error", err)
		return
	}

	ctx := context.Background()

	for _, rule := range rules {
		ruleType := rule.GetString("rule_type")
		if ruleType == "event" {
			continue // event rules handled by EventAlertHandler
		}
		e.evaluateMetricRule(ctx, rule)
	}
}

// HandleEventRule is called by the EventAlertHandler for event-based rules.
func (e *AlertEngine) HandleEventRule(ctx context.Context, rule *core.Record, serverName, containerName string) {
	if !e.checkCooldown(rule) {
		return
	}

	if rule.GetBool("suppress_during_updates") {
		scope := rule.GetString("scope")
		if e.isUpdateInProgress(scope) {
			slog.Debug("alert engine: suppressed during update", "rule", rule.Id)
			return
		}
	}

	// Event rules fire immediately.
	currentState := rule.GetString("state")
	if currentState != "firing" {
		if err := e.store.UpdateRuleState(rule.Id, "firing"); err != nil {
			slog.Error("alert engine: update state failed", "rule", rule.Id, "error", err)
			return
		}
	}

	channels := e.getChannelIDs(rule)
	targetName := serverName
	targetType := "server"
	if containerName != "" {
		targetName = containerName
		targetType = "container"
	}

	e.notifySvc.Dispatch(ctx, channels, rule.Id, notify.AlertMessage{
		RuleName:   rule.GetString("name"),
		TargetType: targetType,
		TargetName: targetName,
		State:      "firing",
	})
}

// ResolveEventRule transitions an event rule back to OK.
func (e *AlertEngine) ResolveEventRule(ctx context.Context, rule *core.Record, serverName, containerName string) {
	currentState := rule.GetString("state")
	if currentState != "firing" {
		return
	}

	if !e.checkCooldown(rule) {
		return
	}

	if err := e.store.UpdateRuleState(rule.Id, "resolved"); err != nil {
		slog.Error("alert engine: update state failed", "rule", rule.Id, "error", err)
		return
	}

	channels := e.getChannelIDs(rule)
	targetName := serverName
	targetType := "server"
	if containerName != "" {
		targetName = containerName
		targetType = "container"
	}

	e.notifySvc.Dispatch(ctx, channels, rule.Id, notify.AlertMessage{
		RuleName:   rule.GetString("name"),
		TargetType: targetType,
		TargetName: targetName,
		State:      "resolved",
	})

	_ = e.store.UpdateRuleState(rule.Id, "ok")
}

func (e *AlertEngine) evaluateMetricRule(ctx context.Context, rule *core.Record) {
	metric := rule.GetString("metric")
	operator := rule.GetString("operator")
	threshold := rule.GetFloat("threshold")
	duration := rule.GetInt("duration") // seconds
	targetType := rule.GetString("target_type")
	targetID := rule.GetString("target_id")
	currentState := rule.GetString("state")
	activeDelay := rule.GetInt("active_delay")
	clearDelay := rule.GetInt("clear_delay")

	if metric == "" || targetID == "" {
		return
	}

	// Check suppress_during_updates.
	if rule.GetBool("suppress_during_updates") {
		scope := fmt.Sprintf("container:%s", targetID)
		if targetType == "server" {
			scope = fmt.Sprintf("server:%s", targetID)
		}
		if e.isUpdateInProgress(scope) {
			slog.Debug("alert engine: suppressed during update", "rule", rule.Id)
			return
		}
	}

	samples, err := e.queryMetrics(targetType, targetID, metric, duration)
	if err != nil {
		slog.Debug("alert engine: metrics query failed", "rule", rule.Id, "error", err)
		return
	}

	if len(samples) == 0 {
		return
	}

	conditionMet := true
	var latestValue float64
	for _, val := range samples {
		latestValue = val
		if !compare(val, operator, threshold) {
			conditionMet = false
			break
		}
	}

	now := time.Now()

	switch {
	case conditionMet && (currentState == "ok" || currentState == "resolved"):
		// Track breach start for active_delay.
		_, _ = e.firstBreachAt.LoadOrStore(rule.Id, now)
		e.firstClearAt.Delete(rule.Id)

		// Check active_delay.
		if activeDelay > 0 {
			if breachStart, ok := e.firstBreachAt.Load(rule.Id); ok {
				if now.Sub(breachStart.(time.Time)) < time.Duration(activeDelay)*time.Second {
					return // Not enough time elapsed.
				}
			}
		}

		// Check cooldown.
		if !e.checkCooldown(rule) {
			return
		}

		slog.Info("alert engine: rule firing", "rule", rule.Id, "metric", metric, "value", latestValue)
		if err := e.store.UpdateRuleState(rule.Id, "firing"); err != nil {
			slog.Error("alert engine: update state failed", "rule", rule.Id, "error", err)
			return
		}

		e.firstBreachAt.Delete(rule.Id)

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
		// Track clear start for clear_delay.
		_, _ = e.firstClearAt.LoadOrStore(rule.Id, now)
		e.firstBreachAt.Delete(rule.Id)

		// Check clear_delay.
		if clearDelay > 0 {
			if clearStart, ok := e.firstClearAt.Load(rule.Id); ok {
				if now.Sub(clearStart.(time.Time)) < time.Duration(clearDelay)*time.Second {
					return // Not enough time elapsed.
				}
			}
		}

		// Check cooldown.
		if !e.checkCooldown(rule) {
			return
		}

		slog.Info("alert engine: rule resolved", "rule", rule.Id, "metric", metric, "value", latestValue)
		if err := e.store.UpdateRuleState(rule.Id, "resolved"); err != nil {
			slog.Error("alert engine: update state failed", "rule", rule.Id, "error", err)
			return
		}

		e.firstClearAt.Delete(rule.Id)

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

		_ = e.store.UpdateRuleState(rule.Id, "ok")

	case conditionMet && currentState == "firing":
		// Still firing — reset clear tracker.
		e.firstClearAt.Delete(rule.Id)

	case !conditionMet && (currentState == "ok" || currentState == "resolved"):
		// Still ok — reset breach tracker.
		e.firstBreachAt.Delete(rule.Id)
	}
}

// checkCooldown returns true if enough time has passed since last notification.
func (e *AlertEngine) checkCooldown(rule *core.Record) bool {
	cooldown := rule.GetInt("notification_cooldown")
	if cooldown <= 0 {
		return true
	}

	lastTriggered := rule.GetString("last_triggered")
	if lastTriggered == "" {
		return true
	}

	t, err := time.Parse(time.RFC3339, lastTriggered)
	if err != nil {
		return true
	}

	return time.Since(t) >= time.Duration(cooldown)*time.Second
}

// isUpdateInProgress checks if the target has an active update.
func (e *AlertEngine) isUpdateInProgress(scope string) bool {
	// Parse scope to get container/server ID.
	var filter string
	if len(scope) > 10 && scope[:10] == "container:" {
		containerID := scope[10:]
		filter = fmt.Sprintf("container = '%s' && (status = 'pending' || status = 'pulling' || status = 'replacing' || status = 'verifying')", containerID)
	} else if len(scope) > 7 && scope[:7] == "server:" {
		serverID := scope[7:]
		filter = fmt.Sprintf("server = '%s' && (status = 'pending' || status = 'pulling' || status = 'replacing' || status = 'verifying')", serverID)
	} else {
		return false
	}

	records, err := e.app.FindRecordsByFilter(constants.CollUpdateHistory, filter, "", 1, 0)
	if err != nil {
		return false
	}
	return len(records) > 0
}

// queryMetrics fetches metric values for the given target within the duration window.
// Supports direct field names (cpu_percent, disk_io_read, etc.) and computed metrics:
//   - swap_percent: computed from swap_used/swap_total
//   - max_temperature: max celsius from temperatures JSON array
//   - partition_usage_<mountpoint>: percent usage from partitions JSON array
func (e *AlertEngine) queryMetrics(targetType, targetID, metric string, durationSec int) ([]float64, error) {
	cutoff := time.Now().UTC().Add(-time.Duration(durationSec) * time.Second).Format("2006-01-02 15:04:05.000Z")

	switch targetType {
	case "server":
		if strings.HasPrefix(metric, "connectivity_") {
			filter := fmt.Sprintf("server = '%s' && timestamp >= '%s'", targetID, cutoff)
			return e.collectMetricValues(constants.CollServerConnectivityRaw, filter, metric, extractConnectivityMetric)
		}
		filter := fmt.Sprintf("server = '%s' && timestamp >= '%s'", targetID, cutoff)
		return e.collectMetricValues(constants.CollServerMetricsRaw, filter, metric, extractMetricValue)
	case "container":
		filter := fmt.Sprintf("container = '%s' && timestamp >= '%s'", targetID, cutoff)
		return e.collectMetricValues(constants.CollContainerMetricsRaw, filter, metric, extractMetricValue)
	case "host_app":
		hostApp, err := e.app.FindRecordById(constants.CollHostApps, targetID)
		if err != nil {
			return nil, fmt.Errorf("host app %s: %w", targetID, err)
		}
		filter := fmt.Sprintf("server = '%s' && source = '%s' && app = '%s' && timestamp >= '%s'",
			hostApp.GetString("server"), hostApp.GetString("source"), hostApp.GetString("name"), cutoff)
		return e.collectMetricValues(constants.CollHostAppMetricsRaw, filter, metric, extractMetricValue)
	default:
		return nil, fmt.Errorf("unknown target type: %s", targetType)
	}
}

func (e *AlertEngine) collectMetricValues(collection, filter, metric string, extract func(*core.Record, string) (float64, bool)) ([]float64, error) {
	records, err := e.app.FindRecordsByFilter(collection, filter, "-timestamp", 200, 0)
	if err != nil {
		return nil, err
	}

	values := make([]float64, 0, len(records))
	for _, rec := range records {
		if val, ok := extract(rec, metric); ok {
			values = append(values, val)
		}
	}

	return values, nil
}

func extractConnectivityMetric(rec *core.Record, metric string) (float64, bool) {
	boolValue := func(value bool) float64 {
		if value {
			return 1
		}
		return 0
	}
	switch metric {
	case "connectivity_internet_ok":
		return boolValue(rec.GetBool("internet_ok")), true
	case "connectivity_dns_ok":
		return boolValue(rec.GetBool("dns_ok")), true
	case "connectivity_gateway_ok":
		return boolValue(rec.GetBool("gateway_ok")), true
	case "connectivity_internet_ms":
		return rec.GetFloat("internet_ms"), true
	default:
		return 0, false
	}
}

// extractMetricValue reads a metric value from a record, handling both direct
// fields and computed/derived metrics.
func extractMetricValue(rec *core.Record, metric string) (float64, bool) {
	switch metric {
	case "swap_percent":
		used := rec.GetFloat("swap_used")
		total := rec.GetFloat("swap_total")
		if total <= 0 {
			return 0, true
		}
		return (used / total) * 100, true

	case "max_temperature":
		return extractMaxTemperature(rec), true

	default:
		// Check for partition_usage_<mountpoint> pattern.
		if strings.HasPrefix(metric, "partition_usage_") {
			mountpoint := strings.TrimPrefix(metric, "partition_usage_")
			val, ok := extractPartitionUsage(rec, mountpoint)
			return val, ok
		}

		// Direct field read — works for cpu_percent, mem_percent, disk_io_read,
		// disk_io_write, swap_used, swap_total, mem_available, etc.
		return rec.GetFloat(metric), true
	}
}

// extractMaxTemperature parses the temperatures JSON field and returns the max celsius.
func extractMaxTemperature(rec *core.Record) float64 {
	raw := rec.Get("temperatures")
	if raw == nil {
		return 0
	}

	var temps []struct {
		Celsius float64 `json:"celsius"`
	}

	switch v := raw.(type) {
	case string:
		if err := json.Unmarshal([]byte(v), &temps); err != nil {
			return 0
		}
	case []any:
		b, err := json.Marshal(v)
		if err != nil {
			return 0
		}
		if err := json.Unmarshal(b, &temps); err != nil {
			return 0
		}
	default:
		return 0
	}

	var maxTemp float64
	for _, t := range temps {
		if t.Celsius > maxTemp {
			maxTemp = t.Celsius
		}
	}
	return maxTemp
}

// extractPartitionUsage parses the partitions JSON field and returns the percent usage
// for the given mountpoint.
func extractPartitionUsage(rec *core.Record, mountpoint string) (float64, bool) {
	raw := rec.Get("partitions")
	if raw == nil {
		return 0, false
	}

	var parts []struct {
		Mountpoint string  `json:"mountpoint"`
		Percent    float64 `json:"percent"`
	}

	switch v := raw.(type) {
	case string:
		if err := json.Unmarshal([]byte(v), &parts); err != nil {
			return 0, false
		}
	case []any:
		b, err := json.Marshal(v)
		if err != nil {
			return 0, false
		}
		if err := json.Unmarshal(b, &parts); err != nil {
			return 0, false
		}
	default:
		return 0, false
	}

	for _, p := range parts {
		if p.Mountpoint == mountpoint {
			return p.Percent, true
		}
	}
	return 0, false
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
	case "host_app":
		rec, err := e.app.FindRecordById(constants.CollHostApps, targetID)
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
