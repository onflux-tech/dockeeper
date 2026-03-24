import { useCallback, useEffect, useRef, useState } from "react";
import { getAccessToken } from "@/lib/api";

export interface LogEntry {
	timestamp: string;
	stream: "stdout" | "stderr";
	message: string;
}

const MAX_LINES = 5000;
const RECONNECT_BASE_MS = 2000;
const RECONNECT_MAX_MS = 30000;

export function useContainerLogs(
	containerId: string | undefined,
	options: { follow?: boolean; tail?: number } = {},
) {
	const { follow = true, tail = 100 } = options;
	const [lines, setLines] = useState<LogEntry[]>([]);
	const [connected, setConnected] = useState(false);
	const [paused, setPaused] = useState(false);
	const abortRef = useRef<AbortController | null>(null);
	const pausedRef = useRef(paused);
	pausedRef.current = paused;

	const bufferRef = useRef<LogEntry[]>([]);
	const retriesRef = useRef(0);
	const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
	const mountedRef = useRef(true);

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
				let buffer = "";

				for (;;) {
					const { done, value } = await reader.read();
					if (done) break;

					buffer += decoder.decode(value, { stream: true });
					const parts = buffer.split("\n\n");
					buffer = parts.pop() ?? "";

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
							bufferRef.current = [...bufferRef.current, entry].slice(-MAX_LINES);

							if (!pausedRef.current) {
								setLines([...bufferRef.current]);
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
	}, []);

	const pause = useCallback(() => {
		setPaused(true);
	}, []);

	const clear = useCallback(() => {
		bufferRef.current = [];
		setLines([]);
	}, []);

	return { lines, connected, paused, pause, resume, clear };
}
