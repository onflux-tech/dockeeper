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

export interface User {
	id: string;
	email: string;
	created: string;
	updated: string;
}

export interface Group {
	id: string;
	name: string;
	description: string;
	capabilities: string[];
	is_preset: boolean;
	created: string;
}

export interface GroupMember {
	id: string;
	group: string;
	user: string;
	created: string;
}

export interface APIKey {
	id: string;
	name: string;
	key_prefix: string;
	user: string;
	capabilities: string[];
	enabled: boolean;
	last_used_at: string;
	expires_at: string;
	created: string;
}

export interface CreatedAPIKey {
	id: string;
	key: string;
	key_prefix: string;
	name: string;
}

// ── Users Queries ────────────────────────────────────────

export function useUsers() {
	return useQuery<User[]>({
		queryKey: ["users"],
		queryFn: () => fetchJson("/api/dk/users", { headers: authHeaders() }),
	});
}

export function useCreateUser() {
	const qc = useQueryClient();
	return useMutation<{ id: string; email: string }, Error, { email: string; password: string }>({
		mutationFn: (input) =>
			fetchJson("/api/dk/users", {
				method: "POST",
				headers: authHeaders(true),
				body: JSON.stringify(input),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["users"] });
			toast.success("User created");
		},
		onError: (err) => toast.error(err.message),
	});
}

export function useUpdateUser() {
	const qc = useQueryClient();
	return useMutation<
		{ id: string; email: string },
		Error,
		{ id: string; email?: string; password?: string }
	>({
		mutationFn: ({ id, ...body }) =>
			fetchJson(`/api/dk/users/${id}`, {
				method: "PATCH",
				headers: authHeaders(true),
				body: JSON.stringify(body),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["users"] });
			toast.success("User updated");
		},
		onError: (err) => toast.error(err.message),
	});
}

export function useDeleteUser() {
	const qc = useQueryClient();
	return useMutation<void, Error, string>({
		mutationFn: (id) =>
			fetchJson(`/api/dk/users/${id}`, {
				method: "DELETE",
				headers: authHeaders(),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["users"] });
			toast.success("User deleted");
		},
		onError: (err) => toast.error(err.message),
	});
}

// ── Groups Queries ───────────────────────────────────────

export function useGroups() {
	return useQuery<Group[]>({
		queryKey: ["groups"],
		queryFn: () => fetchJson("/api/dk/groups", { headers: authHeaders() }),
	});
}

export function useCreateGroup() {
	const qc = useQueryClient();
	return useMutation<Group, Error, { name: string; description: string; capabilities: string[] }>({
		mutationFn: (input) =>
			fetchJson("/api/dk/groups", {
				method: "POST",
				headers: authHeaders(true),
				body: JSON.stringify(input),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["groups"] });
			toast.success("Group created");
		},
		onError: (err) => toast.error(err.message),
	});
}

export function useUpdateGroup() {
	const qc = useQueryClient();
	return useMutation<
		Group,
		Error,
		{ id: string; name?: string; description?: string; capabilities?: string[] }
	>({
		mutationFn: ({ id, ...body }) =>
			fetchJson(`/api/dk/groups/${id}`, {
				method: "PUT",
				headers: authHeaders(true),
				body: JSON.stringify(body),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["groups"] });
			toast.success("Group updated");
		},
		onError: (err) => toast.error(err.message),
	});
}

export function useDeleteGroup() {
	const qc = useQueryClient();
	return useMutation<void, Error, string>({
		mutationFn: (id) =>
			fetchJson(`/api/dk/groups/${id}`, {
				method: "DELETE",
				headers: authHeaders(),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["groups"] });
			toast.success("Group deleted");
		},
		onError: (err) => toast.error(err.message),
	});
}

export function useAddMember() {
	const qc = useQueryClient();
	return useMutation<GroupMember, Error, { groupId: string; userId: string }>({
		mutationFn: ({ groupId, userId }) =>
			fetchJson(`/api/dk/groups/${groupId}/members`, {
				method: "POST",
				headers: authHeaders(true),
				body: JSON.stringify({ user_id: userId }),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["groups"] });
			void qc.invalidateQueries({ queryKey: ["users"] });
			toast.success("Member added");
		},
		onError: (err) => toast.error(err.message),
	});
}

export function useRemoveMember() {
	const qc = useQueryClient();
	return useMutation<void, Error, { groupId: string; userId: string }>({
		mutationFn: ({ groupId, userId }) =>
			fetchJson(`/api/dk/groups/${groupId}/members/${userId}`, {
				method: "DELETE",
				headers: authHeaders(),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["groups"] });
			void qc.invalidateQueries({ queryKey: ["users"] });
			toast.success("Member removed");
		},
		onError: (err) => toast.error(err.message),
	});
}

// ── API Keys Queries ─────────────────────────────────────

export function useAPIKeys() {
	return useQuery<APIKey[]>({
		queryKey: ["api-keys"],
		queryFn: () => fetchJson("/api/dk/api-keys", { headers: authHeaders() }),
	});
}

export function useCreateAPIKey() {
	const qc = useQueryClient();
	return useMutation<
		CreatedAPIKey,
		Error,
		{ name: string; capabilities: string[]; expires_at?: string }
	>({
		mutationFn: (input) =>
			fetchJson("/api/dk/api-keys", {
				method: "POST",
				headers: authHeaders(true),
				body: JSON.stringify(input),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["api-keys"] });
		},
		onError: (err) => toast.error(err.message),
	});
}

export function useDeleteAPIKey() {
	const qc = useQueryClient();
	return useMutation<void, Error, string>({
		mutationFn: (id) =>
			fetchJson(`/api/dk/api-keys/${id}`, {
				method: "DELETE",
				headers: authHeaders(),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["api-keys"] });
			toast.success("API key revoked");
		},
		onError: (err) => toast.error(err.message),
	});
}
