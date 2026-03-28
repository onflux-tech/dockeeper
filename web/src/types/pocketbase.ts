import type { RecordModel } from "pocketbase";

// ── Base record (PB auto-fields) ────────────────────────────────────────────

/** All PB records share these fields. */
export type BaseRecord = RecordModel;

// ── Collection record types ─────────────────────────────────────────────────

export interface ServerRecord extends BaseRecord {
	name: string;
	hostname: string;
	ip_address: string;
	os: string;
	arch: string;
	docker_version: string;
	agent_version: string;
	status: "online" | "offline" | "error" | "pending";
	orchestrator: "docker" | "podman" | "kubernetes";
	last_seen: string;
	enrolled_at: string;
	labels: unknown;
	cert_expires_at: string;
	enrollment_token: string;
	description: string;
}

export interface ContainerRecord extends BaseRecord {
	server: string;
	container_id: string;
	name: string;
	image: string;
	status: string;
	ports: unknown;
	labels: unknown;
	env_vars: unknown;
	volumes: unknown;
	networks: unknown;
	resource_limits: unknown;
	restart_policy: string;
	command: string;
	entrypoint: string;
	container_created_at: string;
	started_at: string;
	compose_stack: string;
	compose_project: string;
	compose_service: string;
	update_digest: string;
	last_update_check: string;
}

export interface ImageRecord extends BaseRecord {
	server: string;
	image_id: string;
	tags: string[];
	size: number;
	image_created_at: string;
	in_use: boolean;
}

export interface NetworkRecord extends BaseRecord {
	server: string;
	network_id: string;
	name: string;
	driver: string;
	scope: string;
	subnet: string;
	gateway: string;
	internal: boolean;
	labels: unknown;
}

export interface VolumeRecord extends BaseRecord {
	server: string;
	name: string;
	driver: string;
	mountpoint: string;
	size: number;
	labels: unknown;
	in_use: boolean;
}

export interface ComposeStackRecord extends BaseRecord {
	server: string;
	name: string;
	yaml_content: string;
	status: "running" | "stopped" | "partial";
	services_count: number;
	deployed_at: string;
}

export interface ServerMetricsRawRecord extends BaseRecord {
	server: string;
	timestamp: string;
	cpu_percent: number;
	mem_percent: number;
	mem_used: number;
	mem_total: number;
	disk_used: number;
	disk_total: number;
	net_rx: number;
	net_tx: number;
	load_1m: number;
	load_5m: number;
	load_15m: number;
}

export interface ServerMetricsAggRecord extends BaseRecord {
	server: string;
	timestamp: string;
	cpu_percent: number;
	mem_percent: number;
	mem_used: number;
	mem_total: number;
	disk_used: number;
	disk_total: number;
	net_rx: number;
	net_tx: number;
	load_1m: number;
	load_5m: number;
	load_15m: number;
	samples: number;
	stats: Record<string, number>;
}

export interface ContainerMetricsRawRecord extends BaseRecord {
	container: string;
	server: string;
	timestamp: string;
	cpu_percent: number;
	mem_used: number;
	mem_limit: number;
	net_rx: number;
	net_tx: number;
	block_read: number;
	block_write: number;
	pids: number;
}

export interface ContainerMetricsAggRecord extends BaseRecord {
	container: string;
	server: string;
	timestamp: string;
	cpu_percent: number;
	mem_used: number;
	mem_limit: number;
	net_rx: number;
	net_tx: number;
	block_read: number;
	block_write: number;
	pids: number;
	samples: number;
	stats: Record<string, number>;
}

export interface UpdatePolicyRecord extends BaseRecord {
	container: string;
	enabled: boolean;
	schedule: string;
	backup_before_update: boolean;
	auto_restart: boolean;
}

export interface BackupPolicyRecord extends BaseRecord {
	volume: string;
	server: string;
	schedule: string;
	retention_days: number;
	s3_bucket: string;
	s3_prefix: string;
	enabled: boolean;
}

export interface BackupRecordRecord extends BaseRecord {
	policy: string;
	volume_name: string;
	server: string;
	s3_key: string;
	size: number;
	status: "pending" | "running" | "success" | "failed";
	started_at: string;
	completed_at: string;
	error: string;
}

