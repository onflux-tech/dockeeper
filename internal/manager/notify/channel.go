package notify

import "context"

// ActionURL represents a clickable action link embedded in notifications.
type ActionURL struct {
	Label string
	URL   string
}

// AlertMessage contains alert information to be sent via notification channels.
type AlertMessage struct {
	RuleName   string
	Metric     string
	Operator   string
	Threshold  float64
	Value      float64
	TargetType string
	TargetName string
	State      string // "firing" or "resolved"
	Actions    []ActionURL
}

// Channel sends alert notifications through a specific delivery mechanism.
type Channel interface {
	// Send delivers an alert message. Returns nil on success.
	Send(ctx context.Context, msg AlertMessage) error

	// Type returns the channel type identifier.
	Type() string
}
