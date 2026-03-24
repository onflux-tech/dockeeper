import { z } from "zod/v4";

// ── Port mapping ────────────────────────────────────────────────────────────

const portSchema = z.object({
	host_port: z.string(),
	container_port: z.string().min(1, "Container port is required"),
	protocol: z.enum(["tcp", "udp"]),
});

// ── Volume mount ────────────────────────────────────────────────────────────

const volumeSchema = z.object({
	source: z.string(),
	target: z.string().min(1, "Mount path is required"),
	read_only: z.boolean(),
});

// ── Key-value entry (env & labels) ──────────────────────────────────────────

const kvSchema = z.object({
	key: z.string().min(1, "Key is required"),
	value: z.string(),
});

// ── Full wizard schema ─────────────────────────────────────────────────────

export const containerWizardSchema = z.object({
	// Step 1 — Image
	image: z.string().min(1, "Image is required"),
	auto_start: z.boolean(),

	// Step 2 — General
	server_id: z.string().min(1, "Server is required"),
	name: z
		.string()
		.min(1, "Name is required")
		.max(128, "Name must be 128 characters or less")
		.regex(/^[a-zA-Z0-9][a-zA-Z0-9._-]*$/, "Must start with alphanumeric, only [a-zA-Z0-9._-]"),
	network: z.string(),
	restart_policy: z.enum(["", "no", "always", "unless-stopped", "on-failure"]),
	command: z.string(),
	entrypoint: z.string(),

	// Step 3 — Ports
	ports: z.array(portSchema),

	// Step 4 — Volumes
	volumes: z.array(volumeSchema),

	// Step 5 — Env
	env: z.array(kvSchema),

	// Step 6 — Labels
	labels: z.array(kvSchema),

	// Step 7 — Resources
	cpu_limit: z.number().min(0),
	mem_limit: z.number().min(0), // stored in MB in the form
	pids_limit: z.number().min(0),
});

export type ContainerWizardData = z.infer<typeof containerWizardSchema>;

// ── Step metadata ───────────────────────────────────────────────────────────

export const STEPS = [
	{ label: "Image", fields: ["image", "auto_start"] },
	{ label: "General", fields: ["server_id", "name", "network", "restart_policy", "command", "entrypoint"] },
	{ label: "Ports", fields: ["ports"] },
	{ label: "Volumes", fields: ["volumes"] },
	{ label: "Env", fields: ["env"] },
	{ label: "Labels", fields: ["labels"] },
	{ label: "Resources", fields: ["cpu_limit", "mem_limit", "pids_limit"] },
	{ label: "Review", fields: [] },
] as const;

export type StepIndex = 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7;
