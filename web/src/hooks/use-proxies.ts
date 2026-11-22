import { useQuery } from "@tanstack/react-query";
import { getAccessToken } from "@/lib/api";
import type { ProxyInstanceRecord, ProxyRouteRecord } from "@/types/pocketbase";

interface ProxyResult<T> {
	page: number;
	perPage: number;
	totalItems: number;
	totalPages: number;
	items: T[];
}

export interface UseProxiesOptions {
	page?: number;
	perPage?: number;
	server?: string;
	kind?: string;
	search?: string;
	enabled?: boolean;
}

function authHeaders(): Record<string, string> {
	const token = getAccessToken();
	const h: Record<string, string> = {};
	if (token) h["Authorization"] = token;
	return h;
}

async function fetchJson<T>(url: string): Promise<T> {
	const res = await fetch(url, { headers: authHeaders() });
	if (!res.ok) {
		const body = await res.json().catch(() => ({ error: res.statusText }));
		throw new Error(body.error || res.statusText);
	}
	return res.json();
}

function buildParams(options: UseProxiesOptions): string {
	const params = new URLSearchParams();
	params.set("page", String(options.page ?? 1));
	params.set("perPage", String(options.perPage ?? 25));
	if (options.server) params.set("server", options.server);
	if (options.kind) params.set("kind", options.kind);
	if (options.search) params.set("search", options.search);
	return params.toString();
}

export function useProxyInstances(options: UseProxiesOptions = {}) {
	const params = buildParams(options);
	return useQuery<ProxyResult<ProxyInstanceRecord>>({
		queryKey: ["proxy-instances", params],
		queryFn: () => fetchJson(`/api/dk/proxies/instances?${params}`),
		enabled: options.enabled !== false,
	});
}

export function useProxyRoutes(options: UseProxiesOptions = {}) {
	const params = buildParams(options);
	return useQuery<ProxyResult<ProxyRouteRecord>>({
		queryKey: ["proxy-routes", params],
		queryFn: () => fetchJson(`/api/dk/proxies/routes?${params}`),
		enabled: options.enabled !== false,
	});
}
