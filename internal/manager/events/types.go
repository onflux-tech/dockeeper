package events

import "time"

// EventType identifies the kind of system event.
type EventType string

const (
	// Container events.
	ContainerCreated    EventType = "container.created"
	ContainerStarted    EventType = "container.started"
	ContainerStopped    EventType = "container.stopped"
	ContainerCrashed    EventType = "container.crashed"
	ContainerRemoved    EventType = "container.removed"
	ContainerRestarting EventType = "container.restarting"

	// Server events.
	ServerOnline   EventType = "server.online"
	ServerOffline  EventType = "server.offline"
	ServerEnrolled EventType = "server.enrolled"
	ServerRemoved  EventType = "server.removed"

	// Compose events.
	ComposeDeployed EventType = "compose.deployed"
	ComposeStopped  EventType = "compose.stopped"
	ComposeFailed   EventType = "compose.failed"

	// System events.
	BackupCompleted EventType = "backup.completed"
	BackupFailed    EventType = "backup.failed"

	// Image events.
	ImagePulled         EventType = "image.pulled"
	ImageRemoved        EventType = "image.removed"
	ImageUpdateAvailable EventType = "image.update.available"

	// Update events.
	ContainerUpdateCompleted  EventType = "container.update.completed"
	ContainerUpdateFailed     EventType = "container.update.failed"
	ContainerUpdateRolledback EventType = "container.update.rolledback"

	// Health events.
	HealthDegraded  EventType = "container.health.degraded"
	HealthRecovered EventType = "container.health.recovered"
	HealthRestart   EventType = "container.health.restart"
	HealthGaveUp    EventType = "container.health.gave_up"

	// Batch events.
	BatchStarted   EventType = "batch.started"
	BatchCompleted EventType = "batch.completed"
	BatchFailed    EventType = "batch.failed"
)

// Severity indicates the importance level of an event.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityCritical Severity = "critical"
)

// Event represents a system event published through the EventBus.
type Event struct {
	Type          EventType      `json:"type"`
	Severity      Severity       `json:"severity"`
	ServerID      string         `json:"server_id,omitempty"`
	ServerName    string         `json:"server_name,omitempty"`
	ContainerID   string         `json:"container_id,omitempty"`
	ContainerName string         `json:"container_name,omitempty"`
	Details       map[string]any `json:"details,omitempty"`
	Timestamp     time.Time      `json:"timestamp"`
}

// SeverityFor returns the default severity for a given event type.
func SeverityFor(t EventType) Severity {
	switch t {
	case ContainerCrashed, ServerOffline, ComposeFailed, BackupFailed, ContainerUpdateFailed, HealthGaveUp, BatchFailed:
		return SeverityCritical
	case ContainerStopped, ContainerRestarting, ComposeStopped, ServerRemoved, ContainerUpdateRolledback, HealthDegraded, HealthRestart, BatchStarted:
		return SeverityWarning
	default:
		return SeverityInfo
	}
}
