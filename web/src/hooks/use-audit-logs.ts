import { useQuery } from "@tanstack/react-query";
import { getAccessToken } from "@/lib/api";

interface AuditLogRecord {
	id: string;
	action: string;
	user: string;
	target_type: string;
	target_id: string;
	details: Record<string, unknown>;
	ip_address: string;
	timestamp: string;
	created: string;
}

interface AuditLogListResult {
	page: number;
	perPage: number;
	totalItems: number;
	totalPages: number;
	items: AuditLogRecord[];
}

function authHeaders(): Record<string, string> {
	const token = getAccessToken();
	const h: Record<string, string> = {};
	if (token) h["Authorization"] = token;
	return h;
}

async function fetchJson<T>(url: string, init?: RequestInit): Promise<T> {
	const res = await fetch(url, init);
	if (!res.ok) {
		const body = await res.json().catch(() => ({ error: res.statusText }));
		throw new Error(body.error || res.statusText);
	}
	return res.json();
}

export interface AuditLogFilters {
	page?: number;
	perPage?: number;
	action?: string;
	target_type?: string;
	search?: string;
	from?: string;
	to?: string;
	server_id?: string;
}

export function useAuditLogs(filters: AuditLogFilters = {}) {
	const qs = new URLSearchParams();
	if (filters.page) qs.set("page", String(filters.page));
	if (filters.perPage) qs.set("perPage", String(filters.perPage));
	if (filters.action) qs.set("action", filters.action);
	if (filters.target_type) qs.set("target_type", filters.target_type);
	if (filters.search) qs.set("search", filters.search);
	if (filters.from) qs.set("from", filters.from);
	if (filters.to) qs.set("to", filters.to);
	if (filters.server_id) qs.set("server_id", filters.server_id);

	const queryStr = qs.toString();
	const url = `/api/dk/audit-logs${queryStr ? `?${queryStr}` : ""}`;

	return useQuery<AuditLogListResult>({
		queryKey: ["audit-logs", filters],
		queryFn: () => fetchJson(url, { headers: authHeaders() }),
	});
}

export type { AuditLogRecord, AuditLogListResult };
