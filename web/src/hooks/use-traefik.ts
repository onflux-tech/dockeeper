import { useQuery } from "@tanstack/react-query";
import { getAccessToken } from "@/lib/api";
import type { TraefikRouteRecord } from "@/types/pocketbase";

interface TraefikRoutesResult {
	page: number;
	perPage: number;
	totalItems: number;
	totalPages: number;
	items: TraefikRouteRecord[];
}

export interface UseTraefikRoutesOptions {
	page?: number;
	perPage?: number;
	sort?: string;
	serverId?: string;
	search?: string;
	enabled?: boolean;
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

export function useTraefikRoutes(options: UseTraefikRoutesOptions = {}) {
	const { page = 1, perPage = 25, sort = "name", serverId, search, enabled = true } = options;

	const params = new URLSearchParams();
	params.set("page", String(page));
	params.set("perPage", String(perPage));
	params.set("sort", sort);
	if (serverId) params.set("server_id", serverId);
	if (search) params.set("search", search);

	return useQuery<TraefikRoutesResult>({
		queryKey: ["traefik-routes", { page, perPage, sort, serverId, search }],
		queryFn: () =>
			fetchJson(`/api/dk/traefik/routes?${params.toString()}`, {
				headers: authHeaders(),
			}),
		enabled,
	});
}
