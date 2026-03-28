import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { getAccessToken } from "@/lib/api";

function authHeaders(json = false): Record<string, string> {
	const token = getAccessToken();
	const h: Record<string, string> = {};
	if (token) h["Authorization"] = token;
	if (json) h["Content-Type"] = "application/json";
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

// ── Types ────────────────────────────────────────────────

interface Widget {
	id: string;
	x: number;
	y: number;
	w: number;
	h: number;
}

export interface DashboardLayout {
	widgets: Widget[];
}

// ── Hooks ────────────────────────────────────────────────

export function useDashboardLayout() {
	return useQuery<DashboardLayout>({
		queryKey: ["dashboard-layout"],
		queryFn: () => fetchJson("/api/dk/dashboard/layout", { headers: authHeaders() }),
	});
}

export function useSaveDashboardLayout() {
	const qc = useQueryClient();
	return useMutation<{ status: string }, Error, DashboardLayout>({
		mutationFn: (layout) =>
			fetchJson("/api/dk/dashboard/layout", {
				method: "PUT",
				headers: authHeaders(true),
				body: JSON.stringify(layout),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["dashboard-layout"] });
		},
	});
}

export function useResetDashboardLayout() {
	const qc = useQueryClient();
	return useMutation<DashboardLayout, Error, void>({
		mutationFn: () =>
			fetchJson("/api/dk/dashboard/layout", {
				method: "DELETE",
				headers: authHeaders(),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["dashboard-layout"] });
		},
	});
}
