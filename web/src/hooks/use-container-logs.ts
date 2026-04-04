import { useCallback, useEffect, useRef, useState } from "react";
import { getAccessToken } from "@/lib/api";

export interface LogEntry {
	timestamp: string;
	stream: "stdout" | "stderr";
	message: string;
}

const RECONNECT_BASE_MS = 2000;
const RECONNECT_MAX_MS = 30000;

export function useContainerLogs(
	containerId: string | undefined,
	options: { follow?: boolean; tail?: number; maxLines?: number } = {},
) {
	const { follow = true, tail = 100, maxLines = 5000 } = options;
	const [lines, setLines] = useState<LogEntry[]>([]);
	const [connected, setConnected] = useState(false);
	const [paused, setPaused] = useState(false);
	const [bufferedCount, setBufferedCount] = useState(0);
	const abortRef = useRef<AbortController | null>(null);
	const pausedRef = useRef(paused);
	pausedRef.current = paused;

	const bufferRef = useRef<LogEntry[]>([]);
	const retriesRef = useRef(0);
	const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
	const mountedRef = useRef(true);
	const maxLinesRef = useRef(maxLines);
	maxLinesRef.current = maxLines;
	// Track line count when paused to show buffered delta.
	const pausedAtRef = useRef(0);

	const connect = useCallback(() => {
		if (!containerId) return;

		abortRef.current?.abort();
		const controller = new AbortController();
		abortRef.current = controller;

		const token = getAccessToken();
		const url = `/api/dk/containers/${containerId}/logs/stream?follow=${follow}&tail=${tail}`;

		fetch(url, {
			headers: token ? { Authorization: token } : {},
			signal: controller.signal,
		})
			.then(async (res) => {
				if (!res.ok || !res.body) {
					if (mountedRef.current) setConnected(false);
					scheduleReconnect();
					return;
				}

				const reader = res.body.getReader();
				const decoder = new TextDecoder();
				let sseBuffer = "";

				for (;;) {
					const { done, value } = await reader.read();
					if (done) break;

					sseBuffer += decoder.decode(value, { stream: true });
					const parts = sseBuffer.split("\n\n");
					sseBuffer = parts.pop() ?? "";

					for (const part of parts) {
						if (!part.trim()) continue;

						// Skip SSE comments (keepalive).
						if (part.startsWith(":")) continue;

						const eventLine = part.split("\n").find((l) => l.startsWith("event: "));
						const eventType = eventLine?.slice(7).trim();

						if (eventType === "connected") {
							retriesRef.current = 0;
							if (mountedRef.current) setConnected(true);
							continue;
						}

						if (eventType === "done" || eventType === "error") {
							if (mountedRef.current) setConnected(false);
							if (follow) scheduleReconnect();
							return;
						}

						const dataLine = part.split("\n").find((l) => l.startsWith("data: "));
						if (!dataLine) continue;

						try {
							const entry = JSON.parse(dataLine.slice(6)) as LogEntry;
							bufferRef.current = [...bufferRef.current, entry].slice(-maxLinesRef.current);

							if (!pausedRef.current) {
								setLines([...bufferRef.current]);
							} else {
								setBufferedCount(bufferRef.current.length - pausedAtRef.current);
							}
						} catch {
							// skip malformed lines
						}
					}
				}

				if (mountedRef.current) setConnected(false);
				if (follow) scheduleReconnect();
			})
			.catch((err) => {
				if ((err as DOMException)?.name === "AbortError") return;
				if (mountedRef.current) setConnected(false);
				scheduleReconnect();
			});

		function scheduleReconnect() {
			if (!mountedRef.current || !follow) return;
			const delay = Math.min(RECONNECT_BASE_MS * 2 ** retriesRef.current, RECONNECT_MAX_MS);
			retriesRef.current++;
			reconnectTimerRef.current = setTimeout(() => {
				if (mountedRef.current) connect();
			}, delay);
		}
	}, [containerId, follow, tail]);

	useEffect(() => {
		mountedRef.current = true;
		// Reset state when tail changes (reconnects).
		bufferRef.current = [];
		setLines([]);
		setBufferedCount(0);
		connect();
		return () => {
			mountedRef.current = false;
			abortRef.current?.abort();
			if (reconnectTimerRef.current) {
				clearTimeout(reconnectTimerRef.current);
			}
		};
	}, [connect]);

	const resume = useCallback(() => {
		setPaused(false);
		setLines([...bufferRef.current]);
		setBufferedCount(0);
	}, []);

	const pause = useCallback(() => {
		setPaused(true);
		pausedAtRef.current = bufferRef.current.length;
		setBufferedCount(0);
	}, []);

	const clear = useCallback(() => {
		bufferRef.current = [];
		setLines([]);
		setBufferedCount(0);
	}, []);

	return { lines, connected, paused, bufferedCount, pause, resume, clear };
}
