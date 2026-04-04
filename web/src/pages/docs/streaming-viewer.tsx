import { useCallback, useEffect, useRef, useState } from "react";
import { ArrowDown, Circle, Loader2, Square, Trash2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import type { ApiEndpoint } from "@/types/docs";
import type { HistoryEntry } from "./playground-sheet";
import { buildRequestUrl } from "./playground-utils";

// ── Constants ──────────────────────────────────────────────────────────────

const MAX_EVENTS = 500;

// ── Types ──────────────────────────────────────────────────────────────────

interface SSEEvent {
	id: number;
	timestamp: string;
	type: string;
	data: string;
}

type StreamStatus = "idle" | "connecting" | "streaming" | "disconnected" | "error";

// ── SSE line parser ────────────────────────────────────────────────────────

function parseSSEChunk(chunk: string): { event: string; data: string }[] {
	const events: { event: string; data: string }[] = [];
	let currentEvent = "";
	let currentData: string[] = [];

	for (const line of chunk.split("\n")) {
		if (line.startsWith("event:")) {
			currentEvent = line.slice(6).trim();
		} else if (line.startsWith("data:")) {
			currentData.push(line.slice(5).trim());
		} else if (line === "" && currentData.length > 0) {
			events.push({ event: currentEvent || "message", data: currentData.join("\n") });
			currentEvent = "";
			currentData = [];
		}
	}

	// Handle trailing data without empty line
	if (currentData.length > 0) {
		events.push({ event: currentEvent || "message", data: currentData.join("\n") });
	}

	return events;
}

// ── Event type badge color ─────────────────────────────────────────────────

function eventTypeColor(type: string): string {
	switch (type) {
		case "progress":
			return "border-blue-400/30 text-blue-400 bg-blue-400/10";
		case "error":
			return "border-red-400/30 text-red-400 bg-red-400/10";
		case "complete":
		case "done":
			return "border-green-400/30 text-green-400 bg-green-400/10";
		default:
			return "border-muted-foreground/30 text-muted-foreground bg-muted/50";
	}
}

// ── Props ──────────────────────────────────────────────────────────────────

interface StreamingViewerProps {
	endpoint: ApiEndpoint;
	apiKey: string;
	pathParams: Record<string, string>;
	queryParams: Record<string, string>;
	onHistory?: (entry: HistoryEntry) => void;
}

// ── Component ──────────────────────────────────────────────────────────────

export function StreamingViewer({
	endpoint,
	apiKey,
	pathParams,
	queryParams,
	onHistory,
}: StreamingViewerProps) {
	const [events, setEvents] = useState<SSEEvent[]>([]);
	const [status, setStatus] = useState<StreamStatus>("idle");
	const [errorMsg, setErrorMsg] = useState<string | null>(null);
	const [elapsed, setElapsed] = useState(0);
	const [autoScroll, setAutoScroll] = useState(true);
	const [droppedCount, setDroppedCount] = useState(0);

	const abortRef = useRef<AbortController | null>(null);
	const containerRef = useRef<HTMLDivElement>(null);
	const eventIdRef = useRef(0);
	const startTimeRef = useRef(0);
	const timerRef = useRef<ReturnType<typeof setInterval>>(undefined);

	// Auto-scroll
	useEffect(() => {
		if (autoScroll && containerRef.current) {
			containerRef.current.scrollTop = containerRef.current.scrollHeight;
		}
	}, [events, autoScroll]);

	// Detect manual scroll
	const handleScroll = useCallback(() => {
		const el = containerRef.current;
		if (!el) return;
		const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 40;
		setAutoScroll(atBottom);
	}, []);

	const jumpToBottom = useCallback(() => {
		if (containerRef.current) {
			containerRef.current.scrollTop = containerRef.current.scrollHeight;
			setAutoScroll(true);
		}
	}, []);

	// Timer
	useEffect(() => {
		if (status === "streaming") {
			startTimeRef.current = Date.now();
			timerRef.current = setInterval(() => {
				setElapsed(Math.floor((Date.now() - startTimeRef.current) / 1000));
			}, 1000);
		} else {
			clearInterval(timerRef.current);
		}
		return () => clearInterval(timerRef.current);
	}, [status]);

	// Connect
	const connect = useCallback(async () => {
		abortRef.current?.abort();
		const controller = new AbortController();
		abortRef.current = controller;

		setEvents([]);
		setDroppedCount(0);
		setErrorMsg(null);
		setStatus("connecting");
		setElapsed(0);
		eventIdRef.current = 0;

		const origin = window.location.origin;
		const url = buildRequestUrl(origin, endpoint.path, pathParams, queryParams);
		const headers: Record<string, string> = { Accept: "text/event-stream" };
		if (apiKey) headers["X-API-Key"] = apiKey;

		try {
			const res = await fetch(url, {
				method: endpoint.method,
				headers,
				signal: controller.signal,
			});

			if (!res.ok) {
				setStatus("error");
				setErrorMsg(`${res.status} ${res.statusText}`);

				onHistory?.({
					id: crypto.randomUUID(),
					method: endpoint.method,
					path: endpoint.path,
					title: endpoint.title,
					status: res.status,
					duration: 0,
					timestamp: Date.now(),
				});
				return;
			}

			setStatus("streaming");

			const reader = res.body?.getReader();
			if (!reader) {
				setStatus("error");
				setErrorMsg("No response body");
				return;
			}

			const decoder = new TextDecoder();
			let buffer = "";

			while (true) {
				const { done, value } = await reader.read();
				if (done) break;

				buffer += decoder.decode(value, { stream: true });
				const parsed = parseSSEChunk(buffer);

				// Keep tail of buffer if it doesn't end with double newline
				const lastNewlines = buffer.lastIndexOf("\n\n");
				if (lastNewlines >= 0) {
					buffer = buffer.slice(lastNewlines + 2);
				}

				if (parsed.length > 0) {
					setEvents((prev) => {
						const now = new Date().toLocaleTimeString();
						const newEvents: SSEEvent[] = parsed.map((e) => ({
							id: eventIdRef.current++,
							timestamp: now,
							type: e.event,
							data: e.data,
						}));
						const combined = [...prev, ...newEvents];
						if (combined.length > MAX_EVENTS) {
							const dropped = combined.length - MAX_EVENTS;
							setDroppedCount((c) => c + dropped);
							return combined.slice(-MAX_EVENTS);
						}
						return combined;
					});
				}
			}

			setStatus("disconnected");
		} catch (err: unknown) {
			if (err instanceof Error && err.name === "AbortError") {
				setStatus("disconnected");
			} else {
				setStatus("error");
				setErrorMsg(err instanceof Error ? err.message : "Connection failed");
			}
		}
	}, [endpoint, apiKey, pathParams, queryParams, onHistory]);

	const stop = useCallback(() => {
		abortRef.current?.abort();
		setStatus("disconnected");
	}, []);

	const clear = useCallback(() => {
		setEvents([]);
		setDroppedCount(0);
		eventIdRef.current = 0;
	}, []);

	// Cleanup on unmount
	useEffect(() => {
		return () => abortRef.current?.abort();
	}, []);

	const formatElapsed = (s: number) => {
		const m = Math.floor(s / 60);
		const sec = s % 60;
		return `${String(m).padStart(2, "0")}:${String(sec).padStart(2, "0")}`;
	};

	return (
		<div className="space-y-3">
			{/* Status bar */}
			<div className="flex items-center gap-3 flex-wrap">
				{status === "idle" && (
					<Button size="sm" onClick={connect}>
						Connect
					</Button>
				)}
				{status === "connecting" && (
					<Badge variant="outline" className="border-blue-400/30 text-blue-400">
						<Loader2 className="mr-1 h-3 w-3 animate-spin" />
						Connecting
					</Badge>
				)}
				{status === "streaming" && (
					<>
						<Badge variant="outline" className="border-green-400/30 text-green-400">
							<Circle className="mr-1 h-2 w-2 animate-pulse fill-green-400" />
							STREAMING
						</Badge>
						<span className="font-mono text-xs text-muted-foreground">{events.length} events</span>
						<span className="font-mono text-xs text-muted-foreground">
							{formatElapsed(elapsed)}
						</span>
						<Button size="sm" variant="destructive" onClick={stop} className="ml-auto">
							<Square className="mr-1 h-3 w-3" /> Stop
						</Button>
					</>
				)}
				{status === "disconnected" && (
					<>
						<Badge variant="outline" className="border-muted-foreground/30 text-muted-foreground">
							Disconnected
						</Badge>
						<span className="font-mono text-xs text-muted-foreground">{events.length} events</span>
						<Button size="sm" onClick={connect} className="ml-auto">
							Reconnect
						</Button>
					</>
				)}
				{status === "error" && (
					<>
						<Badge variant="outline" className="border-red-400/30 text-red-400">
							Error
						</Badge>
						<span className="text-xs text-red-400">{errorMsg}</span>
						<Button size="sm" onClick={connect} className="ml-auto">
							Retry
						</Button>
					</>
				)}
				{events.length > 0 && (
					<Button variant="ghost" size="sm" onClick={clear}>
						<Trash2 className="mr-1 h-3 w-3" /> Clear
					</Button>
				)}
			</div>

			{/* Dropped indicator */}
			{droppedCount > 0 && (
				<p className="text-xs text-amber-400">
					{droppedCount} older events dropped (buffer limit: {MAX_EVENTS})
				</p>
			)}

			{/* Events */}
			<div
				ref={containerRef}
				onScroll={handleScroll}
				className="relative max-h-[60vh] overflow-y-auto rounded-lg border border-border bg-muted/30"
			>
				{events.length === 0 && status !== "connecting" && (
					<div className="flex items-center justify-center py-8 text-xs text-muted-foreground">
						{status === "idle" ? "Click Connect to start streaming" : "No events received"}
					</div>
				)}
				{events.map((event) => (
					<div
						key={event.id}
						className="flex gap-2 border-b border-border/50 px-3 py-1.5 text-xs last:border-0"
					>
						<span className="shrink-0 font-mono text-muted-foreground">{event.timestamp}</span>
						<Badge
							variant="outline"
							className={cn("shrink-0 text-[10px] font-mono", eventTypeColor(event.type))}
						>
							{event.type}
						</Badge>
						<pre className="min-w-0 flex-1 overflow-x-auto whitespace-pre-wrap break-all font-mono text-muted-foreground">
							{event.data}
						</pre>
					</div>
				))}
			</div>

			{/* Jump to bottom */}
			{!autoScroll && events.length > 10 && (
				<Button variant="outline" size="sm" onClick={jumpToBottom} className="w-full text-xs">
					<ArrowDown className="mr-1 h-3 w-3" /> Jump to bottom
				</Button>
			)}
		</div>
	);
}
