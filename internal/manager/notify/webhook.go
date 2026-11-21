package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/onflux-tech/dockeeper/internal/shared/security"
)

// WebhookChannel sends notifications via generic HTTP POST.
type WebhookChannel struct {
	URL     string
	Headers map[string]string
}

// NewWebhookChannel creates a Webhook channel from PocketBase config JSON.
func NewWebhookChannel(config map[string]any) (*WebhookChannel, error) {
	url, _ := config["url"].(string)
	if url == "" {
		return nil, fmt.Errorf("webhook: url is required")
	}

	// SSRF validation: block private IPs, link-local, loopback.
	if err := security.ValidateWebhookURL(url); err != nil {
		return nil, fmt.Errorf("webhook: %w", err)
	}

	headers := make(map[string]string)
	if h, ok := config["headers"].(map[string]any); ok {
		for k, v := range h {
			if s, ok := v.(string); ok {
				headers[k] = s
			}
		}
	}

	return &WebhookChannel{URL: url, Headers: headers}, nil
}

func (c *WebhookChannel) Type() string { return "webhook" }

func (c *WebhookChannel) Send(ctx context.Context, msg AlertMessage) error {
	payload := map[string]any{
		"alert":       msg.RuleName,
		"state":       msg.State,
		"metric":      msg.Metric,
		"operator":    msg.Operator,
		"threshold":   msg.Threshold,
		"value":       msg.Value,
		"target_type": msg.TargetType,
		"target_name": msg.TargetName,
		"message":     formatMessage(msg),
	}
	if len(msg.Actions) > 0 {
		actions := make([]map[string]string, 0, len(msg.Actions))
		for _, a := range msg.Actions {
			actions = append(actions, map[string]string{"label": a.Label, "url": a.URL})
		}
		payload["actions"] = actions
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("webhook marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range c.Headers {
		req.Header.Set(k, v)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("webhook send: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook: status %d", resp.StatusCode)
	}
	return nil
}
