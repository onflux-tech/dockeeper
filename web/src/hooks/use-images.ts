import { useCallback, useRef, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { getAccessToken } from "@/lib/api";

// ── Pull with SSE progress ──────────────────────────

export interface PullProgress {
	id: string;
	status: string;
	current: number;
	total: number;
	done: boolean;
	error: string;
}

export interface PullState {
	pulling: boolean;
	layers: Map<string, PullProgress>;
	error: string | null;
	overallPercent: number;
}

export function useImagePull() {
	const queryClient = useQueryClient();
	const [state, setState] = useState<PullState>({
		pulling: false,
		layers: new Map(),
		error: null,
		overallPercent: 0,
	});
	const abortRef = useRef<AbortController | null>(null);

	const pull = useCallback(
		(serverID: string, imageRef: string) => {
			if (state.pulling) return;

			const abort = new AbortController();
			abortRef.current = abort;

			setState({ pulling: true, layers: new Map(), error: null, overallPercent: 0 });

			const token = getAccessToken();
			const params = new URLSearchParams({ server_id: serverID, image: imageRef });
			const url = `/api/dk/images/pull?${params.toString()}`;

			const eventSource = new EventSource(url);

			// EventSource doesn't support headers. Use fetch+ReadableStream for auth.
			// Actually, SSE via EventSource won't inject auth header. Let's use fetch.
			eventSource.close();

			fetch(url, {
				headers: token ? { Authorization: token } : {},
				signal: abort.signal,
			})
				.then(async (res) => {
					if (!res.ok) {
						const body = await res.json().catch(() => ({ error: res.statusText }));
						throw new Error(body.error || "Pull failed");
					}

					const reader = res.body?.getReader();
					if (!reader) throw new Error("No response body");

					const decoder = new TextDecoder();
					let buffer = "";

					const layers = new Map<string, PullProgress>();

					while (true) {
						const { done, value } = await reader.read();
						if (done) break;

						buffer += decoder.decode(value, { stream: true });
						const lines = buffer.split("\n");
						buffer = lines.pop() ?? "";

						let eventType = "";
						for (const line of lines) {
							if (line.startsWith("event: ")) {
								eventType = line.slice(7);
							} else if (line.startsWith("data: ")) {
								const data = line.slice(6);
								if (eventType === "progress") {
									try {
										const progress = JSON.parse(data) as PullProgress;
										if (progress.id) {
											layers.set(progress.id, progress);
										}

										let totalBytes = 0;
										let currentBytes = 0;
										for (const l of layers.values()) {
											if (l.total > 0) {
												totalBytes += l.total;
												currentBytes += l.current;
											}
										}

										const pct = totalBytes > 0 ? Math.round((currentBytes / totalBytes) * 100) : 0;
										setState({
											pulling: true,
											layers: new Map(layers),
											error: progress.error || null,
											overallPercent: pct,
										});

										if (progress.done) {
											if (progress.error) {
												setState((prev) => ({
													...prev,
													pulling: false,
													error: progress.error,
												}));
												toast.error(`Pull failed: ${progress.error}`);
											} else {
												setState((prev) => ({
													...prev,
													pulling: false,
													overallPercent: 100,
												}));
												toast.success(`Image ${imageRef} pulled successfully`);
												void queryClient.invalidateQueries({
													queryKey: ["collection", "images"],
												});
											}
											return;
										}
									} catch {
										// ignore parse errors
									}
								} else if (eventType === "done") {
									setState((prev) => ({
										...prev,
										pulling: false,
										overallPercent: 100,
									}));
									toast.success(`Image ${imageRef} pulled successfully`);
									void queryClient.invalidateQueries({
										queryKey: ["collection", "images"],
									});
									return;
								} else if (eventType === "error") {
									try {
										const err = JSON.parse(data) as { error?: string };
										setState((prev) => ({
											...prev,
											pulling: false,
											error: err.error ?? "Unknown error",
										}));
										toast.error(err.error ?? "Pull failed");
									} catch {
										setState((prev) => ({
											...prev,
											pulling: false,
											error: "Pull failed",
										}));
									}
									return;
								}
								eventType = "";
							}
						}
					}

					// Stream ended without explicit done event
					setState((prev) => {
						if (prev.pulling) {
							toast.success(`Image ${imageRef} pulled successfully`);
							void queryClient.invalidateQueries({
								queryKey: ["collection", "images"],
							});
							return { ...prev, pulling: false, overallPercent: 100 };
						}
						return prev;
					});
				})
				.catch((err: Error) => {
					if (err.name === "AbortError") return;
					setState((prev) => ({ ...prev, pulling: false, error: err.message }));
					toast.error(`Pull failed: ${err.message}`);
				});
		},
		[state.pulling, queryClient],
	);

	const cancel = useCallback(() => {
		abortRef.current?.abort();
		setState((prev) => ({ ...prev, pulling: false }));
	}, []);

	return { ...state, pull, cancel };
}

// ── Remove Image ────────────────────────────────────

export function useImageRemove() {
	const queryClient = useQueryClient();

	return useMutation<unknown, Error, { id: string; force?: boolean }>({
		mutationFn: async ({ id, force }) => {
			const token = getAccessToken();
			const params = force ? "?force=true" : "";
			const res = await fetch(`/api/dk/images/${id}${params}`, {
				method: "DELETE",
				headers: token ? { Authorization: token } : {},
			});
			if (!res.ok) {
				const body = await res.json().catch(() => ({ error: res.statusText }));
				throw new Error(body.error || "Remove failed");
			}
			return res.json();
		},
		onSuccess: () => {
			void queryClient.invalidateQueries({ queryKey: ["collection", "images"] });
			toast.success("Image removed");
		},
		onError: (err) => {
			toast.error(`Remove failed: ${err.message}`);
		},
	});
}

// ── Prune Images ────────────────────────────────────

export function useImagePrune() {
	const queryClient = useQueryClient();

	return useMutation<
		{ success: boolean; space_reclaimed: number },
		Error,
		{ serverID: string; all: boolean }
	>({
		mutationFn: async ({ serverID, all }) => {
			const token = getAccessToken();
			const res = await fetch("/api/dk/images/prune", {
				method: "POST",
				headers: {
					"Content-Type": "application/json",
					...(token ? { Authorization: token } : {}),
				},
				body: JSON.stringify({ server_id: serverID, all }),
			});
			if (!res.ok) {
				const body = await res.json().catch(() => ({ error: res.statusText }));
				throw new Error(body.error || "Prune failed");
			}
			return res.json();
		},
		onSuccess: (data) => {
			void queryClient.invalidateQueries({ queryKey: ["collection", "images"] });
			const mb = (data.space_reclaimed / (1024 * 1024)).toFixed(1);
			toast.success(`Images pruned — ${mb} MB reclaimed`);
		},
		onError: (err) => {
			toast.error(`Prune failed: ${err.message}`);
		},
	});
}
