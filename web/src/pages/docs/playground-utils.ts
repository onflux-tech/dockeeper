import type { ApiEndpoint, ApiParameter, HttpMethod } from "@/types/docs";

// ── Path parameter parsing ──────────────────────────────────────────────────

/** Extract parameter names from a path template, e.g. "/api/dk/containers/{id}/start" → ["id"] */
export function parsePathParams(path: string): string[] {
	const matches = path.matchAll(/\{(\w+)\}/g);
	return [...matches].map((m) => m[1]);
}

/** Replace {param} placeholders with actual values */
export function resolvePath(path: string, params: Record<string, string>): string {
	return path.replace(/\{(\w+)\}/g, (_, key) => params[key] || `{${key}}`);
}

// ── URL builder ─────────────────────────────────────────────────────────────

export function buildRequestUrl(
	origin: string,
	path: string,
	pathParams: Record<string, string>,
	queryParams: Record<string, string>,
): string {
	const resolved = resolvePath(path, pathParams);
	const url = new URL(resolved, origin);

	for (const [key, value] of Object.entries(queryParams)) {
		if (value) url.searchParams.set(key, value);
	}

	return url.toString();
}

// ── Body template generation ────────────────────────────────────────────────

function defaultForType(type: string): unknown {
	const t = type.toLowerCase();
	if (t === "number" || t === "integer" || t === "int") return 0;
	if (t === "boolean" || t === "bool") return false;
	if (t.startsWith("array") || t.startsWith("[]")) return [];
	if (t === "object") return {};
	return "";
}

export function generateBodyTemplate(fields: ApiParameter[]): string {
	const obj: Record<string, unknown> = {};
	for (const f of fields) {
		obj[f.name] = f.example ?? defaultForType(f.type);
	}
	return JSON.stringify(obj, null, 2);
}

// ── cURL generation ─────────────────────────────────────────────────────────

export function generateCurl(
	method: HttpMethod,
	url: string,
	headers: Record<string, string>,
	body?: string,
): string {
	const parts: string[] = ["curl"];

	if (method !== "GET") {
		parts.push(`-X ${method}`);
	}

	parts.push(`'${url}'`);

	for (const [key, value] of Object.entries(headers)) {
		if (value) parts.push(`\\\n  -H '${key}: ${value}'`);
	}

	if (body && method !== "GET" && method !== "DELETE") {
		parts.push(`\\\n  -d '${body}'`);
	}

	return parts.join(" ");
}

// ── JavaScript code generation ──────────────────────────────────────────────

export function generateJsSnippet(
	method: HttpMethod,
	url: string,
	headers: Record<string, string>,
	body?: string,
): string {
	const opts: string[] = [];

	if (method !== "GET") {
		opts.push(`  method: "${method}",`);
	}

	const hdr: string[] = [];
	for (const [key, value] of Object.entries(headers)) {
		if (value) hdr.push(`    "${key}": "${value}",`);
	}
	if (hdr.length > 0) {
		opts.push(`  headers: {\n${hdr.join("\n")}\n  },`);
	}

	if (body && method !== "GET" && method !== "DELETE") {
		opts.push(`  body: JSON.stringify(${body}),`);
	}

	if (opts.length === 0) {
		return `const res = await fetch("${url}");\nconst data = await res.json();`;
	}

	return `const res = await fetch("${url}", {\n${opts.join("\n")}\n});\nconst data = await res.json();`;
}

// ── Quick send eligibility ──────────────────────────────────────────────────

export function isQuickEligible(endpoint: ApiEndpoint): boolean {
	return (
		!endpoint.streaming &&
		!endpoint.websocket &&
		!endpoint.requestBody &&
		(endpoint.parameters?.length ?? 0) <= 1
	);
}

// ── Resource type detection from endpoint ───────────────────────────────────

const PATH_RESOURCE_MAP: [RegExp, string][] = [
	[/^\/api\/dk\/servers/, "servers"],
	[/^\/api\/dk\/containers/, "containers"],
	[/^\/api\/dk\/images/, "images"],
	[/^\/api\/dk\/compose/, "compose"],
	[/^\/api\/dk\/networks/, "networks"],
	[/^\/api\/dk\/volumes/, "volumes"],
	[/^\/api\/dk\/alerts/, "alerts"],
	[/^\/api\/dk\/backups\/policies/, "backup-policies"],
	[/^\/api\/dk\/backups\/records/, "backup-records"],
	[/^\/api\/dk\/health-checks/, "health-checks"],
	[/^\/api\/dk\/notifications\/channels/, "notification-channels"],
	[/^\/api\/dk\/users/, "users"],
	[/^\/api\/dk\/groups/, "groups"],
	[/^\/api\/dk\/api-keys/, "api-keys"],
	[/^\/api\/dk\/tags/, "tags"],
	[/^\/api\/dk\/batch/, "batch"],
];

export function detectResourceType(path: string): string | null {
	for (const [regex, type] of PATH_RESOURCE_MAP) {
		if (regex.test(path)) return type;
	}
	return null;
}

// ── Status code color ───────────────────────────────────────────────────────

export function statusColor(status: number): string {
	if (status >= 200 && status < 300) return "text-green-400 border-green-400/30 bg-green-400/10";
	if (status >= 300 && status < 400) return "text-blue-400 border-blue-400/30 bg-blue-400/10";
	if (status >= 400 && status < 500) return "text-amber-400 border-amber-400/30 bg-amber-400/10";
	return "text-red-400 border-red-400/30 bg-red-400/10";
}

// ── Response size formatting ────────────────────────────────────────────────

export function formatBytes(bytes: number): string {
	if (bytes < 1024) return `${bytes} B`;
	if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
	return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export const MAX_DISPLAY_BYTES = 100 * 1024; // 100KB
