import { useQuery } from "@tanstack/react-query";
import { getAccessToken } from "@/lib/api";

export interface EventRecord {
	id: string;
	event_type: string;
	severity: "info" | "warning" | "critical";
	server_id: string;
	server_name: string;
	container_id: string;
	container_name: string;
	details: Record<string, unknown>;
	timestamp: string;
	created: string;
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

export interface EventFilters {
	type?: string;
	severity?: string;
	server_id?: string;
	from?: string;
	to?: string;
	limit?: number;
	offset?: number;
}

export function useEvents(filters: EventFilters = {}) {
	const qs = new URLSearchParams();
	if (filters.type) qs.set("type", filters.type);
	if (filters.severity) qs.set("severity", filters.severity);
	if (filters.server_id) qs.set("server_id", filters.server_id);
	if (filters.from) qs.set("from", filters.from);
	if (filters.to) qs.set("to", filters.to);
	if (filters.limit) qs.set("limit", String(filters.limit));
	if (filters.offset) qs.set("offset", String(filters.offset));

	const queryStr = qs.toString();
	const url = `/api/dk/events${queryStr ? `?${queryStr}` : ""}`;

	return useQuery<EventRecord[]>({
		queryKey: ["events", filters],
		queryFn: () => fetchJson(url, { headers: authHeaders() }),
		refetchInterval: 5000,
	});
}
