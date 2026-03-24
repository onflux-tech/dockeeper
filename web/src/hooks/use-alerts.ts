import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { getAccessToken } from "@/lib/api";
import type { AlertRuleRecord } from "@/types/pocketbase";

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

export function useAlertRules(serverId?: string) {
	const qs = serverId ? `?server_id=${serverId}` : "";
	return useQuery<AlertRuleRecord[]>({
		queryKey: ["alert-rules", serverId ?? "all"],
		queryFn: () =>
			fetchJson(`/api/dk/alerts${qs}`, { headers: authHeaders() }),
	});
}

// ── Mutations ────────────────────────────────────────────

interface CreateRuleInput {
	name: string;
	metric: string;
	operator: string;
	threshold: number;
	duration: number;
	target_type: string;
	target_id: string;
	channels: string[];
	enabled: boolean;
}

export function useCreateRule() {
	const qc = useQueryClient();
	return useMutation<{ id: string }, Error, CreateRuleInput>({
		mutationFn: (input) =>
			fetchJson("/api/dk/alerts", {
				method: "POST",
				headers: authHeaders(true),
				body: JSON.stringify(input),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["alert-rules"] });
			toast.success("Alert rule created");
		},
		onError: (err) => toast.error(`Create failed: ${err.message}`),
	});
}

interface UpdateRuleInput {
	id: string;
	fields: Record<string, unknown>;
}

export function useUpdateRule() {
	const qc = useQueryClient();
	return useMutation<{ status: string }, Error, UpdateRuleInput>({
		mutationFn: ({ id, fields }) =>
			fetchJson(`/api/dk/alerts/${id}`, {
				method: "PUT",
				headers: authHeaders(true),
				body: JSON.stringify(fields),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["alert-rules"] });
			toast.success("Alert rule updated");
		},
		onError: (err) => toast.error(`Update failed: ${err.message}`),
	});
}

export function useDeleteRule() {
	const qc = useQueryClient();
	return useMutation<{ status: string }, Error, string>({
		mutationFn: (id) =>
			fetchJson(`/api/dk/alerts/${id}`, {
				method: "DELETE",
				headers: authHeaders(),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["alert-rules"] });
			toast.success("Alert rule deleted");
		},
		onError: (err) => toast.error(`Delete failed: ${err.message}`),
	});
}

export function useToggleRule() {
	const qc = useQueryClient();
	return useMutation<{ enabled: boolean }, Error, string>({
		mutationFn: (id) =>
			fetchJson(`/api/dk/alerts/${id}/toggle`, {
				method: "POST",
				headers: authHeaders(),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["alert-rules"] });
		},
		onError: (err) => toast.error(`Toggle failed: ${err.message}`),
	});
}
