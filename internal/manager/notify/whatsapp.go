package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// WhatsAppChannel sends notifications via WuzAPI.
type WhatsAppChannel struct {
	Endpoint string // WuzAPI base URL (e.g. http://localhost:3000)
	Phone    string // recipient phone number
	Token    string // WuzAPI auth token
}

// NewWhatsAppChannel creates a WhatsApp channel from PocketBase config JSON.
func NewWhatsAppChannel(config map[string]any) (*WhatsAppChannel, error) {
	endpoint, _ := config["endpoint"].(string)
	phone, _ := config["phone"].(string)
	token, _ := config["token"].(string)

	if endpoint == "" || phone == "" {
		return nil, fmt.Errorf("whatsapp: endpoint and phone are required")
	}

	return &WhatsAppChannel{
		Endpoint: strings.TrimRight(endpoint, "/"),
		Phone:    phone,
		Token:    token,
	}, nil
}

func (c *WhatsAppChannel) Type() string { return "whatsapp" }

func (c *WhatsAppChannel) Send(ctx context.Context, msg AlertMessage) error {
	text := formatMessage(msg)

	payload := map[string]any{
		"Phone": c.Phone,
		"Body":  text,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("whatsapp marshal: %w", err)
	}

	url := c.Endpoint + "/chat/send/text"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("whatsapp request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("token", c.Token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("whatsapp send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("whatsapp: status %d", resp.StatusCode)
	}
	return nil
}
