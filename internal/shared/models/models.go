package models

import "github.com/pocketbase/pocketbase/tools/types"

// Server represents a monitored server running a Dockeeper agent.
type Server struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	Hostname      string         `json:"hostname"`
	IPAddress     string         `json:"ip_address"`
	OS            string         `json:"os"`
	Arch          string         `json:"arch"`
	DockerVersion string         `json:"docker_version"`
	AgentVersion  string         `json:"agent_version"`
	Status        string         `json:"status"`       // online | offline | error
	Orchestrator  string         `json:"orchestrator"` // docker | podman | kubernetes
	LastSeen      types.DateTime `json:"last_seen"`
	EnrolledAt    types.DateTime `json:"enrolled_at"`
	Labels        types.JSONRaw  `json:"labels"`
	CertExpiresAt types.DateTime `json:"cert_expires_at"`
	Created       types.DateTime `json:"created"`
	Updated       types.DateTime `json:"updated"`
}

// Container represents a Docker/Podman container on a server.
type Container struct {
	ID                 string         `json:"id"`
	Server             string         `json:"server"`
	ContainerID        string         `json:"container_id"`
	Name               string         `json:"name"`
	Image              string         `json:"image"`
	Status             string         `json:"status"` // running | stopped | paused | restarting | removing | created | exited | dead
	Ports              types.JSONRaw  `json:"ports"`
	Labels             types.JSONRaw  `json:"labels"`
	EnvVars            types.JSONRaw  `json:"env_vars"`
	Volumes            types.JSONRaw  `json:"volumes"`
	Networks           types.JSONRaw  `json:"networks"`
	ResourceLimits     types.JSONRaw  `json:"resource_limits"`
	RestartPolicy      string         `json:"restart_policy"`
	Command            string         `json:"command"`
	Entrypoint         string         `json:"entrypoint"`
	ContainerCreatedAt types.DateTime `json:"container_created_at"`
	StartedAt          types.DateTime `json:"started_at"`
	ComposeStack       string         `json:"compose_stack"`
	Created            types.DateTime `json:"created"`
	Updated            types.DateTime `json:"updated"`
}

// Image represents a Docker image on a server.
type Image struct {
	ID             string         `json:"id"`
	Server         string         `json:"server"`
	ImageID        string         `json:"image_id"`
	Tags           types.JSONRaw  `json:"tags"`
	Size           float64        `json:"size"`
	ImageCreatedAt types.DateTime `json:"image_created_at"`
	InUse          bool           `json:"in_use"`
	Created        types.DateTime `json:"created"`
	Updated        types.DateTime `json:"updated"`
}

// Network represents a Docker network on a server.
type Network struct {
	ID        string         `json:"id"`
	Server    string         `json:"server"`
	NetworkID string         `json:"network_id"`
	Name      string         `json:"name"`
	Driver    string         `json:"driver"`
	Scope     string         `json:"scope"`
	Subnet    string         `json:"subnet"`
	Gateway   string         `json:"gateway"`
	Internal  bool           `json:"internal"`
	Labels    types.JSONRaw  `json:"labels"`
	Created   types.DateTime `json:"created"`
	Updated   types.DateTime `json:"updated"`
}

// Volume represents a Docker volume on a server.
type Volume struct {
	ID         string         `json:"id"`
	Server     string         `json:"server"`
	Name       string         `json:"name"`
	Driver     string         `json:"driver"`
	Mountpoint string         `json:"mountpoint"`
	Size       float64        `json:"size"`
	Labels     types.JSONRaw  `json:"labels"`
	InUse      bool           `json:"in_use"`
	Created    types.DateTime `json:"created"`
	Updated    types.DateTime `json:"updated"`
}

// ComposeStack represents a Docker Compose stack on a server.
type ComposeStack struct {
	ID            string         `json:"id"`
	Server        string         `json:"server"`
	Name          string         `json:"name"`
	YAMLContent   string         `json:"yaml_content"`
	Status        string         `json:"status"` // running | stopped | partial
	ServicesCount int            `json:"services_count"`
	DeployedAt    types.DateTime `json:"deployed_at"`
	Created       types.DateTime `json:"created"`
	Updated       types.DateTime `json:"updated"`
}

