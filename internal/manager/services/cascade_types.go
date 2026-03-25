package services

// ── Types ────────────────────────────────────────────

// CascadeResult represents the outcome of a cascade delete operation.
type CascadeResult struct {
	ServerID           string           `json:"server_id,omitempty"`
	EntityType         string           `json:"entity_type"` // server | container | volume
	EntityID           string           `json:"entity_id"`
	EntityName         string           `json:"entity_name"`
	DeletedResources   map[string]int64 `json:"deleted_resources"`   // type -> count
	ProtectedResources map[string]int64 `json:"protected_resources"` // type -> count
	Options            map[string]bool  `json:"options,omitempty"`   // cascade options used
}

// ContainerCascadePreview represents the impact of removing a container.
type ContainerCascadePreview struct {
	Entity Entity        `json:"entity"`
	Impact CascadeImpact `json:"impact"`
}

// VolumeCascadePreview represents the impact of removing a volume.
type VolumeCascadePreview struct {
	Entity  Entity              `json:"entity"`
	Impact  VolumeCascadeImpact `json:"impact"`
	InUseBy []string            `json:"in_use_by"` // container names using this volume
}

// ServerCascadePreview represents the impact of removing a server.
type ServerCascadePreview struct {
	Entity Entity              `json:"entity"`
	Impact ServerCascadeImpact `json:"impact"`
}

// Entity represents a deletable entity.
type Entity struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"` // server | container | volume | image | network
}

// CascadeImpact represents the counts and eligibility of related resources.
type CascadeImpact struct {
	AlertRules       ResourceCount          `json:"alert_rules"`
	Volumes          ResourceCountWithItems `json:"volumes"`
	NotificationLogs ResourceCount          `json:"notification_logs"`
	BackupPolicies   ResourceCount          `json:"backup_policies"`
	TerminalSessions ResourceCount          `json:"terminal_sessions"`
	MetricsRecords   ResourceCount          `json:"metrics_records"`
}

// VolumeCascadeImpact for volume deletion.
type VolumeCascadeImpact struct {
	BackupPolicies ResourceCount `json:"backup_policies"`
	BackupRecords  ResourceCount `json:"backup_records"`
}

// ServerCascadeImpact for server deletion (full impact).
type ServerCascadeImpact struct {
	Containers       ResourceCount `json:"containers"`
	Volumes          ResourceCount `json:"volumes"`
	Images           ResourceCount `json:"images"`
	Networks         ResourceCount `json:"networks"`
	ComposeStacks    ResourceCount `json:"compose_stacks"`
	AlertRules       ResourceCount `json:"alert_rules"`
	BackupPolicies   ResourceCount `json:"backup_policies"`
	BackupRecords    ResourceCount `json:"backup_records"`
	NotificationLogs ResourceCount `json:"notification_logs"`
	TerminalSessions ResourceCount `json:"terminal_sessions"`
	TraefikRoutes    ResourceCount `json:"traefik_routes"`
	MetricsRecords   ResourceCount `json:"metrics_records"`
}

// ResourceCount represents deletable and protected counts.
type ResourceCount struct {
	Deletable int64 `json:"deletable"`
	Protected int64 `json:"protected"`
}

// ResourceCountWithItems includes individual item details.
type ResourceCountWithItems struct {
	Deletable int64          `json:"deletable"`
	Protected int64          `json:"protected"`
	Items     []ResourceItem `json:"items"`
}

// ResourceItem represents a single resource with eligibility info.
type ResourceItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Eligible bool   `json:"eligible"`
	Reason   string `json:"reason"`
}

// ── Container Cascade Options ────────────────────────

// ContainerCascadeOptions controls what to delete when removing a container.
type ContainerCascadeOptions struct {
	RemoveVolumes          bool `json:"remove_volumes"`
	RemoveAlerts           bool `json:"remove_alerts"`
	RemoveBackups          bool `json:"remove_backups"`
	RemoveNotificationLogs bool `json:"remove_notification_logs"`
}

// VolumeCascadeOptions controls what to preserve when removing a volume.
type VolumeCascadeOptions struct {
	KeepBackupRecords bool `json:"keep_backup_records"`
}

// ── Docker mount/network structures for parsing ────────

// MountPoint represents a Docker volume mount.
type MountPoint struct {
	Type        string `json:"Type"`
	Name        string `json:"Name"`
	Source      string `json:"Source"`
	Destination string `json:"Destination"`
	Driver      string `json:"Driver,omitempty"`
	Mode        string `json:"Mode,omitempty"`
	RW          bool   `json:"RW,omitempty"`
	Propagation string `json:"Propagation,omitempty"`
}

// EndpointSettings represents Docker network endpoint settings.
type EndpointSettings map[string]struct {
	IPAddress           string            `json:"IPAddress"`
	IPPrefixLen         int               `json:"IPPrefixLen"`
	Gateway             string            `json:"Gateway"`
	MacAddress          string            `json:"MacAddress"`
	DriverOpt           map[string]string `json:"DriverOpt"`
	GlobalIPv6Address   string            `json:"GlobalIPv6Address"`
	GlobalIPv6PrefixLen int               `json:"GlobalIPv6PrefixLen"`
	IPv6Gateway         string            `json:"IPv6Gateway"`
	Aliases             []string          `json:"Aliases"`
}

// ── Alert Rule Target Info ────────────────────────

// AlertRuleTarget represents what an alert rule is targeting.
type AlertRuleTarget struct {
	Type string `json:"target_type"` // server | container
	ID   string `json:"target_id"`
}