export interface AlertRuleRecord extends BaseRecord {
	name: string;
	rule_type: "metric" | "event";
	metric: string;
	operator: "gt" | "lt" | "gte" | "lte" | "eq";
	threshold: number;
	duration: number;
	target_type: "server" | "container";
	target_id: string;
	event_type: string;
	scope: string;
	active_delay: number;
	clear_delay: number;
	notification_cooldown: number;
	suppress_during_updates: boolean;
	template: string;
	channels: string[];
	enabled: boolean;
	state: "ok" | "firing" | "resolved";
	last_triggered: string;
	annotations: unknown;
}

export interface NotificationChannelRecord extends BaseRecord {
	name: string;
	type: "whatsapp" | "telegram" | "webhook" | "discord" | "slack" | "smtp" | "ntfy";
	config: unknown;
	enabled: boolean;
}

export interface BlackoutWindowRecord extends BaseRecord {
	server: string;
	name: string;
	start_time: string;
	end_time: string;
	recurring: boolean;
	weekdays: number[];
	time_start: string;
	time_end: string;
	enabled: boolean;
}

export interface NotificationLogRecord extends BaseRecord {
	channel: string;
	alert_rule: string;
	status: "sent" | "failed" | "retrying";
	message: string;
	error: string;
	sent_at: string;
	retries: number;
}

export interface AuditLogRecord extends BaseRecord {
	action: string;
	user: string;
	target_type: string;
	target_id: string;
	details: unknown;
	ip_address: string;
	timestamp: string;
}

export interface TerminalSessionRecord extends BaseRecord {
	container: string;
	server: string;
	user: string;
	started_at: string;
	ended_at: string;
	duration: number;
}

export interface SettingRecord extends BaseRecord {
	key: string;
	value: unknown;
}

export interface TraefikRouteRecord extends BaseRecord {
	server: string;
	name: string;
	rule: string;
	service: string;
	entrypoints: string[];
	tls: boolean;
	middlewares: unknown;
	provider: string;
	status: string;
}

export interface HealthCheckRecord extends BaseRecord {
	name: string;
	container: string;
	server: string;
	url_path: string;
	port: number;
	expected_status: number;
	interval_seconds: number;
	timeout_seconds: number;
	failure_threshold: number;
	success_threshold: number;
	auto_restart: boolean;
	max_restarts: number;
	restart_window_minutes: number;
	enabled: boolean;
	status: "healthy" | "unhealthy" | "unknown";
	consecutive_failures: number;
	last_check_at: string;
	next_check_at: string;
}

export interface HealthCheckLogRecord extends BaseRecord {
	check: string;
	status_code: number;
	response_time_ms: number;
	success: boolean;
	error: string;
	checked_at: string;
}

export interface BatchJobRecord extends BaseRecord {
	action: string;
	items: BatchItemRecord[];
	status: "pending" | "running" | "completed" | "cancelled";
	total: number;
	succeeded: number;
	failed: number;
	cancelled: number;
	created_by: string;
	started_at: string;
	completed_at: string;
}

export interface BatchItemRecord {
	container_id: string;
	server_id: string;
	status: "pending" | "running" | "success" | "failed" | "cancelled";
	error?: string;
	started_at?: string;
	completed_at?: string;
}

export interface TagRecord extends BaseRecord {
	name: string;
	color: string;
	resource_type: "container" | "server";
	resource_id: string;
}

// ── Collection name → record type map ───────────────────────────────────────

export interface Collections {
	servers: ServerRecord;
	containers: ContainerRecord;
	images: ImageRecord;
	networks: NetworkRecord;
	volumes: VolumeRecord;
	compose_stacks: ComposeStackRecord;
	server_metrics_raw: ServerMetricsRawRecord;
	server_metrics_hourly: ServerMetricsAggRecord;
	server_metrics_daily: ServerMetricsAggRecord;
	container_metrics_raw: ContainerMetricsRawRecord;
	container_metrics_hourly: ContainerMetricsAggRecord;
	container_metrics_daily: ContainerMetricsAggRecord;
	update_policies: UpdatePolicyRecord;
	backup_policies: BackupPolicyRecord;
	backup_records: BackupRecordRecord;
	alert_rules: AlertRuleRecord;
	notification_channels: NotificationChannelRecord;
	notification_logs: NotificationLogRecord;
	audit_logs: AuditLogRecord;
	terminal_sessions: TerminalSessionRecord;
	settings: SettingRecord;
	traefik_routes: TraefikRouteRecord;
	blackout_windows: BlackoutWindowRecord;
	health_checks: HealthCheckRecord;
	health_check_logs: HealthCheckLogRecord;
	batch_jobs: BatchJobRecord;
	tags: TagRecord;
}

/** All valid collection names. */
export type CollectionName = keyof Collections;