// ServerMetricsRaw holds raw server metrics collected every 10s.
type ServerMetricsRaw struct {
	ID         string         `json:"id"`
	Server     string         `json:"server"`
	Timestamp  types.DateTime `json:"timestamp"`
	CPUPercent float64        `json:"cpu_percent"`
	MemPercent float64        `json:"mem_percent"`
	MemUsed    float64        `json:"mem_used"`
	MemTotal   float64        `json:"mem_total"`
	DiskUsed   float64        `json:"disk_used"`
	DiskTotal  float64        `json:"disk_total"`
	NetRx      float64        `json:"net_rx"`
	NetTx      float64        `json:"net_tx"`
	Load1m     float64        `json:"load_1m"`
	Load5m     float64        `json:"load_5m"`
	Load15m    float64        `json:"load_15m"`
	Created    types.DateTime `json:"created"`
}

// ServerMetricsAggregated holds hourly/daily aggregated server metrics.
type ServerMetricsAggregated struct {
	ID         string         `json:"id"`
	Server     string         `json:"server"`
	Timestamp  types.DateTime `json:"timestamp"`
	CPUPercent float64        `json:"cpu_percent"`
	MemPercent float64        `json:"mem_percent"`
	MemUsed    float64        `json:"mem_used"`
	MemTotal   float64        `json:"mem_total"`
	DiskUsed   float64        `json:"disk_used"`
	DiskTotal  float64        `json:"disk_total"`
	NetRx      float64        `json:"net_rx"`
	NetTx      float64        `json:"net_tx"`
	Load1m     float64        `json:"load_1m"`
	Load5m     float64        `json:"load_5m"`
	Load15m    float64        `json:"load_15m"`
	Samples    int            `json:"samples"`
	Stats      types.JSONRaw  `json:"stats"` // {min_*, max_*, p95_*}
	Created    types.DateTime `json:"created"`
}

// ContainerMetricsRaw holds raw container metrics collected every 10s.
type ContainerMetricsRaw struct {
	ID         string         `json:"id"`
	Container  string         `json:"container"`
	Server     string         `json:"server"`
	Timestamp  types.DateTime `json:"timestamp"`
	CPUPercent float64        `json:"cpu_percent"`
	MemUsed    float64        `json:"mem_used"`
	MemLimit   float64        `json:"mem_limit"`
	NetRx      float64        `json:"net_rx"`
	NetTx      float64        `json:"net_tx"`
	BlockRead  float64        `json:"block_read"`
	BlockWrite float64        `json:"block_write"`
	PIDs       int            `json:"pids"`
	Created    types.DateTime `json:"created"`
}

// ContainerMetricsAggregated holds hourly/daily aggregated container metrics.
type ContainerMetricsAggregated struct {
	ID         string         `json:"id"`
	Container  string         `json:"container"`
	Server     string         `json:"server"`
	Timestamp  types.DateTime `json:"timestamp"`
	CPUPercent float64        `json:"cpu_percent"`
	MemUsed    float64        `json:"mem_used"`
	MemLimit   float64        `json:"mem_limit"`
	NetRx      float64        `json:"net_rx"`
	NetTx      float64        `json:"net_tx"`
	BlockRead  float64        `json:"block_read"`
	BlockWrite float64        `json:"block_write"`
	PIDs       int            `json:"pids"`
	Samples    int            `json:"samples"`
	Stats      types.JSONRaw  `json:"stats"` // {min_*, max_*, p95_*}
	Created    types.DateTime `json:"created"`
}

// UpdatePolicy defines auto-update rules for a container.
type UpdatePolicy struct {
	ID                 string         `json:"id"`
	Container          string         `json:"container"`
	Enabled            bool           `json:"enabled"`
	Schedule           string         `json:"schedule"`
	BackupBeforeUpdate bool           `json:"backup_before_update"`
	AutoRestart        bool           `json:"auto_restart"`
	Created            types.DateTime `json:"created"`
	Updated            types.DateTime `json:"updated"`
}

// BackupPolicy defines scheduled backup rules for a volume.
type BackupPolicy struct {
	ID            string         `json:"id"`
	Volume        string         `json:"volume"`
	Server        string         `json:"server"`
	Schedule      string         `json:"schedule"`
	RetentionDays int            `json:"retention_days"`
	S3Bucket      string         `json:"s3_bucket"`
	S3Prefix      string         `json:"s3_prefix"`
	Enabled       bool           `json:"enabled"`
	Created       types.DateTime `json:"created"`
	Updated       types.DateTime `json:"updated"`
}

