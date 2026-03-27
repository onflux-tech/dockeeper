package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// DiscordChannel sends notifications via Discord webhook with rich embeds.
type DiscordChannel struct {
	WebhookURL string
}

// NewDiscordChannel creates a Discord channel from config.
func NewDiscordChannel(config map[string]any) (*DiscordChannel, error) {
	url, _ := config["webhook_url"].(string)
	if url == "" {
		return nil, fmt.Errorf("discord: webhook_url is required")
	}
	return &DiscordChannel{WebhookURL: url}, nil
}

func (c *DiscordChannel) Type() string { return "discord" }

func (c *DiscordChannel) Send(ctx context.Context, msg AlertMessage) error {
	color := 0x2ECC71 // green (resolved/info)
	emoji := "green_circle"
	if msg.State == "firing" {
		color = 0xE74C3C // red
		emoji = "red_circle"
	}

	title := fmt.Sprintf(":%s: %s: %s", emoji, msg.State, msg.RuleName)

	fields := []map[string]any{
		{"name": "Target", "value": fmt.Sprintf("%s (%s)", msg.TargetName, msg.TargetType), "inline": true},
	}
	if msg.Metric != "" {
		fields = append(fields,
			map[string]any{"name": "Metric", "value": fmt.Sprintf("%s %s %.2f", msg.Metric, operatorSymbol(msg.Operator), msg.Threshold), "inline": true},
			map[string]any{"name": "Current", "value": fmt.Sprintf("%.2f", msg.Value), "inline": true},
		)
	}

	payload := map[string]any{
		"embeds": []map[string]any{{
			"title":     title,
			"color":     color,
			"fields":    fields,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("discord marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("discord request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("discord send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return fmt.Errorf("discord: rate limited (429), retry later")
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("discord: status %d", resp.StatusCode)
	}
	return nil
}
