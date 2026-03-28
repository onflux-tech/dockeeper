package notify

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
)

// NtfyChannel sends push notifications via ntfy.sh or self-hosted ntfy.
type NtfyChannel struct {
	ServerURL string
	Topic     string
	Token     string
}

// NewNtfyChannel creates an ntfy channel from config.
func NewNtfyChannel(config map[string]any) (*NtfyChannel, error) {
	topic, _ := config["topic"].(string)
	if topic == "" {
		return nil, fmt.Errorf("ntfy: topic is required")
	}

	serverURL := "https://ntfy.sh"
	if s, ok := config["server_url"].(string); ok && s != "" {
		serverURL = s
	}

	token, _ := config["token"].(string)

	return &NtfyChannel{ServerURL: serverURL, Topic: topic, Token: token}, nil
}

func (c *NtfyChannel) Type() string { return "ntfy" }

func (c *NtfyChannel) Send(ctx context.Context, msg AlertMessage) error {
	url := fmt.Sprintf("%s/%s", c.ServerURL, c.Topic)

	body := formatMessage(msg)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBufferString(body))
	if err != nil {
		return fmt.Errorf("ntfy request: %w", err)
	}

	title := fmt.Sprintf("%s: %s", msg.State, msg.RuleName)
	req.Header.Set("Title", title)
	req.Header.Set("Tags", ntfyTags(msg.State))
	req.Header.Set("Priority", ntfyPriority(msg.State))

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	// Add action buttons via ntfy Actions header.
	if len(msg.Actions) > 0 {
		var actions []string
		for _, a := range msg.Actions {
			actions = append(actions, fmt.Sprintf("view, %s, %s", a.Label, a.URL))
		}
		req.Header.Set("Actions", strings.Join(actions, "; "))
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ntfy send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("ntfy: status %d", resp.StatusCode)
	}
	return nil
}

func ntfyPriority(state string) string {
	if state == "firing" {
		return "urgent"
	}
	return "default"
}

func ntfyTags(state string) string {
	if state == "firing" {
		return "warning,computer"
	}
	return "white_check_mark,computer"
}
