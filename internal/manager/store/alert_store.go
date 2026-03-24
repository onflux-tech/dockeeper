package store

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/shared/constants"
)

// AlertStore provides CRUD for alert_rules, notification_channels, and notification_logs.
type AlertStore struct {
	app core.App
}

// NewAlertStore creates an AlertStore.
func NewAlertStore(app core.App) *AlertStore {
	return &AlertStore{app: app}
}

// ── Alert Rules ──────────────────────────────────────────

// GetEnabledRules returns all enabled alert rules.
func (s *AlertStore) GetEnabledRules() ([]*core.Record, error) {
	records, err := s.app.FindRecordsByFilter(
		constants.CollAlertRules,
		"enabled = true",
		"created",
		0, 0,
	)
	if err != nil {
		return nil, fmt.Errorf("list enabled rules: %w", err)
	}
	return records, nil
}

// UpdateRuleState updates the state and last_triggered fields of an alert rule.
func (s *AlertStore) UpdateRuleState(ruleID, state string) error {
	rec, err := s.app.FindRecordById(constants.CollAlertRules, ruleID)
	if err != nil {
		return fmt.Errorf("rule not found: %w", err)
	}

	rec.Set("state", state)
	rec.Set("last_triggered", time.Now().UTC().Format(time.RFC3339))

	return s.app.Save(rec)
}

// CreateRule creates a new alert rule.
func (s *AlertStore) CreateRule(fields map[string]any) (string, error) {
	col, err := s.app.FindCachedCollectionByNameOrId(constants.CollAlertRules)
	if err != nil {
		return "", fmt.Errorf("collection %s: %w", constants.CollAlertRules, err)
	}

	rec := core.NewRecord(col)
	for k, v := range fields {
		rec.Set(k, v)
	}
	if rec.GetString("state") == "" {
		rec.Set("state", "ok")
	}

	if err := s.app.Save(rec); err != nil {
		return "", fmt.Errorf("save rule: %w", err)
	}
	return rec.Id, nil
}

// UpdateRule updates an existing alert rule.
func (s *AlertStore) UpdateRule(ruleID string, fields map[string]any) error {
	rec, err := s.app.FindRecordById(constants.CollAlertRules, ruleID)
	if err != nil {
		return fmt.Errorf("rule not found: %w", err)
	}

	for k, v := range fields {
		rec.Set(k, v)
	}
	return s.app.Save(rec)
}

// DeleteRule deletes an alert rule.
func (s *AlertStore) DeleteRule(ruleID string) error {
	rec, err := s.app.FindRecordById(constants.CollAlertRules, ruleID)
	if err != nil {
		return fmt.Errorf("rule not found: %w", err)
	}
	return s.app.Delete(rec)
}

// ── Notification Channels ────────────────────────────────

// GetEnabledChannels returns all enabled notification channels.
func (s *AlertStore) GetEnabledChannels() ([]*core.Record, error) {
	records, err := s.app.FindRecordsByFilter(
		constants.CollNotificationChannels,
		"enabled = true",
		"created",
		0, 0,
	)
	if err != nil {
		return nil, fmt.Errorf("list enabled channels: %w", err)
	}
	return records, nil
}

// GetChannelsByIDs returns channels matching the given IDs.
func (s *AlertStore) GetChannelsByIDs(ids []string) ([]*core.Record, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	filter := ""
	for i, id := range ids {
		if i > 0 {
			filter += " || "
		}
		filter += fmt.Sprintf("id = '%s'", id)
	}

	records, err := s.app.FindRecordsByFilter(
		constants.CollNotificationChannels,
		filter,
		"", 0, 0,
	)
	if err != nil {
		return nil, fmt.Errorf("get channels by ids: %w", err)
	}
	return records, nil
}

// CreateChannel creates a new notification channel.
func (s *AlertStore) CreateChannel(fields map[string]any) (string, error) {
	col, err := s.app.FindCachedCollectionByNameOrId(constants.CollNotificationChannels)
	if err != nil {
		return "", fmt.Errorf("collection %s: %w", constants.CollNotificationChannels, err)
	}

	rec := core.NewRecord(col)
	for k, v := range fields {
		rec.Set(k, v)
	}
	if err := s.app.Save(rec); err != nil {
		return "", fmt.Errorf("save channel: %w", err)
	}
	return rec.Id, nil
}

// UpdateChannel updates an existing notification channel.
func (s *AlertStore) UpdateChannel(channelID string, fields map[string]any) error {
	rec, err := s.app.FindRecordById(constants.CollNotificationChannels, channelID)
	if err != nil {
		return fmt.Errorf("channel not found: %w", err)
	}

	for k, v := range fields {
		rec.Set(k, v)
	}
	return s.app.Save(rec)
}

// DeleteChannel deletes a notification channel.
func (s *AlertStore) DeleteChannel(channelID string) error {
	rec, err := s.app.FindRecordById(constants.CollNotificationChannels, channelID)
	if err != nil {
		return fmt.Errorf("channel not found: %w", err)
	}
	return s.app.Delete(rec)
}

// ── Notification Logs ────────────────────────────────────

// CreateNotificationLog saves a notification attempt.
func (s *AlertStore) CreateNotificationLog(channelID, alertRuleID, status, message, errMsg string, retries int) error {
	col, err := s.app.FindCachedCollectionByNameOrId(constants.CollNotificationLogs)
	if err != nil {
		return fmt.Errorf("collection %s: %w", constants.CollNotificationLogs, err)
	}

	rec := core.NewRecord(col)
	rec.Set("channel", channelID)
	rec.Set("alert_rule", alertRuleID)
	rec.Set("status", status)
	rec.Set("message", message)
	rec.Set("error", errMsg)
	rec.Set("sent_at", time.Now().UTC().Format(time.RFC3339))
	rec.Set("retries", retries)

	return s.app.Save(rec)
}
