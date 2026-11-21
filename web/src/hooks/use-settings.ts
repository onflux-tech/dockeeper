import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { getAccessToken } from "@/lib/api";
import pb from "@/lib/pb";
import type { SettingRecord } from "@/types/pocketbase";

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

// ── Settings Queries ─────────────────────────────────────

export function useSettings() {
	return useQuery<SettingRecord[]>({
		queryKey: ["settings"],
		queryFn: async () => {
			const result = await pb.collection("settings").getFullList();
			return result as SettingRecord[];
		},
	});
}

/** Get a setting value by key from the settings list. */
export function getSettingValue(settings: SettingRecord[] | undefined, key: string): string {
	const rec = settings?.find((s) => s.key === key);
	if (!rec) return "";
	if (typeof rec.value === "string") return rec.value;
	return String(rec.value ?? "");
}

// ── Settings Mutations ───────────────────────────────────

export function useSaveSettings() {
	const qc = useQueryClient();
	return useMutation<void, Error, Record<string, string>>({
		mutationFn: async (keyValues) => {
			const existing = await pb.collection("settings").getFullList();
			const existingMap = new Map(existing.map((r) => [r.key, r.id]));

			for (const [key, value] of Object.entries(keyValues)) {
				const id = existingMap.get(key);
				if (id) {
					await pb.collection("settings").update(id, { value });
				} else {
					await pb.collection("settings").create({ key, value });
				}
			}
		},
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["settings"] });
			toast.success("Settings saved");
		},
		onError: (err) => toast.error(`Save failed: ${err.message}`),
	});
}

// ── Password Change ──────────────────────────────────────

interface ChangePasswordInput {
	current_password: string;
	new_password: string;
}

export function useChangePassword() {
	return useMutation<{ status: string }, Error, ChangePasswordInput>({
		mutationFn: (input) =>
			fetchJson("/api/dk/auth/password", {
				method: "POST",
				headers: authHeaders(true),
				body: JSON.stringify(input),
			}),
		onSuccess: () => toast.success("Password changed successfully"),
		onError: (err) => toast.error(`Password change failed: ${err.message}`),
	});
}

// ── Health / About ───────────────────────────────────────

interface HealthInfo {
	status: string;
	version: string;
	commit: string;
}

export function useHealth() {
	return useQuery<HealthInfo>({
		queryKey: ["health"],
		queryFn: () => fetchJson("/api/dk/health"),
		staleTime: 60_000,
	});
}

// ── Global Enrollment Token ─────────────────────────────

interface TokenResult {
	token: string;
}

export function useEnrollmentToken() {
	return useQuery<TokenResult>({
		queryKey: ["enrollment-token"],
		queryFn: () =>
			fetchJson("/api/dk/enrollment-token", {
				headers: authHeaders(),
			}),
	});
}

export function useRegenerateEnrollmentToken() {
	const qc = useQueryClient();
	return useMutation<TokenResult, Error>({
		mutationFn: () =>
			fetchJson("/api/dk/enrollment-token/regenerate", {
				method: "POST",
				headers: authHeaders(),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["enrollment-token"] });
			toast.success("Enrollment token regenerated");
		},
		onError: (err) => toast.error(`Failed to regenerate token: ${err.message}`),
	});
}

// ── Blackout Windows ──────────────────────────────────────

export interface BlackoutWindow {
	id: string;
	server: string;
	name: string;
	start_time: string;
	end_time: string;
	recurring: boolean;
	weekdays: number[];
	time_start: string;
	time_end: string;
	enabled: boolean;
}

export function useBlackoutWindows() {
	return useQuery<BlackoutWindow[]>({
		queryKey: ["blackout-windows"],
		queryFn: () => fetchJson("/api/dk/blackout-windows", { headers: authHeaders() }),
	});
}

export function useCreateBlackoutWindow() {
	const qc = useQueryClient();
	return useMutation<{ id: string }, Error, Record<string, unknown>>({
		mutationFn: (input) =>
			fetchJson("/api/dk/blackout-windows", {
				method: "POST",
				headers: authHeaders(true),
				body: JSON.stringify(input),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["blackout-windows"] });
			toast.success("Blackout window created");
		},
		onError: (err) => toast.error(`Create failed: ${err.message}`),
	});
}

export function useUpdateBlackoutWindow() {
	const qc = useQueryClient();
	return useMutation<{ status: string }, Error, { id: string; fields: Record<string, unknown> }>({
		mutationFn: ({ id, fields }) =>
			fetchJson(`/api/dk/blackout-windows/${id}`, {
				method: "PUT",
				headers: authHeaders(true),
				body: JSON.stringify(fields),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["blackout-windows"] });
			toast.success("Blackout window updated");
		},
		onError: (err) => toast.error(`Update failed: ${err.message}`),
	});
}

