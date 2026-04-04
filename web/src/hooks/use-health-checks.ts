import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { getAccessToken } from "@/lib/api";
import type { HealthCheckRecord, HealthCheckLogRecord } from "@/types/pocketbase";

// ── Helpers ──────────────────────────────────────────────

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

// ── Queries ──────────────────────────────────────────────

export function useHealthChecks(serverId?: string) {
	const qs = serverId ? `?server_id=${serverId}` : "";
	return useQuery<HealthCheckRecord[]>({
		queryKey: ["health-checks", serverId ?? "all"],
		queryFn: () => fetchJson(`/api/dk/health-checks${qs}`, { headers: authHeaders() }),
		refetchInterval: 30_000,
	});
}

interface CheckLogsResult {
	items: HealthCheckLogRecord[];
	total_items: number;
	page: number;
	per_page: number;
}

export function useHealthCheckLogs(checkId: string, page = 1, perPage = 50) {
	return useQuery<CheckLogsResult>({
		queryKey: ["health-check-logs", checkId, page, perPage],
		queryFn: () =>
			fetchJson(`/api/dk/health-checks/${checkId}/logs?page=${page}&per_page=${perPage}`, {
				headers: authHeaders(),
			}),
		enabled: !!checkId,
	});
}

// ── Mutations ────────────────────────────────────────────

export interface CreateHealthCheckInput {
	name: string;
	container: string;
	server: string;
	url_path?: string;
	port: number;
	expected_status?: number;
	interval_seconds?: number;
	timeout_seconds?: number;
	failure_threshold?: number;
	success_threshold?: number;
	auto_restart?: boolean;
	max_restarts?: number;
	restart_window_minutes?: number;
	enabled: boolean;
}

export function useCreateHealthCheck() {
	const qc = useQueryClient();
	return useMutation<{ id: string }, Error, CreateHealthCheckInput>({
		mutationFn: (input) =>
			fetchJson("/api/dk/health-checks", {
				method: "POST",
				headers: authHeaders(true),
				body: JSON.stringify(input),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["health-checks"] });
			toast.success("Health check created");
		},
		onError: (err) => toast.error(`Create failed: ${err.message}`),
	});
}

export function useUpdateHealthCheck() {
	const qc = useQueryClient();
	return useMutation<{ status: string }, Error, { id: string; fields: Record<string, unknown> }>({
		mutationFn: ({ id, fields }) =>
			fetchJson(`/api/dk/health-checks/${id}`, {
				method: "PUT",
				headers: authHeaders(true),
				body: JSON.stringify(fields),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["health-checks"] });
			toast.success("Health check updated");
		},
		onError: (err) => toast.error(`Update failed: ${err.message}`),
	});
}

export function useDeleteHealthCheck() {
	const qc = useQueryClient();
	return useMutation<{ status: string }, Error, string>({
		mutationFn: (id) =>
			fetchJson(`/api/dk/health-checks/${id}`, {
				method: "DELETE",
				headers: authHeaders(),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["health-checks"] });
			toast.success("Health check deleted");
		},
		onError: (err) => toast.error(`Delete failed: ${err.message}`),
	});
}

export function useTriggerHealthCheck() {
	const qc = useQueryClient();
	return useMutation<{ status: string }, Error, string>({
		mutationFn: (id) =>
			fetchJson(`/api/dk/health-checks/${id}/trigger`, {
				method: "POST",
				headers: authHeaders(),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["health-checks"] });
			toast.success("Health check triggered");
		},
		onError: (err) => toast.error(`Trigger failed: ${err.message}`),
	});
}
