package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// TelegramChannel sends notifications via Telegram Bot API.
type TelegramChannel struct {
	BotToken string
	ChatID   string
}

// NewTelegramChannel creates a Telegram channel from PocketBase config JSON.
func NewTelegramChannel(config map[string]any) (*TelegramChannel, error) {
	botToken, _ := config["bot_token"].(string)
	chatID, _ := config["chat_id"].(string)

	if botToken == "" || chatID == "" {
		return nil, fmt.Errorf("telegram: bot_token and chat_id are required")
	}

	return &TelegramChannel{
		BotToken: botToken,
		ChatID:   chatID,
	}, nil
}

func (c *TelegramChannel) Type() string { return "telegram" }

func (c *TelegramChannel) Send(ctx context.Context, msg AlertMessage) error {
	text := AlertMessage2HTML(msg)

	payload := map[string]any{
		"chat_id":    c.ChatID,
		"text":       text,
		"parse_mode": "HTML",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("telegram marshal: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", c.BotToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("telegram send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram: status %d", resp.StatusCode)
	}
	return nil
}