export function useDeleteBlackoutWindow() {
	const qc = useQueryClient();
	return useMutation<{ status: string }, Error, string>({
		mutationFn: (id) =>
			fetchJson(`/api/dk/blackout-windows/${id}`, {
				method: "DELETE",
				headers: authHeaders(),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["blackout-windows"] });
			toast.success("Blackout window deleted");
		},
		onError: (err) => toast.error(`Delete failed: ${err.message}`),
	});
}

// ── Retention Settings ────────────────────────────────────

export type RetentionSettings = Record<string, string>;

export function useRetentionSettings() {
	return useQuery<RetentionSettings>({
		queryKey: ["retention-settings"],
		queryFn: () => fetchJson("/api/dk/settings/retention", { headers: authHeaders() }),
	});
}

export function useUpdateRetention() {
	const qc = useQueryClient();
	return useMutation<{ status: string }, Error, RetentionSettings>({
		mutationFn: (input) =>
			fetchJson("/api/dk/settings/retention", {
				method: "PUT",
				headers: authHeaders(true),
				body: JSON.stringify(input),
			}),
		onSuccess: () => {
			void qc.invalidateQueries({ queryKey: ["retention-settings"] });
			toast.success("Retention settings saved");
		},
		onError: (err) => toast.error(`Save failed: ${err.message}`),
	});
}

// ── Self-Update System ───────────────────────────────────

export interface UpdateCheckResult {
	available: boolean;
	current: string;
	latest?: string;
	changelog?: string;
	published_at?: string;
	assets?: { name: string; size: number; url: string }[];
}

export function useUpdateCheck() {
	return useMutation<UpdateCheckResult, Error>({
		mutationFn: () =>
			fetchJson("/api/dk/system/update/check", {
				headers: authHeaders(),
			}),
	});
}

export function useLatestRelease() {
	return useQuery<UpdateCheckResult, Error>({
		queryKey: ["latest-release"],
		queryFn: () =>
			fetchJson("/api/dk/system/update/check", {
				headers: authHeaders(),
			}),
		staleTime: 5 * 60_000,
		retry: false,
	});
}

export function useSystemVersion() {
	return useQuery<{ version: string }>({
		queryKey: ["system-version"],
		queryFn: () => fetchJson("/api/dk/system/version"),
		staleTime: 60_000,
	});
}

// ── Agent Update (SSE) ──────────────────────────────────

export interface AgentUpdateProgress {
	step: string;
	message: string;
	version?: string;
}

export function useAgentUpdate() {
	return useMutation<void, Error, string>({
		mutationFn: (serverId: string) =>
			new Promise((resolve, reject) => {
				const token = getAccessToken();
				const url = `/api/dk/system/update/agent/${serverId}`;

				fetch(url, {
					method: "POST",
					headers: token ? { Authorization: token } : {},
				})
					.then((res) => {
						if (!res.ok) {
							res
								.json()
								.then((b) => reject(new Error(b.error || b.message || res.statusText)))
								.catch(() => reject(new Error(res.statusText)));
							return;
						}

						const contentType = res.headers.get("content-type") || "";

						// Non-SSE response (already up to date).
						if (!contentType.includes("text/event-stream")) {
							toast.info("Agent already up to date");
							resolve();
							return;
						}

						const reader = res.body?.getReader();
						if (!reader) {
							resolve();
							return;
						}

						const decoder = new TextDecoder();
						let buffer = "";

						function read() {
							reader!.read().then(({ done, value }) => {
								if (done) {
									resolve();
									return;
								}
								buffer += decoder.decode(value, { stream: true });
								const lines = buffer.split("\n");
								buffer = lines.pop() || "";

								for (const line of lines) {
									if (line.startsWith("event: error")) {
										// Next data line has the error.
										const idx = lines.indexOf(line);
										const dataLine = lines[idx + 1];
										if (dataLine?.startsWith("data: ")) {
											try {
												const parsed = JSON.parse(dataLine.slice(6));
												reject(new Error(parsed.message || "Agent update failed"));
											} catch {
												reject(new Error("Agent update failed"));
											}
										}
										return;
									}
									if (line.startsWith("event: complete")) {
										toast.success("Agent updated successfully");
									}
								}
								read();
							});
						}
						read();
					})
					.catch(reject);
			}),
		onError: (err) => toast.error(`Agent update failed: ${err.message}`),
	});
}
