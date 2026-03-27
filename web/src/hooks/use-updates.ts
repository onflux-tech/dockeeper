import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { getAccessToken } from "@/lib/api";

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

// ── Types ────────────────────────────────────────────────

export interface CheckResult {
	container_id: string;
	container_name: string;
	server_id: string;
	image: string;
	current_digest?: string;
	remote_digest?: string;
	update_available: boolean;
	error?: string;
}

export interface UpdateHistoryRecord {
	id: string;
	container: string;
	server: string;
	old_image: string;
	new_image: string;
	status: string;
	started_at: string;
	completed_at: string;
	error: string;
	created: string;
}

export interface RegistryCredential {
	id: string;
	name: string;
	registry_host: string;
	created: string;
	updated: string;
}

// ── Update Check ─────────────────────────────────────────

export function useCheckUpdates() {
	return useMutation<{ total: number; results: CheckResult[] }, Error>({
		mutationFn: () =>
			fetchJson("/api/dk/updates/check", {
				headers: authHeaders(),
			}),
		onSuccess: (data) => {
			const available = data.results.filter((r) => r.update_available).length;
			if (available > 0) {
				toast.success(`${available} update(s) available`);
			} else {
				toast.info("All images are up to date");
			}
		},
		onError: (err) => {
			toast.error(`Update check failed: ${err.message}`);
		},
	});
}

// ── Update History ───────────────────────────────────────

export function useUpdateHistory(containerId?: string) {
	const params = new URLSearchParams();
	if (containerId) params.set("container_id", containerId);

	return useQuery<UpdateHistoryRecord[]>({
		queryKey: ["updates", "history", containerId ?? "all"],
		queryFn: () =>
			fetchJson(`/api/dk/updates/history?${params}`, {
				headers: authHeaders(),
			}),
	});
}

// ── Apply Update ─────────────────────────────────────────

export function useApplyUpdate() {
	const qc = useQueryClient();
	return useMutation<{ status: string; message: string }, Error, string>({
		mutationFn: (containerId) =>
			fetchJson(`/api/dk/updates/${containerId}/apply`, {
				method: "POST",
				headers: authHeaders(true),
			}),
		onSuccess: (_data, containerId) => {
			toast.success("Container updated successfully");
			qc.invalidateQueries({ queryKey: ["collection", "containers"] });
			qc.invalidateQueries({ queryKey: ["record", "containers", containerId] });
			qc.invalidateQueries({ queryKey: ["updates", "history"] });
		},
		onError: (err) => {
			toast.error(`Update failed: ${err.message}`);
		},
	});
}

// ── Registry Credentials ─────────────────────────────────

export function useRegistryCredentials() {
	return useQuery<RegistryCredential[]>({
		queryKey: ["registry-credentials"],
		queryFn: () =>
			fetchJson("/api/dk/registry-credentials", {
				headers: authHeaders(),
			}),
	});
}

export function useCreateRegistryCredential() {
	const qc = useQueryClient();
	return useMutation<
		RegistryCredential,
		Error,
		{ name: string; registry_host: string; username: string; password: string }
	>({
		mutationFn: (data) =>
			fetchJson("/api/dk/registry-credentials", {
				method: "POST",
				headers: authHeaders(true),
				body: JSON.stringify(data),
			}),
		onSuccess: () => {
			toast.success("Registry credential added");
			qc.invalidateQueries({ queryKey: ["registry-credentials"] });
		},
		onError: (err) => {
			toast.error(`Failed to add credential: ${err.message}`);
		},
	});
}

export function useDeleteRegistryCredential() {
	const qc = useQueryClient();
	return useMutation<{ status: string }, Error, string>({
		mutationFn: (id) =>
			fetchJson(`/api/dk/registry-credentials/${id}`, {
				method: "DELETE",
				headers: authHeaders(),
			}),
		onSuccess: () => {
			toast.success("Registry credential removed");
			qc.invalidateQueries({ queryKey: ["registry-credentials"] });
		},
		onError: (err) => {
			toast.error(`Failed to remove credential: ${err.message}`);
		},
	});
}
