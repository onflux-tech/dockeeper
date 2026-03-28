package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/onflux-tech/dockeeper/internal/manager/notify"
	"github.com/onflux-tech/dockeeper/internal/manager/store"
)

const (
	maxRetries    = 3
	baseRetryWait = 1 * time.Second
)

// NotificationService dispatches alert messages to configured channels with retry logic.
type NotificationService struct {
	alertStore     *store.AlertStore
	actionTokenSvc *ActionTokenService
	baseURL        string
}

// NewNotificationService creates a NotificationService.
func NewNotificationService(alertStore *store.AlertStore) *NotificationService {
	return &NotificationService{alertStore: alertStore}
}

// SetActionTokenService wires the action token service for generating action URLs in notifications.
func (s *NotificationService) SetActionTokenService(svc *ActionTokenService, baseURL string) {
	s.actionTokenSvc = svc
	s.baseURL = baseURL
}

// Dispatch sends an alert message to the channels specified by their IDs.
func (s *NotificationService) Dispatch(ctx context.Context, channelIDs []string, ruleID string, msg notify.AlertMessage) {
	// Generate action tokens for firing alerts.
	if msg.State == "firing" && s.actionTokenSvc != nil && s.baseURL != "" {
		msg.Actions = s.generateActionURLs(ruleID, msg)
	}

	channels, err := s.alertStore.GetChannelsByIDs(channelIDs)
	if err != nil {
		slog.Error("notification: failed to load channels", "error", err)
		return
	}

	for _, ch := range channels {
		if !ch.GetBool("enabled") {
			continue
		}
		go s.sendWithRetry(ctx, ch, ruleID, msg)
	}
}

func (s *NotificationService) sendWithRetry(ctx context.Context, channelRec *core.Record, ruleID string, msg notify.AlertMessage) {
	channelID := channelRec.Id
	channelType := channelRec.GetString("type")

	sender, err := s.buildChannel(channelRec)
	if err != nil {
		slog.Error("notification: invalid channel config", "channel", channelID, "type", channelType, "error", err)
		_ = s.alertStore.CreateNotificationLog(channelID, ruleID, "failed", notify.AlertMessage2Text(msg), err.Error(), 0)
		return
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			wait := baseRetryWait * (1 << (attempt - 1)) // 1s, 2s, 4s
			select {
			case <-ctx.Done():
				return
			case <-time.After(wait):
			}
		}

		lastErr = sender.Send(ctx, msg)
		if lastErr == nil {
			_ = s.alertStore.CreateNotificationLog(channelID, ruleID, "sent", notify.AlertMessage2Text(msg), "", attempt)
			return
		}

		slog.Warn("notification: send failed, retrying",
			"channel", channelID, "type", channelType, "attempt", attempt+1, "error", lastErr,
		)
	}

	// All retries exhausted.
	errMsg := ""
	if lastErr != nil {
		errMsg = lastErr.Error()
	}
	_ = s.alertStore.CreateNotificationLog(channelID, ruleID, "failed", notify.AlertMessage2Text(msg), errMsg, maxRetries)
	slog.Error("notification: all retries exhausted", "channel", channelID, "type", channelType, "rule", ruleID)
}

// TestChannel sends a test message to a single channel by record.
func (s *NotificationService) TestChannel(ctx context.Context, channelRec *core.Record) error {
	sender, err := s.buildChannel(channelRec)
	if err != nil {
		return fmt.Errorf("invalid channel config: %w", err)
	}

	testMsg := notify.AlertMessage{
		RuleName:   "Test Alert",
		Metric:     "cpu_percent",
		Operator:   "gt",
		Threshold:  80,
		Value:      92.5,
		TargetType: "server",
		TargetName: "test-server",
		State:      "firing",
	}

	return sender.Send(ctx, testMsg)
}

func (s *NotificationService) buildChannel(rec *core.Record) (notify.Channel, error) {
	channelType := rec.GetString("type")

	// Parse config JSON.
	// PocketBase returns types.JsonRaw ([]byte alias) for JSON fields loaded from DB,
	// but may return map[string]any for in-memory records. Handle all cases.
	configRaw := rec.Get("config")
	config := make(map[string]any)
	switch v := configRaw.(type) {
	case map[string]any:
		config = v
	case string:
		if v == "" {
			return nil, fmt.Errorf("empty config")
		}
		if err := json.Unmarshal([]byte(v), &config); err != nil {
			return nil, fmt.Errorf("invalid config JSON: %w", err)
		}
	default:
		if configRaw == nil {
			return nil, fmt.Errorf("empty config")
		}
		raw, err := json.Marshal(configRaw)
		if err != nil || string(raw) == "null" {
			return nil, fmt.Errorf("unsupported or empty config type: %T", configRaw)
		}
		if err := json.Unmarshal(raw, &config); err != nil {
			return nil, fmt.Errorf("invalid config JSON: %w", err)
		}
	}

	switch channelType {
	case "whatsapp":
		return notify.NewWhatsAppChannel(config)
	case "telegram":
		return notify.NewTelegramChannel(config)
	case "webhook":
		return notify.NewWebhookChannel(config)
	case "discord":
		return notify.NewDiscordChannel(config)
	case "slack":
		return notify.NewSlackChannel(config)
	case "smtp":
		return notify.NewSMTPChannel(config)
	case "ntfy":
		return notify.NewNtfyChannel(config)
	default:
		return nil, fmt.Errorf("unsupported channel type: %s", channelType)
	}
}

// generateActionURLs creates single-use action token URLs for a firing alert.
func (s *NotificationService) generateActionURLs(ruleID string, msg notify.AlertMessage) []notify.ActionURL {
	var actions []notify.ActionURL

	// Acknowledge alert.
	if token, err := s.actionTokenSvc.Create("ack_alert", map[string]string{"rule_id": ruleID}, "alert_engine"); err == nil {
		actions = append(actions, notify.ActionURL{
			Label: "Acknowledge",
			URL:   fmt.Sprintf("%s/api/dk/actions/%s", s.baseURL, token),
		})
	}

	// Silence alert for 4 hours.
	if token, err := s.actionTokenSvc.Create("silence_alert", map[string]string{"rule_id": ruleID}, "alert_engine"); err == nil {
		actions = append(actions, notify.ActionURL{
			Label: "Silence 4h",
			URL:   fmt.Sprintf("%s/api/dk/actions/%s", s.baseURL, token),
		})
	}

	return actions
}
