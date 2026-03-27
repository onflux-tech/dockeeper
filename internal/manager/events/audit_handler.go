package events

// AuditLogger is an interface for writing audit log entries.
// This avoids an import cycle between events and store.
type AuditLogger interface {
	Log(action, userID, targetType, targetID, ip string, details map[string]any)
}

// AuditHandler creates audit_log entries for relevant system events.
type AuditHandler struct {
	audit AuditLogger
}

// NewAuditHandler creates an audit handler.
func NewAuditHandler(audit AuditLogger) *AuditHandler {
	return &AuditHandler{audit: audit}
}

// Handle maps the event to an audit log entry.
func (h *AuditHandler) Handle(e Event) {
	action := auditAction(e.Type)
	if action == "" {
		return
	}

	targetType, targetID := auditTarget(e)

	h.audit.Log(action, "system", targetType, targetID, "", e.Details)
}

// auditAction maps event types to audit log actions.
func auditAction(t EventType) string {
	switch t {
	case ContainerCreated:
		return "container.created"
	case ContainerStarted:
		return "container.started"
	case ContainerStopped:
		return "container.stopped"
	case ContainerCrashed:
		return "container.crashed"
	case ContainerRemoved:
		return "container.removed"
	case ServerOnline:
		return "server.online"
	case ServerOffline:
		return "server.offline"
	case ServerEnrolled:
		return "server.enrolled"
	case ServerRemoved:
		return "server.removed"
	case ComposeDeployed:
		return "compose.deployed"
	case ComposeStopped:
		return "compose.stopped"
	case ComposeFailed:
		return "compose.failed"
	case BackupCompleted:
		return "backup.completed"
	case BackupFailed:
		return "backup.failed"
	case ImagePulled:
		return "image.pulled"
	case ImageRemoved:
		return "image.removed"
	default:
		return ""
	}
}

// auditTarget extracts the target type and ID from the event.
func auditTarget(e Event) (string, string) {
	if e.ContainerID != "" {
		return "container", e.ContainerID
	}
	if e.ServerID != "" {
		return "server", e.ServerID
	}
	return "system", ""
}
