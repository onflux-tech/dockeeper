package notify

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

// SMTPChannel sends notifications via SMTP email.
type SMTPChannel struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
	To       []string
	UseTLS   bool
}

// NewSMTPChannel creates an SMTP channel from config.
func NewSMTPChannel(config map[string]any) (*SMTPChannel, error) {
	host, _ := config["host"].(string)
	from, _ := config["from"].(string)
	toRaw, _ := config["to"].(string)
	if host == "" || from == "" || toRaw == "" {
		return nil, fmt.Errorf("smtp: host, from, and to are required")
	}

	port := "587"
	if p, ok := config["port"].(string); ok && p != "" {
		port = p
	} else if pf, ok := config["port"].(float64); ok {
		port = fmt.Sprintf("%d", int(pf))
	}

	username, _ := config["username"].(string)
	password, _ := config["password"].(string)
	useTLS, _ := config["tls"].(bool)

	recipients := strings.Split(toRaw, ",")
	cleaned := make([]string, 0, len(recipients))
	for _, r := range recipients {
		if t := strings.TrimSpace(r); t != "" {
			cleaned = append(cleaned, t)
		}
	}

	return &SMTPChannel{
		Host: host, Port: port,
		Username: username, Password: password,
		From: from, To: cleaned, UseTLS: useTLS,
	}, nil
}

func (c *SMTPChannel) Type() string { return "smtp" }

func (c *SMTPChannel) Send(ctx context.Context, msg AlertMessage) error {
	stateUpper := strings.ToUpper(msg.State)
	subject := fmt.Sprintf("[Dockeeper] %s: %s", msg.RuleName, stateUpper)

	body := buildHTMLEmail(msg)

	mime := "MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=\"UTF-8\"\r\n" +
		"From: " + c.From + "\r\n" +
		"To: " + strings.Join(c.To, ", ") + "\r\n" +
		"Subject: " + subject + "\r\n\r\n" +
		body

	addr := net.JoinHostPort(c.Host, c.Port)

	var auth smtp.Auth
	if c.Username != "" {
		auth = smtp.PlainAuth("", c.Username, c.Password, c.Host)
	}

	// Implicit TLS (port 465).
	if c.UseTLS && c.Port == "465" {
		return c.sendImplicitTLS(addr, auth, mime)
	}

	// STARTTLS (port 587 or other).
	return smtp.SendMail(addr, auth, c.From, c.To, []byte(mime))
}

func (c *SMTPChannel) sendImplicitTLS(addr string, auth smtp.Auth, message string) error {
	tlsConfig := &tls.Config{ServerName: c.Host, MinVersion: tls.VersionTLS12}
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("smtp tls dial: %w", err)
	}

	client, err := smtp.NewClient(conn, c.Host)
	if err != nil {
		return fmt.Errorf("smtp new client: %w", err)
	}
	defer func() { _ = client.Close() }()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(c.From); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	for _, to := range c.To {
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("smtp rcpt: %w", err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write([]byte(message)); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}

	return client.Quit()
}

func buildHTMLEmail(msg AlertMessage) string {
	color := "#2ECC71"
	if msg.State == "firing" {
		color = "#E74C3C"
	}

	var metricRow string
	if msg.Metric != "" {
		metricRow = fmt.Sprintf(`<tr><td style="padding:4px 8px;color:#666">Metric</td><td style="padding:4px 8px"><code>%s %s %.2f</code></td></tr>
		<tr><td style="padding:4px 8px;color:#666">Current</td><td style="padding:4px 8px"><code>%.2f</code></td></tr>`,
			escapeHTML(msg.Metric), escapeHTML(operatorSymbol(msg.Operator)), msg.Threshold, msg.Value)
	}

	html := fmt.Sprintf(`<!DOCTYPE html><html><body style="font-family:sans-serif;margin:0;padding:20px;background:#f5f5f5">
<div style="max-width:500px;margin:0 auto;background:#fff;border-radius:8px;overflow:hidden">
<div style="background:%s;color:#fff;padding:16px 20px;font-size:18px;font-weight:bold">%s — %s</div>
<table style="width:100%%;border-collapse:collapse;padding:16px">
<tr><td style="padding:4px 8px;color:#666">Target</td><td style="padding:4px 8px">%s (%s)</td></tr>
%s
</table>`,
		color, strings.ToUpper(msg.State), escapeHTML(msg.RuleName),
		escapeHTML(msg.TargetName), escapeHTML(msg.TargetType), metricRow)

	// Add action buttons.
	if len(msg.Actions) > 0 {
		html += `<div style="padding:12px 20px;text-align:center">`
		for _, a := range msg.Actions {
			html += fmt.Sprintf(`<a href="%s" style="display:inline-block;padding:8px 16px;margin:4px;background:#3498db;color:#fff;border-radius:4px;text-decoration:none;font-size:14px">%s</a>`,
				escapeHTML(a.URL), escapeHTML(a.Label))
		}
		html += `</div>`
	}

	html += `</div></body></html>`
	return html
}
