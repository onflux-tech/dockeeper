import { useQuery } from "@tanstack/react-query";
import { getAccessToken } from "@/lib/api";
import type { TraefikRouteRecord } from "@/types/pocketbase";

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

export function useTraefikRoutes(serverId?: string) {
	const qs = serverId ? `?server_id=${serverId}` : "";
	return useQuery<TraefikRouteRecord[]>({
		queryKey: ["traefik-routes", serverId ?? "all"],
		queryFn: () =>
			fetchJson(`/api/dk/traefik/routes${qs}`, {
				headers: authHeaders(),
			}),
	});
}
