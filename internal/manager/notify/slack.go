package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// SlackChannel sends notifications via Slack incoming webhook with Block Kit.
type SlackChannel struct {
	WebhookURL string
}

// NewSlackChannel creates a Slack channel from config.
func NewSlackChannel(config map[string]any) (*SlackChannel, error) {
	url, _ := config["webhook_url"].(string)
	if url == "" {
		return nil, fmt.Errorf("slack: webhook_url is required")
	}
	return &SlackChannel{WebhookURL: url}, nil
}

func (c *SlackChannel) Type() string { return "slack" }

func (c *SlackChannel) Send(ctx context.Context, msg AlertMessage) error {
	emoji := ":large_green_circle:"
	if msg.State == "firing" {
		emoji = ":red_circle:"
	}

	headerText := fmt.Sprintf("%s %s: %s", emoji, msg.State, msg.RuleName)

	sectionFields := []map[string]any{
		{"type": "mrkdwn", "text": fmt.Sprintf("*Target:*\n%s (%s)", msg.TargetName, msg.TargetType)},
	}
	if msg.Metric != "" {
		sectionFields = append(sectionFields,
			map[string]any{"type": "mrkdwn", "text": fmt.Sprintf("*Metric:*\n%s %s %.2f", msg.Metric, operatorSymbol(msg.Operator), msg.Threshold)},
			map[string]any{"type": "mrkdwn", "text": fmt.Sprintf("*Current:*\n%.2f", msg.Value)},
		)
	}

	blocks := []map[string]any{
		{"type": "header", "text": map[string]any{"type": "plain_text", "text": headerText}},
		{"type": "section", "fields": sectionFields},
	}

	payload := map[string]any{"blocks": blocks}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("slack marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("slack send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return fmt.Errorf("slack: status %d — %s", resp.StatusCode, string(respBody))
	}
	return nil
}
