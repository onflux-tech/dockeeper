import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { getAccessToken } from "@/lib/api";
import { toast } from "sonner";

// ── Types ────────────────────────────────────────────

export interface Tag {
	id: string;
	name: string;
	color: string;
	resource_type: "container" | "server";
	resource_id: string;
	created: string;
}

export interface TagNameCount {
	name: string;
	color: string;
	count: number;
}

// ── Helpers ──────────────────────────────────────────

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

// ── Hooks ────────────────────────────────────────────

export function useTags(resourceType?: string) {
	return useQuery<{ items: Tag[] }>({
		queryKey: ["tags", resourceType],
		queryFn: () => {
			const params = resourceType ? `?resource_type=${resourceType}` : "";
			return fetchJson(`/api/dk/tags${params}`, { headers: authHeaders() });
		},
	});
}

export function useTagNames(resourceType: string = "container") {
	return useQuery<{ items: TagNameCount[] }>({
		queryKey: ["tag-names", resourceType],
		queryFn: () =>
			fetchJson(`/api/dk/tags/names?resource_type=${resourceType}`, {
				headers: authHeaders(),
			}),
	});
}

export function useApplyTag() {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: (data: {
			name: string;
			color?: string;
			resource_type: string;
			resource_id: string;
		}) =>
			fetchJson("/api/dk/tags", {
				method: "POST",
				headers: authHeaders(true),
				body: JSON.stringify(data),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["tags"] });
			void qc.invalidateQueries({ queryKey: ["tag-names"] });
			void qc.invalidateQueries({ queryKey: ["collection"] });
			toast.success("Tag applied");
		},
		onError: (err: Error) => toast.error(err.message),
	});
}

export function useRemoveTag() {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: (id: string) =>
			fetchJson(`/api/dk/tags/${id}`, {
				method: "DELETE",
				headers: authHeaders(),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["tags"] });
			void qc.invalidateQueries({ queryKey: ["tag-names"] });
			void qc.invalidateQueries({ queryKey: ["collection"] });
			toast.success("Tag removed");
		},
		onError: (err: Error) => toast.error(err.message),
	});
}

export function useBulkApplyTag() {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: (data: {
			tag: string;
			color?: string;
			resource_type: string;
			resource_ids: string[];
		}) =>
			fetchJson("/api/dk/tags/bulk", {
				method: "POST",
				headers: authHeaders(true),
				body: JSON.stringify(data),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["tags"] });
			void qc.invalidateQueries({ queryKey: ["tag-names"] });
			void qc.invalidateQueries({ queryKey: ["collection"] });
			toast.success("Tags applied in bulk");
		},
		onError: (err: Error) => toast.error(err.message),
	});
}

export function useBulkRemoveTag() {
	const qc = useQueryClient();
	return useMutation({
		mutationFn: (data: { tag: string; resource_type: string; resource_ids: string[] }) =>
			fetchJson("/api/dk/tags/bulk", {
				method: "DELETE",
				headers: authHeaders(true),
				body: JSON.stringify(data),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["tags"] });
			void qc.invalidateQueries({ queryKey: ["tag-names"] });
			void qc.invalidateQueries({ queryKey: ["collection"] });
			toast.success("Tags removed in bulk");
		},
		onError: (err: Error) => toast.error(err.message),
	});
}
