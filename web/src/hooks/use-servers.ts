import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { getAccessToken } from "@/lib/api";
import type { ServerRecord } from "@/types/pocketbase";

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

export function useServers() {
	return useQuery<ServerRecord[]>({
		queryKey: ["servers"],
		queryFn: () => fetchJson("/api/dk/servers", { headers: authHeaders() }),
	});
}

interface ProvisionResult {
	id: string;
	name: string;
	enrollment_token: string;
	status: string;
}

export function useProvisionServer() {
	const qc = useQueryClient();
	return useMutation<ProvisionResult, Error, { name: string; description?: string }>({
		mutationFn: (body) =>
			fetchJson("/api/dk/servers/provision", {
				method: "POST",
				headers: { "Content-Type": "application/json", ...authHeaders() },
				body: JSON.stringify(body),
			}),
		onSuccess: () => qc.invalidateQueries({ queryKey: ["servers"] }),
	});
}

interface InstallScriptResult {
	mode: string;
	command: string;
}

export function useInstallScript(serverId: string, mode: "docker" | "binary") {
	return useQuery<InstallScriptResult>({
		queryKey: ["install-script", serverId, mode],
		queryFn: () =>
			fetchJson(`/api/dk/servers/${serverId}/install-script?mode=${mode}`, {
				headers: authHeaders(),
			}),
		enabled: !!serverId,
	});
}

export function useDeleteServer() {
	const qc = useQueryClient();
	return useMutation<unknown, Error, string>({
		mutationFn: (id) =>
			fetchJson(`/api/dk/servers/${id}`, {
				method: "DELETE",
				headers: authHeaders(),
			}),
		onSuccess: () => qc.invalidateQueries({ queryKey: ["servers"] }),
	});
}

export function useUpdateServer() {
	const qc = useQueryClient();
	return useMutation<
		{ id: string; name: string; description: string; view_scope: string },
		Error,
		{ id: string; name: string; description: string; view_scope: string }
	>({
		mutationFn: ({ id, ...body }) =>
			fetchJson(`/api/dk/servers/${id}`, {
				method: "PATCH",
				headers: { "Content-Type": "application/json", ...authHeaders() },
				body: JSON.stringify(body),
			}),
		onSuccess: () => {
			qc.invalidateQueries({ queryKey: ["servers"] });
			qc.invalidateQueries({ queryKey: ["collection", "servers"] });
			toast.success("Server updated");
		},
		onError: (err) => {
			toast.error(err.message || "Failed to update server");
		},
	});
}

export function useQuickInstallCommand() {
	return useQuery<{ command: string }>({
		queryKey: ["quick-install-command"],
		queryFn: () =>
			fetchJson("/api/dk/servers/quick-install", {
				headers: authHeaders(),
			}),
	});
}
