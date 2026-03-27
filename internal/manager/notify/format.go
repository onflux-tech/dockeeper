package notify

import (
	"fmt"
	"strings"
)

// formatMessage produces a human-readable alert notification text (plain text).
func formatMessage(msg AlertMessage) string {
	return AlertMessage2Text(msg)
}

// AlertMessage2Text converts an AlertMessage to a readable plain-text string.
func AlertMessage2Text(msg AlertMessage) string {
	var b strings.Builder

	title := "FIRING"
	if msg.State == "resolved" {
		title = "RESOLVED"
	}

	fmt.Fprintf(&b, "%s | %s\n\n", title, msg.RuleName)
	fmt.Fprintf(&b, "Metric:    %s %s %.2f\n", msg.Metric, operatorSymbol(msg.Operator), msg.Threshold)
	fmt.Fprintf(&b, "Current:  %.2f\n", msg.Value)
	fmt.Fprintf(&b, "Target:    %s (%s)\n", msg.TargetName, msg.TargetType)

	return b.String()
}

// AlertMessage2HTML converts an AlertMessage to Telegram-safe HTML string.
func AlertMessage2HTML(msg AlertMessage) string {
	var b strings.Builder

	title := "FIRING"
	if msg.State == "resolved" {
		title = "RESOLVED"
	}

	fmt.Fprintf(&b, "<b>%s</b> | <b>%s</b>\n\n", title, escapeHTML(msg.RuleName))
	fmt.Fprintf(&b, "Metric:    <code>%s %s %.2f</code>\n", escapeHTML(msg.Metric), escapeHTML(operatorSymbol(msg.Operator)), msg.Threshold)
	fmt.Fprintf(&b, "Current:  <code>%.2f</code>\n", msg.Value)
	fmt.Fprintf(&b, "Target:    %s <i>(%s)</i>\n", escapeHTML(msg.TargetName), escapeHTML(msg.TargetType))

	return b.String()
}

// escapeHTML escapes special characters for Telegram HTML parse mode.
func escapeHTML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}

func operatorSymbol(op string) string {
	switch op {
	case "gt":
		return ">"
	case "lt":
		return "<"
	case "gte":
		return ">="
	case "lte":
		return "<="
	case "eq":
		return "=="
	default:
		return op
	}
}
