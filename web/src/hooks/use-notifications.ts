import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { getAccessToken } from "@/lib/api";
import type {
	NotificationChannelRecord,
	NotificationLogRecord,
} from "@/types/pocketbase";

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

export function useNotificationChannels() {
	return useQuery<NotificationChannelRecord[]>({
		queryKey: ["notification-channels"],
		queryFn: () =>
			fetchJson("/api/dk/notifications/channels", {
				headers: authHeaders(),
			}),
	});
}

export function useNotificationLogs(
	alertRuleId?: string,
	channelId?: string,
) {
	const params = new URLSearchParams();
	if (alertRuleId) params.set("alert_rule_id", alertRuleId);
	else if (channelId) params.set("channel_id", channelId);
	const qs = params.toString() ? `?${params.toString()}` : "";

	return useQuery<NotificationLogRecord[]>({
		queryKey: ["notification-logs", alertRuleId ?? "", channelId ?? ""],
		queryFn: () =>
			fetchJson(`/api/dk/notifications/logs${qs}`, {
				headers: authHeaders(),
			}),
		refetchInterval: 15_000,
	});
}

// ── Mutations ────────────────────────────────────────────

interface CreateChannelInput {
	name: string;
	type: string;
	config: Record<string, unknown>;
	enabled: boolean;
}

export function useCreateChannel() {
	const qc = useQueryClient();
	return useMutation<{ id: string }, Error, CreateChannelInput>({
		mutationFn: (input) =>
			fetchJson("/api/dk/notifications/channels", {
				method: "POST",
				headers: authHeaders(true),
				body: JSON.stringify(input),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["notification-channels"] });
			toast.success("Notification channel created");
		},
		onError: (err) => toast.error(`Create failed: ${err.message}`),
	});
}

interface UpdateChannelInput {
	id: string;
	fields: Record<string, unknown>;
}

export function useUpdateChannel() {
	const qc = useQueryClient();
	return useMutation<{ status: string }, Error, UpdateChannelInput>({
		mutationFn: ({ id, fields }) =>
			fetchJson(`/api/dk/notifications/channels/${id}`, {
				method: "PUT",
				headers: authHeaders(true),
				body: JSON.stringify(fields),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["notification-channels"] });
			toast.success("Channel updated");
		},
		onError: (err) => toast.error(`Update failed: ${err.message}`),
	});
}

export function useDeleteChannel() {
	const qc = useQueryClient();
	return useMutation<{ status: string }, Error, string>({
		mutationFn: (id) =>
			fetchJson(`/api/dk/notifications/channels/${id}`, {
				method: "DELETE",
				headers: authHeaders(),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["notification-channels"] });
			toast.success("Channel deleted");
		},
		onError: (err) => toast.error(`Delete failed: ${err.message}`),
	});
}

export function useTestChannel() {
	return useMutation<{ status: string }, Error, string>({
		mutationFn: (id) =>
			fetchJson(`/api/dk/notifications/channels/${id}/test`, {
				method: "POST",
				headers: authHeaders(),
			}),
		onSuccess: () => toast.success("Test message sent"),
		onError: (err) => toast.error(`Test failed: ${err.message}`),
	});
}