// BackupRecord represents a single backup execution result.
type BackupRecord struct {
	ID          string         `json:"id"`
	Policy      string         `json:"policy"`
	VolumeName  string         `json:"volume_name"`
	Server      string         `json:"server"`
	S3Key       string         `json:"s3_key"`
	Size        float64        `json:"size"`
	Status      string         `json:"status"` // pending | running | success | failed
	StartedAt   types.DateTime `json:"started_at"`
	CompletedAt types.DateTime `json:"completed_at"`
	Error       string         `json:"error"`
	Created     types.DateTime `json:"created"`
}

// AlertRule defines a condition that triggers a notification.
type AlertRule struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Metric        string         `json:"metric"`
	Operator      string         `json:"operator"` // gt | lt | gte | lte | eq
	Threshold     float64        `json:"threshold"`
	Duration      int            `json:"duration"`
	TargetType    string         `json:"target_type"` // server | container
	TargetID      string         `json:"target_id"`
	Channels      []string       `json:"channels"`
	Enabled       bool           `json:"enabled"`
	State         string         `json:"state"` // ok | firing | resolved
	LastTriggered types.DateTime `json:"last_triggered"`
	Created       types.DateTime `json:"created"`
	Updated       types.DateTime `json:"updated"`
}

// NotificationChannel configures a delivery target (WhatsApp, Telegram, Webhook).
type NotificationChannel struct {
	ID      string         `json:"id"`
	Name    string         `json:"name"`
	Type    string         `json:"type"` // whatsapp | telegram | webhook
	Config  types.JSONRaw  `json:"config"`
	Enabled bool           `json:"enabled"`
	Created types.DateTime `json:"created"`
	Updated types.DateTime `json:"updated"`
}

// NotificationLog records each notification delivery attempt.
type NotificationLog struct {
	ID        string         `json:"id"`
	Channel   string         `json:"channel"`
	AlertRule string         `json:"alert_rule"`
	Status    string         `json:"status"` // sent | failed | retrying
	Message   string         `json:"message"`
	Error     string         `json:"error"`
	SentAt    types.DateTime `json:"sent_at"`
	Retries   int            `json:"retries"`
	Created   types.DateTime `json:"created"`
}

// AuditLog records all admin actions for compliance.
type AuditLog struct {
	ID         string         `json:"id"`
	Action     string         `json:"action"`
	User       string         `json:"user"`
	TargetType string         `json:"target_type"`
	TargetID   string         `json:"target_id"`
	Details    types.JSONRaw  `json:"details"`
	IPAddress  string         `json:"ip_address"`
	Timestamp  types.DateTime `json:"timestamp"`
	Created    types.DateTime `json:"created"`
}

// TerminalSession records a web terminal session to a container.
type TerminalSession struct {
	ID        string         `json:"id"`
	Container string         `json:"container"`
	Server    string         `json:"server"`
	User      string         `json:"user"`
	StartedAt types.DateTime `json:"started_at"`
	EndedAt   types.DateTime `json:"ended_at"`
	Duration  int            `json:"duration"`
	Created   types.DateTime `json:"created"`
}

// Setting stores a key-value application configuration entry.
type Setting struct {
	ID      string         `json:"id"`
	Key     string         `json:"key"`
	Value   types.JSONRaw  `json:"value"`
	Created types.DateTime `json:"created"`
	Updated types.DateTime `json:"updated"`
}

// TraefikRoute represents a Traefik route discovered on a server.
type TraefikRoute struct {
	ID          string         `json:"id"`
	Server      string         `json:"server"`
	Name        string         `json:"name"`
	Rule        string         `json:"rule"`
	Service     string         `json:"service"`
	Entrypoints types.JSONRaw  `json:"entrypoints"`
	TLS         bool           `json:"tls"`
	Middlewares types.JSONRaw  `json:"middlewares"`
	Provider    string         `json:"provider"`
	Status      string         `json:"status"`
	Created     types.DateTime `json:"created"`
	Updated     types.DateTime `json:"updated"`
}
