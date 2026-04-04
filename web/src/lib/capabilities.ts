export const CAP_DOMAINS: { domain: string; caps: string[] }[] = [
	{ domain: "Servers", caps: ["servers.view", "servers.manage"] },
	{ domain: "Containers", caps: ["containers.view", "containers.manage"] },
	{ domain: "Images", caps: ["images.view", "images.manage"] },
	{ domain: "Compose", caps: ["compose.view", "compose.manage"] },
	{ domain: "Networks", caps: ["networks.manage"] },
	{ domain: "Volumes", caps: ["volumes.manage"] },
	{ domain: "Backups", caps: ["backups.view", "backups.manage"] },
	{ domain: "Alerts", caps: ["alerts.view", "alerts.manage"] },
	{ domain: "Notifications", caps: ["notifications.view", "notifications.manage"] },
	{ domain: "Health Checks", caps: ["health.view", "health.manage"] },
	{ domain: "Settings", caps: ["settings.view", "settings.manage"] },
	{ domain: "Users", caps: ["users.view", "users.manage"] },
	{ domain: "Audit", caps: ["audit.view"] },
	{ domain: "Traefik", caps: ["traefik.view"] },
	{ domain: "Events", caps: ["events.view"] },
	{ domain: "Batch", caps: ["batch.view", "batch.manage"] },
	{ domain: "Tags", caps: ["tags.view", "tags.manage"] },
];

export const ALL_CAPS = CAP_DOMAINS.flatMap((d) => d.caps);
