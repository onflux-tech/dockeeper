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
export function getSettingValue(
	settings: SettingRecord[] | undefined,
	key: string,
): string {
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
			const existingMap = new Map(
				existing.map((r) => [r.key, r.id]),
			);

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
		onError: (err) =>
			toast.error(`Failed to regenerate token: ${err.message}`),
	});
}
