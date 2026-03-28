import { useState, useEffect } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { getAccessToken } from "@/lib/api";
import { toast } from "sonner";
import type { BatchItemRecord } from "@/types/pocketbase";

// ── Types ────────────────────────────────────────────

export interface BatchProgress {
	job_id: string;
	container_id: string;
	status: string;
	error?: string;
	succeeded: number;
	failed: number;
	cancelled: number;
	total: number;
	job_status: string;
}

export interface BatchJob {
	id: string;
	action: string;
	status: string;
	total: number;
	succeeded: number;
	failed: number;
	cancelled: number;
	created_by: string;
	started_at: string;
	completed_at: string;
	items?: BatchItemRecord[];
	created: string;
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

export function useBatchSubmit() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: (data: {
			action: string;
			container_ids?: string[];
			tag_filter?: string;
		}) =>
			fetchJson<{ job_id: string; status: string }>("/api/dk/batch", {
				method: "POST",
				headers: authHeaders(true),
				body: JSON.stringify(data),
			}),
		onSuccess: () => {
			void queryClient.invalidateQueries({ queryKey: ["batch-jobs"] });
		},
		onError: (err: Error) => toast.error(err.message),
	});
}

export function useBatchCancel() {
	return useMutation({
		mutationFn: (jobId: string) =>
			fetchJson(`/api/dk/batch/${jobId}/cancel`, {
				method: "POST",
				headers: authHeaders(),
			}),
		onSuccess: () => toast.success("Batch cancellation requested"),
		onError: (err: Error) => toast.error(err.message),
	});
}

export function useBatchJob(jobId: string | null) {
	return useQuery<BatchJob>({
		queryKey: ["batch-job", jobId],
		queryFn: () =>
			fetchJson<BatchJob>(`/api/dk/batch/${jobId}`, {
				headers: authHeaders(),
			}),
		enabled: !!jobId,
	});
}

export function useBatchJobs() {
	return useQuery<{ items: BatchJob[]; page: number }>({
		queryKey: ["batch-jobs"],
		queryFn: () =>
			fetchJson<{ items: BatchJob[]; page: number }>("/api/dk/batch?per_page=50", {
				headers: authHeaders(),
			}),
	});
}

export function useBatchProgress(jobId: string | null) {
	const [state, setState] = useState<{
		progress: BatchProgress[];
		isComplete: boolean;
		trackedJobId: string | null;
	}>({ progress: [], isComplete: false, trackedJobId: null });

	const queryClient = useQueryClient();

	// Reset state when jobId changes (outside the effect).
	if (state.trackedJobId !== jobId) {
		setState({ progress: [], isComplete: false, trackedJobId: jobId });
	}

	useEffect(() => {
		if (!jobId) return;

		const ctrl = new AbortController();
		const token = getAccessToken();
		const url = `/api/dk/batch/${jobId}/progress`;

		(async () => {
			try {
				const res = await fetch(url, {
					headers: token ? { Authorization: token } : {},
					signal: ctrl.signal,
				});

				if (!res.ok) return;

				const reader = res.body?.getReader();
				if (!reader) return;

				const decoder = new TextDecoder();
				let buffer = "";
				let eventType = "";

				while (true) {
					const { done, value } = await reader.read();
					if (done) break;

					buffer += decoder.decode(value, { stream: true });
					const lines = buffer.split("\n");
					buffer = lines.pop() ?? "";

					for (const line of lines) {
						if (line.startsWith("event: ")) {
							eventType = line.slice(7);
						} else if (line.startsWith("data: ")) {
							const raw = line.slice(6);
							try {
								const data: BatchProgress = JSON.parse(raw);
								if (eventType === "progress") {
									setState((prev) => ({
										...prev,
										progress: [...prev.progress, data],
									}));
								} else if (eventType === "complete") {
									setState((prev) => ({
										...prev,
										progress: [...prev.progress, data],
										isComplete: true,
									}));
									void queryClient.invalidateQueries({ queryKey: ["collection", "containers"] });
									void queryClient.invalidateQueries({ queryKey: ["batch-jobs"] });
								}
							} catch {
								// ignore parse errors
							}
							eventType = "";
						}
					}
				}
			} catch {
				// AbortError or network error
			} finally {
				setState((prev) => ({ ...prev, isComplete: true }));
			}
		})();

		return () => {
			ctrl.abort();
		};
	}, [jobId, queryClient]);

	const latestProgress =
		state.progress.length > 0 ? state.progress[state.progress.length - 1] : null;

	return { progress: state.progress, latestProgress, isComplete: state.isComplete };
}

// Helper to get the latest item statuses from progress stream.
export function buildItemStatusMap(
	progress: BatchProgress[],
): Map<string, { status: string; error?: string }> {
	const map = new Map<string, { status: string; error?: string }>();
	for (const p of progress) {
		if (p.container_id) {
			map.set(p.container_id, { status: p.status, error: p.error });
		}
	}
	return map;
}
