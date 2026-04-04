package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		// ── 1. servers ──────────────────────────────────────
		servers := core.NewBaseCollection("servers")
		servers.Fields.Add(
			&core.TextField{Name: "name", Required: true, Max: 255},
			&core.TextField{Name: "hostname", Max: 255},
			&core.TextField{Name: "ip_address", Max: 45},
			&core.TextField{Name: "os", Max: 100},
			&core.TextField{Name: "arch", Max: 50},
			&core.TextField{Name: "docker_version", Max: 50},
			&core.TextField{Name: "agent_version", Max: 50},
			&core.SelectField{
				Name:      "status",
				Values:    []string{"online", "offline", "error"},
				MaxSelect: 1,
				Required:  true,
			},
			&core.SelectField{
				Name:      "orchestrator",
				Values:    []string{"docker", "podman", "kubernetes"},
				MaxSelect: 1,
			},
			&core.DateField{Name: "last_seen"},
			&core.DateField{Name: "enrolled_at"},
			&core.JSONField{Name: "labels"},
			&core.DateField{Name: "cert_expires_at"},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		servers.AddIndex("idx_servers_status", false, "status", "")
		servers.AddIndex("idx_servers_name", false, "name", "")
		if err := app.Save(servers); err != nil {
			return err
		}

		// ── 2. compose_stacks ───────────────────────────────
		composeStacks := core.NewBaseCollection("compose_stacks")
		composeStacks.Fields.Add(
			&core.RelationField{
				Name:          "server",
				CollectionId:  servers.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.TextField{Name: "name", Required: true, Max: 255},
			&core.TextField{Name: "yaml_content"},
			&core.SelectField{
				Name:      "status",
				Values:    []string{"running", "stopped", "partial"},
				MaxSelect: 1,
			},
			&core.NumberField{Name: "services_count", OnlyInt: true},
			&core.DateField{Name: "deployed_at"},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		composeStacks.AddIndex("idx_compose_stacks_server", false, "server", "")
		composeStacks.AddIndex("idx_compose_stacks_status", false, "status", "")
		if err := app.Save(composeStacks); err != nil {
			return err
		}

		// ── 3. containers ───────────────────────────────────
		containers := core.NewBaseCollection("containers")
		containers.Fields.Add(
			&core.RelationField{
				Name:          "server",
				CollectionId:  servers.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.TextField{Name: "container_id", Required: true, Max: 100},
			&core.TextField{Name: "name", Required: true, Max: 255},
			&core.TextField{Name: "image", Required: true, Max: 500},
			&core.SelectField{
				Name:      "status",
				Values:    []string{"running", "stopped", "paused", "restarting", "removing", "created", "exited", "dead"},
				MaxSelect: 1,
				Required:  true,
			},
			&core.JSONField{Name: "ports"},
			&core.JSONField{Name: "labels"},
			&core.JSONField{Name: "env_vars"},
			&core.JSONField{Name: "volumes"},
			&core.JSONField{Name: "networks"},
			&core.JSONField{Name: "resource_limits"},
			&core.TextField{Name: "restart_policy", Max: 50},
			&core.TextField{Name: "command"},
			&core.TextField{Name: "entrypoint"},
			&core.DateField{Name: "container_created_at"},
			&core.DateField{Name: "started_at"},
			&core.RelationField{
				Name:         "compose_stack",
				CollectionId: composeStacks.Id,
				MaxSelect:    1,
			},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		containers.AddIndex("idx_containers_server", false, "server", "")
		containers.AddIndex("idx_containers_status", false, "status", "")
		containers.AddIndex("idx_containers_container_id", false, "container_id", "")
		if err := app.Save(containers); err != nil {
			return err
		}

		// ── 4. images ───────────────────────────────────────
		images := core.NewBaseCollection("images")
		images.Fields.Add(
			&core.RelationField{
				Name:          "server",
				CollectionId:  servers.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.TextField{Name: "image_id", Required: true, Max: 100},
			&core.JSONField{Name: "tags"},
			&core.NumberField{Name: "size"},
			&core.DateField{Name: "image_created_at"},
			&core.BoolField{Name: "in_use"},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		images.AddIndex("idx_images_server", false, "server", "")
		if err := app.Save(images); err != nil {
			return err
		}

		// ── 5. networks ─────────────────────────────────────
		networks := core.NewBaseCollection("networks")
		networks.Fields.Add(
			&core.RelationField{
				Name:          "server",
				CollectionId:  servers.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.TextField{Name: "network_id", Required: true, Max: 100},
			&core.TextField{Name: "name", Required: true, Max: 255},
			&core.TextField{Name: "driver", Max: 50},
			&core.TextField{Name: "scope", Max: 50},
			&core.TextField{Name: "subnet", Max: 50},
			&core.TextField{Name: "gateway", Max: 50},
			&core.BoolField{Name: "internal"},
			&core.JSONField{Name: "labels"},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		networks.AddIndex("idx_networks_server", false, "server", "")
		if err := app.Save(networks); err != nil {
			return err
		}

		// ── 6. volumes ──────────────────────────────────────
		volumes := core.NewBaseCollection("volumes")
		volumes.Fields.Add(
			&core.RelationField{
				Name:          "server",
				CollectionId:  servers.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.TextField{Name: "name", Required: true, Max: 255},
			&core.TextField{Name: "driver", Max: 50},
			&core.TextField{Name: "mountpoint"},
			&core.NumberField{Name: "size"},
			&core.JSONField{Name: "labels"},
			&core.BoolField{Name: "in_use"},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		volumes.AddIndex("idx_volumes_server", false, "server", "")
		if err := app.Save(volumes); err != nil {
			return err
		}

		// ── 7. server_metrics_raw ───────────────────────────
		serverMetricsRaw := core.NewBaseCollection("server_metrics_raw")
		serverMetricsRaw.Fields.Add(
			&core.RelationField{
				Name:          "server",
				CollectionId:  servers.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.DateField{Name: "timestamp", Required: true},
			&core.NumberField{Name: "cpu_percent"},
			&core.NumberField{Name: "mem_percent"},
			&core.NumberField{Name: "mem_used"},
			&core.NumberField{Name: "mem_total"},
			&core.NumberField{Name: "disk_used"},
			&core.NumberField{Name: "disk_total"},
			&core.NumberField{Name: "net_rx"},
			&core.NumberField{Name: "net_tx"},
			&core.NumberField{Name: "load_1m"},
			&core.NumberField{Name: "load_5m"},
			&core.NumberField{Name: "load_15m"},
			&core.AutodateField{Name: "created", OnCreate: true},
		)
		serverMetricsRaw.AddIndex("idx_server_metrics_raw_server", false, "server", "")
		serverMetricsRaw.AddIndex("idx_server_metrics_raw_timestamp", false, "timestamp", "")
		serverMetricsRaw.AddIndex("idx_server_metrics_raw_server_ts", false, "server, timestamp", "")
		if err := app.Save(serverMetricsRaw); err != nil {
			return err
		}

		// ── 8. server_metrics_hourly ────────────────────────
		serverMetricsHourly := core.NewBaseCollection("server_metrics_hourly")
		serverMetricsHourly.Fields.Add(
			&core.RelationField{
				Name:          "server",
				CollectionId:  servers.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.DateField{Name: "timestamp", Required: true},
			&core.NumberField{Name: "cpu_percent"},
			&core.NumberField{Name: "mem_percent"},
			&core.NumberField{Name: "mem_used"},
			&core.NumberField{Name: "mem_total"},
			&core.NumberField{Name: "disk_used"},
			&core.NumberField{Name: "disk_total"},
			&core.NumberField{Name: "net_rx"},
			&core.NumberField{Name: "net_tx"},
			&core.NumberField{Name: "load_1m"},
			&core.NumberField{Name: "load_5m"},
			&core.NumberField{Name: "load_15m"},
			&core.NumberField{Name: "samples", OnlyInt: true},
			&core.JSONField{Name: "stats"},
			&core.AutodateField{Name: "created", OnCreate: true},
		)
		serverMetricsHourly.AddIndex("idx_server_metrics_hourly_server", false, "server", "")
		serverMetricsHourly.AddIndex("idx_server_metrics_hourly_timestamp", false, "timestamp", "")
		serverMetricsHourly.AddIndex("idx_server_metrics_hourly_server_ts", false, "server, timestamp", "")
		if err := app.Save(serverMetricsHourly); err != nil {
			return err
		}

		// ── 9. server_metrics_daily ─────────────────────────
		serverMetricsDaily := core.NewBaseCollection("server_metrics_daily")
		serverMetricsDaily.Fields.Add(
			&core.RelationField{
				Name:          "server",
				CollectionId:  servers.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.DateField{Name: "timestamp", Required: true},
			&core.NumberField{Name: "cpu_percent"},
			&core.NumberField{Name: "mem_percent"},
			&core.NumberField{Name: "mem_used"},
			&core.NumberField{Name: "mem_total"},
			&core.NumberField{Name: "disk_used"},
			&core.NumberField{Name: "disk_total"},
			&core.NumberField{Name: "net_rx"},
			&core.NumberField{Name: "net_tx"},
			&core.NumberField{Name: "load_1m"},
			&core.NumberField{Name: "load_5m"},
			&core.NumberField{Name: "load_15m"},
			&core.NumberField{Name: "samples", OnlyInt: true},
			&core.JSONField{Name: "stats"},
			&core.AutodateField{Name: "created", OnCreate: true},
		)
		serverMetricsDaily.AddIndex("idx_server_metrics_daily_server", false, "server", "")
		serverMetricsDaily.AddIndex("idx_server_metrics_daily_timestamp", false, "timestamp", "")
		serverMetricsDaily.AddIndex("idx_server_metrics_daily_server_ts", false, "server, timestamp", "")
		if err := app.Save(serverMetricsDaily); err != nil {
			return err
		}

		// ── 10. container_metrics_raw ───────────────────────
		containerMetricsRaw := core.NewBaseCollection("container_metrics_raw")
		containerMetricsRaw.Fields.Add(
			&core.RelationField{
				Name:          "container",
				CollectionId:  containers.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.RelationField{
				Name:          "server",
				CollectionId:  servers.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.DateField{Name: "timestamp", Required: true},
			&core.NumberField{Name: "cpu_percent"},
			&core.NumberField{Name: "mem_used"},
			&core.NumberField{Name: "mem_limit"},
			&core.NumberField{Name: "net_rx"},
			&core.NumberField{Name: "net_tx"},
			&core.NumberField{Name: "block_read"},
			&core.NumberField{Name: "block_write"},
			&core.NumberField{Name: "pids", OnlyInt: true},
			&core.AutodateField{Name: "created", OnCreate: true},
		)
		containerMetricsRaw.AddIndex("idx_container_metrics_raw_container", false, "container", "")
		containerMetricsRaw.AddIndex("idx_container_metrics_raw_server", false, "server", "")
		containerMetricsRaw.AddIndex("idx_container_metrics_raw_timestamp", false, "timestamp", "")
		containerMetricsRaw.AddIndex("idx_container_metrics_raw_container_ts", false, "container, timestamp", "")
		if err := app.Save(containerMetricsRaw); err != nil {
			return err
		}

		// ── 11. container_metrics_hourly ────────────────────
		containerMetricsHourly := core.NewBaseCollection("container_metrics_hourly")
		containerMetricsHourly.Fields.Add(
			&core.RelationField{
				Name:          "container",
				CollectionId:  containers.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.RelationField{
				Name:          "server",
				CollectionId:  servers.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.DateField{Name: "timestamp", Required: true},
			&core.NumberField{Name: "cpu_percent"},
			&core.NumberField{Name: "mem_used"},
			&core.NumberField{Name: "mem_limit"},
			&core.NumberField{Name: "net_rx"},
			&core.NumberField{Name: "net_tx"},
			&core.NumberField{Name: "block_read"},
			&core.NumberField{Name: "block_write"},
			&core.NumberField{Name: "pids", OnlyInt: true},
			&core.NumberField{Name: "samples", OnlyInt: true},
			&core.JSONField{Name: "stats"},
			&core.AutodateField{Name: "created", OnCreate: true},
		)
		containerMetricsHourly.AddIndex("idx_container_metrics_hourly_container", false, "container", "")
		containerMetricsHourly.AddIndex("idx_container_metrics_hourly_server", false, "server", "")
		containerMetricsHourly.AddIndex("idx_container_metrics_hourly_timestamp", false, "timestamp", "")
		containerMetricsHourly.AddIndex("idx_container_metrics_hourly_container_ts", false, "container, timestamp", "")
		if err := app.Save(containerMetricsHourly); err != nil {
			return err
		}

		// ── 12. container_metrics_daily ─────────────────────
		containerMetricsDaily := core.NewBaseCollection("container_metrics_daily")
		containerMetricsDaily.Fields.Add(
			&core.RelationField{
				Name:          "container",
				CollectionId:  containers.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.RelationField{
				Name:          "server",
				CollectionId:  servers.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.DateField{Name: "timestamp", Required: true},
			&core.NumberField{Name: "cpu_percent"},
			&core.NumberField{Name: "mem_used"},
			&core.NumberField{Name: "mem_limit"},
			&core.NumberField{Name: "net_rx"},
			&core.NumberField{Name: "net_tx"},
			&core.NumberField{Name: "block_read"},
			&core.NumberField{Name: "block_write"},
			&core.NumberField{Name: "pids", OnlyInt: true},
			&core.NumberField{Name: "samples", OnlyInt: true},
			&core.JSONField{Name: "stats"},
			&core.AutodateField{Name: "created", OnCreate: true},
		)
		containerMetricsDaily.AddIndex("idx_container_metrics_daily_container", false, "container", "")
		containerMetricsDaily.AddIndex("idx_container_metrics_daily_server", false, "server", "")
		containerMetricsDaily.AddIndex("idx_container_metrics_daily_timestamp", false, "timestamp", "")
		containerMetricsDaily.AddIndex("idx_container_metrics_daily_container_ts", false, "container, timestamp", "")
		if err := app.Save(containerMetricsDaily); err != nil {
			return err
		}

		// ── 13. update_policies ─────────────────────────────
		updatePolicies := core.NewBaseCollection("update_policies")
		updatePolicies.Fields.Add(
			&core.RelationField{
				Name:          "container",
				CollectionId:  containers.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.BoolField{Name: "enabled"},
			&core.TextField{Name: "schedule", Max: 100},
			&core.BoolField{Name: "backup_before_update"},
			&core.BoolField{Name: "auto_restart"},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		updatePolicies.AddIndex("idx_update_policies_container", false, "container", "")
		if err := app.Save(updatePolicies); err != nil {
			return err
		}

		// ── 14. backup_policies ─────────────────────────────
		backupPolicies := core.NewBaseCollection("backup_policies")
		backupPolicies.Fields.Add(
			&core.TextField{Name: "volume", Required: true, Max: 255},
			&core.RelationField{
				Name:          "server",
				CollectionId:  servers.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.TextField{Name: "schedule", Max: 100},
			&core.NumberField{Name: "retention_days", OnlyInt: true},
			&core.TextField{Name: "s3_bucket", Max: 255},
			&core.TextField{Name: "s3_prefix", Max: 500},
			&core.BoolField{Name: "enabled"},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		backupPolicies.AddIndex("idx_backup_policies_server", false, "server", "")
		backupPolicies.AddIndex("idx_backup_policies_volume", false, "volume", "")
		if err := app.Save(backupPolicies); err != nil {
			return err
		}

		// ── 15. notification_channels ───────────────────────
		notificationChannels := core.NewBaseCollection("notification_channels")
		notificationChannels.Fields.Add(
			&core.TextField{Name: "name", Required: true, Max: 255},
			&core.SelectField{
				Name:      "type",
				Values:    []string{"whatsapp", "telegram", "webhook"},
				MaxSelect: 1,
				Required:  true,
			},
			&core.JSONField{Name: "config", Required: true},
			&core.BoolField{Name: "enabled"},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		if err := app.Save(notificationChannels); err != nil {
			return err
		}

		// ── 16. alert_rules ─────────────────────────────────
		alertRules := core.NewBaseCollection("alert_rules")
		alertRules.Fields.Add(
			&core.TextField{Name: "name", Required: true, Max: 255},
			&core.TextField{Name: "metric", Required: true, Max: 100},
			&core.SelectField{
				Name:      "operator",
				Values:    []string{"gt", "lt", "gte", "lte", "eq"},
				MaxSelect: 1,
				Required:  true,
			},
			&core.NumberField{Name: "threshold", Required: true},
			&core.NumberField{Name: "duration", OnlyInt: true},
			&core.SelectField{
				Name:      "target_type",
				Values:    []string{"server", "container"},
				MaxSelect: 1,
				Required:  true,
			},
			&core.TextField{Name: "target_id", Max: 50},
			&core.RelationField{
				Name:         "channels",
				CollectionId: notificationChannels.Id,
				MaxSelect:    50,
			},
			&core.BoolField{Name: "enabled"},
			&core.SelectField{
				Name:      "state",
				Values:    []string{"ok", "firing", "resolved"},
				MaxSelect: 1,
			},
			&core.DateField{Name: "last_triggered"},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		alertRules.AddIndex("idx_alert_rules_state", false, "state", "")
		alertRules.AddIndex("idx_alert_rules_enabled", false, "enabled", "")
		if err := app.Save(alertRules); err != nil {
			return err
		}

		// ── 17. backup_records ──────────────────────────────
		backupRecords := core.NewBaseCollection("backup_records")
		backupRecords.Fields.Add(
			&core.RelationField{
				Name:          "policy",
				CollectionId:  backupPolicies.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.TextField{Name: "volume_name", Max: 255},
			&core.RelationField{
				Name:          "server",
				CollectionId:  servers.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.TextField{Name: "s3_key", Max: 1000},
			&core.NumberField{Name: "size"},
			&core.SelectField{
				Name:      "status",
				Values:    []string{"pending", "running", "success", "failed"},
				MaxSelect: 1,
				Required:  true,
			},
			&core.DateField{Name: "started_at"},
			&core.DateField{Name: "completed_at"},
			&core.TextField{Name: "error"},
			&core.AutodateField{Name: "created", OnCreate: true},
		)
		backupRecords.AddIndex("idx_backup_records_policy", false, "policy", "")
		backupRecords.AddIndex("idx_backup_records_server", false, "server", "")
		backupRecords.AddIndex("idx_backup_records_status", false, "status", "")
		if err := app.Save(backupRecords); err != nil {
			return err
		}

		// ── 18. notification_logs ───────────────────────────
		notificationLogs := core.NewBaseCollection("notification_logs")
		notificationLogs.Fields.Add(
			&core.RelationField{
				Name:          "channel",
				CollectionId:  notificationChannels.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.RelationField{
				Name:         "alert_rule",
				CollectionId: alertRules.Id,
				MaxSelect:    1,
			},
			&core.SelectField{
				Name:      "status",
				Values:    []string{"sent", "failed", "retrying"},
				MaxSelect: 1,
				Required:  true,
			},
			&core.TextField{Name: "message"},
			&core.TextField{Name: "error"},
			&core.DateField{Name: "sent_at"},
			&core.NumberField{Name: "retries", OnlyInt: true},
			&core.AutodateField{Name: "created", OnCreate: true},
		)
		notificationLogs.AddIndex("idx_notification_logs_channel", false, "channel", "")
		notificationLogs.AddIndex("idx_notification_logs_status", false, "status", "")
		if err := app.Save(notificationLogs); err != nil {
			return err
		}

		// ── 19. audit_logs ──────────────────────────────────
		auditLogs := core.NewBaseCollection("audit_logs")
		auditLogs.Fields.Add(
			&core.TextField{Name: "action", Required: true, Max: 255},
			&core.TextField{Name: "user", Max: 255},
			&core.TextField{Name: "target_type", Max: 100},
			&core.TextField{Name: "target_id", Max: 50},
			&core.JSONField{Name: "details"},
			&core.TextField{Name: "ip_address", Max: 45},
			&core.DateField{Name: "timestamp", Required: true},
			&core.AutodateField{Name: "created", OnCreate: true},
		)
		auditLogs.AddIndex("idx_audit_logs_action", false, "action", "")
		auditLogs.AddIndex("idx_audit_logs_timestamp", false, "timestamp", "")
		auditLogs.AddIndex("idx_audit_logs_user", false, "user", "")
		if err := app.Save(auditLogs); err != nil {
			return err
		}

		// ── 20. terminal_sessions ───────────────────────────
		terminalSessions := core.NewBaseCollection("terminal_sessions")
		terminalSessions.Fields.Add(
			&core.RelationField{
				Name:          "container",
				CollectionId:  containers.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.RelationField{
				Name:          "server",
				CollectionId:  servers.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.TextField{Name: "user", Max: 255},
			&core.DateField{Name: "started_at"},
			&core.DateField{Name: "ended_at"},
			&core.NumberField{Name: "duration", OnlyInt: true},
			&core.AutodateField{Name: "created", OnCreate: true},
		)
		terminalSessions.AddIndex("idx_terminal_sessions_container", false, "container", "")
		terminalSessions.AddIndex("idx_terminal_sessions_server", false, "server", "")
		if err := app.Save(terminalSessions); err != nil {
			return err
		}

		// ── 21. settings ────────────────────────────────────
		settings := core.NewBaseCollection("settings")
		settings.Fields.Add(
			&core.TextField{Name: "key", Required: true, Max: 255},
			&core.JSONField{Name: "value"},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		settings.AddIndex("idx_settings_key", true, "key", "")
		if err := app.Save(settings); err != nil {
			return err
		}

		// ── 22. traefik_routes ──────────────────────────────
		traefikRoutes := core.NewBaseCollection("traefik_routes")
		traefikRoutes.Fields.Add(
			&core.RelationField{
				Name:          "server",
				CollectionId:  servers.Id,
				MaxSelect:     1,
				Required:      true,
				CascadeDelete: true,
			},
			&core.TextField{Name: "name", Required: true, Max: 255},
			&core.TextField{Name: "rule"},
			&core.TextField{Name: "service", Max: 255},
			&core.JSONField{Name: "entrypoints"},
			&core.BoolField{Name: "tls"},
			&core.JSONField{Name: "middlewares"},
			&core.TextField{Name: "provider", Max: 100},
			&core.TextField{Name: "status", Max: 50},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		traefikRoutes.AddIndex("idx_traefik_routes_server", false, "server", "")
		if err := app.Save(traefikRoutes); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		// down: delete in reverse order
		names := []string{
			"traefik_routes", "settings", "terminal_sessions",
			"audit_logs", "notification_logs", "backup_records",
			"alert_rules", "notification_channels",
			"backup_policies", "update_policies",
			"container_metrics_daily", "container_metrics_hourly", "container_metrics_raw",
			"server_metrics_daily", "server_metrics_hourly", "server_metrics_raw",
			"volumes", "networks", "images",
			"containers", "compose_stacks", "servers",
		}
		for _, name := range names {
			col, err := app.FindCollectionByNameOrId(name)
			if err == nil {
				if err := app.Delete(col); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
