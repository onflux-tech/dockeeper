import { useCallback, useEffect, useRef, useState } from "react";
import { getAccessToken } from "@/lib/api";

export interface LogEntry {
	timestamp: string;
	stream: "stdout" | "stderr";
	message: string;
}

const MAX_LINES = 5000;

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

	const connect = useCallback(() => {
		if (!containerId) return;

		abortRef.current?.abort();
		const controller = new AbortController();
		abortRef.current = controller;

		const token = getAccessToken();
		const url = `/api/dk/containers/${containerId}/logs/stream?follow=${follow}&tail=${tail}`;

		setConnected(true);

		fetch(url, {
			headers: token ? { Authorization: token } : {},
			signal: controller.signal,
		})
			.then(async (res) => {
				if (!res.ok || !res.body) {
					setConnected(false);
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

						const dataLine = part
							.split("\n")
							.find((l) => l.startsWith("data: "));
						if (!dataLine) continue;

						const eventLine = part
							.split("\n")
							.find((l) => l.startsWith("event: "));
						const eventType = eventLine?.slice(7).trim();

						if (eventType === "done") {
							setConnected(false);
							return;
						}

						try {
							const entry = JSON.parse(dataLine.slice(6)) as LogEntry;
							bufferRef.current = [...bufferRef.current, entry].slice(
								-MAX_LINES,
							);

							if (!pausedRef.current) {
								setLines([...bufferRef.current]);
							}
						} catch {
							// skip malformed lines
						}
					}
				}

				setConnected(false);
			})
			.catch(() => {
				setConnected(false);
			});
	}, [containerId, follow, tail]);

	useEffect(() => {
		connect();
		return () => {
			abortRef.current?.abort();
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
