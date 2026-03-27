package constants

// Collection names used across the application.
const (
	CollServers              = "servers"
	CollContainers           = "containers"
	CollImages               = "images"
	CollNetworks             = "networks"
	CollVolumes              = "volumes"
	CollComposeStacks        = "compose_stacks"
	CollServerMetricsRaw     = "server_metrics_raw"
	CollServerMetricsHourly  = "server_metrics_hourly"
	CollServerMetricsDaily   = "server_metrics_daily"
	CollContainerMetricsRaw  = "container_metrics_raw"
	CollContainerMetricsHourly = "container_metrics_hourly"
	CollContainerMetricsDaily  = "container_metrics_daily"
	CollUpdatePolicies       = "update_policies"
	CollBackupPolicies       = "backup_policies"
	CollBackupRecords        = "backup_records"
	CollAlertRules           = "alert_rules"
	CollNotificationChannels = "notification_channels"
	CollNotificationLogs     = "notification_logs"
	CollAuditLogs            = "audit_logs"
	CollTerminalSessions     = "terminal_sessions"
	CollSettings             = "settings"
	CollTraefikRoutes        = "traefik_routes"

	// V2 Collections (Sprint 13+)
	CollGroups       = "groups"
	CollGroupMembers = "group_members"
	CollAPIKeys      = "api_keys"
)
