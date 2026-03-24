package notify

import (
	"fmt"
	"strings"
)

// formatMessage produces a human-readable alert notification text.
func formatMessage(msg AlertMessage) string {
	return AlertMessage2Text(msg)
}

// AlertMessage2Text converts an AlertMessage to a readable string.
func AlertMessage2Text(msg AlertMessage) string {
	var b strings.Builder

	icon := "🔴"
	title := "FIRING"
	if msg.State == "resolved" {
		icon = "✅"
		title = "RESOLVED"
	}

	fmt.Fprintf(&b, "%s *[%s] %s*\n", icon, title, msg.RuleName)
	fmt.Fprintf(&b, "Metric: %s %s %.2f\n", msg.Metric, operatorSymbol(msg.Operator), msg.Threshold)
	fmt.Fprintf(&b, "Current: %.2f\n", msg.Value)
	fmt.Fprintf(&b, "Target: %s (%s)\n", msg.TargetName, msg.TargetType)

	return b.String()
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
