import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { getAccessToken } from "@/lib/api";
import type {
	BackupPolicyRecord,
	BackupRecordRecord,
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

export function useBackupPolicies(serverId?: string) {
	const qs = serverId ? `?server_id=${serverId}` : "";
	return useQuery<BackupPolicyRecord[]>({
		queryKey: ["backup-policies", serverId ?? "all"],
		queryFn: () =>
			fetchJson(`/api/dk/backups/policies${qs}`, {
				headers: authHeaders(),
			}),
	});
}

export function useBackupRecords(policyId?: string, serverId?: string) {
	const params = new URLSearchParams();
	if (policyId) params.set("policy_id", policyId);
	else if (serverId) params.set("server_id", serverId);
	const qs = params.toString() ? `?${params.toString()}` : "";

	return useQuery<BackupRecordRecord[]>({
		queryKey: ["backup-records", policyId ?? "", serverId ?? ""],
		queryFn: () =>
			fetchJson(`/api/dk/backups/records${qs}`, {
				headers: authHeaders(),
			}),
		refetchInterval: 15_000,
	});
}

// ── Mutations ────────────────────────────────────────────

interface CreatePolicyInput {
	server_id: string;
	volume: string;
	schedule: string;
	retention_days: number;
	s3_bucket: string;
	s3_prefix: string;
	enabled: boolean;
}

export function useCreatePolicy() {
	const qc = useQueryClient();
	return useMutation<{ id: string }, Error, CreatePolicyInput>({
		mutationFn: (input) =>
			fetchJson("/api/dk/backups/policies", {
				method: "POST",
				headers: authHeaders(true),
				body: JSON.stringify(input),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["backup-policies"] });
			toast.success("Backup policy created");
		},
		onError: (err) => toast.error(`Create failed: ${err.message}`),
	});
}

interface UpdatePolicyInput {
	id: string;
	fields: Record<string, unknown>;
}

export function useUpdatePolicy() {
	const qc = useQueryClient();
	return useMutation<{ status: string }, Error, UpdatePolicyInput>({
		mutationFn: ({ id, fields }) =>
			fetchJson(`/api/dk/backups/policies/${id}`, {
				method: "PUT",
				headers: authHeaders(true),
				body: JSON.stringify(fields),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["backup-policies"] });
			toast.success("Policy updated");
		},
		onError: (err) => toast.error(`Update failed: ${err.message}`),
	});
}

export function useDeletePolicy() {
	const qc = useQueryClient();
	return useMutation<{ status: string }, Error, string>({
		mutationFn: (id) =>
			fetchJson(`/api/dk/backups/policies/${id}`, {
				method: "DELETE",
				headers: authHeaders(),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["backup-policies"] });
			void qc.invalidateQueries({ queryKey: ["backup-records"] });
			toast.success("Policy deleted");
		},
		onError: (err) => toast.error(`Delete failed: ${err.message}`),
	});
}

export function useTriggerBackup() {
	const qc = useQueryClient();
	return useMutation<
		{ record_id: string; status: string },
		Error,
		string
	>({
		mutationFn: (policyId) =>
			fetchJson(`/api/dk/backups/policies/${policyId}/trigger`, {
				method: "POST",
				headers: authHeaders(),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["backup-records"] });
			toast.success("Backup triggered");
		},
		onError: (err) => toast.error(`Trigger failed: ${err.message}`),
	});
}

export function useRestoreBackup() {
	const qc = useQueryClient();
	return useMutation<{ status: string }, Error, string>({
		mutationFn: (recordId) =>
			fetchJson(`/api/dk/backups/records/${recordId}/restore`, {
				method: "POST",
				headers: authHeaders(),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["backup-records"] });
			toast.success("Restore started");
		},
		onError: (err) => toast.error(`Restore failed: ${err.message}`),
	});
}
