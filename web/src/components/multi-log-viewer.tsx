import { useState, useRef, useEffect, useCallback } from "react";
import { Columns2, List, Pause, Play, Search, X } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import { getAccessToken } from "@/lib/api";

interface MultiLogEntry {
	container_id: string;
	container_name: string;
	timestamp: string;
	stream: string;
	message: string;
}

interface MultiLogViewerProps {
	containerIds: string[];
	onClose?: () => void;
}

const COLORS = [
	"text-log-1",
	"text-log-2",
	"text-log-3",
	"text-log-4",
	"text-log-5",
	"text-log-6",
];

const BORDER_COLORS = [
	"border-l-log-1",
	"border-l-log-2",
	"border-l-log-3",
	"border-l-log-4",
	"border-l-log-5",
	"border-l-log-6",
];

export function MultiLogViewer({ containerIds, onClose }: MultiLogViewerProps) {
	const [mode, setMode] = useState<"interleaved" | "columns">("interleaved");
	const [following, setFollowing] = useState(true);
	const [search, setSearch] = useState("");
	const [logs, setLogs] = useState<MultiLogEntry[]>([]);
	const [connected, setConnected] = useState(false);
	const logsEndRef = useRef<HTMLDivElement>(null);
	const abortRef = useRef<AbortController | null>(null);
	const bufferRef = useRef<MultiLogEntry[]>([]);
	const flushTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);

	// Color map for container IDs
	const colorMap = useRef(new Map<string, number>());
	containerIds.forEach((id, i) => {
		if (!colorMap.current.has(id)) {
			colorMap.current.set(id, i % COLORS.length);
		}
	});

	const getColor = (id: string) => COLORS[colorMap.current.get(id) ?? 0];
	const getBorderColor = (id: string) => BORDER_COLORS[colorMap.current.get(id) ?? 0];

	// Start SSE stream
	useEffect(() => {
		if (containerIds.length < 2) return;

		const ctrl = new AbortController();
		abortRef.current = ctrl;

		const token = getAccessToken();
		const ids = containerIds.join(",");
		const url = `/api/dk/containers/multi-logs?ids=${ids}&tail=100&follow=true`;

		(async () => {
			try {
				const res = await fetch(url, {
					headers: token ? { Authorization: token } : {},
					signal: ctrl.signal,
				});

				if (!res.ok) return;

				setConnected(true);

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
							if (eventType === "log") {
								try {
									const entry = JSON.parse(line.slice(6)) as MultiLogEntry;
									bufferRef.current.push(entry);
								} catch {
									// ignore parse errors
								}
							} else if (eventType === "done") {
								setConnected(false);
							}
							eventType = "";
						}
					}
				}
			} catch {
				// AbortError or network error
			} finally {
				setConnected(false);
			}
		})();

		// Throttle DOM updates to max 10/sec
		flushTimerRef.current = setInterval(() => {
			if (bufferRef.current.length > 0) {
				const batch = bufferRef.current.splice(0);
				setLogs((prev) => {
					const next = [...prev, ...batch];
					// Cap at 5000 entries to prevent memory issues
					return next.length > 5000 ? next.slice(-5000) : next;
				});
			}
		}, 100);

		return () => {
			ctrl.abort();
			if (flushTimerRef.current) {
				clearInterval(flushTimerRef.current);
			}
			bufferRef.current = [];
		};
	}, [containerIds]);

	// Auto-scroll when following
	useEffect(() => {
		if (following && logsEndRef.current) {
			logsEndRef.current.scrollIntoView({ behavior: "smooth" });
		}
	}, [logs, following]);

	// Filter logs by search
	const filteredLogs = search
		? logs.filter(
				(l) =>
					l.message.toLowerCase().includes(search.toLowerCase()) ||
					l.container_name.toLowerCase().includes(search.toLowerCase()),
			)
		: logs;

	// Group by container for columns mode
	const groupedLogs = useCallback(() => {
		const groups = new Map<string, MultiLogEntry[]>();
		for (const id of containerIds) {
			groups.set(id, []);
		}
		for (const entry of filteredLogs) {
			const arr = groups.get(entry.container_id);
			if (arr) arr.push(entry);
		}
		return groups;
	}, [containerIds, filteredLogs]);

	const formatTime = (ts: string) => {
		if (!ts) return "";
		try {
			return new Date(ts).toLocaleTimeString();
		} catch {
			return ts;
		}
	};

	return (
		<div className="flex flex-col gap-3 rounded-lg border bg-card p-4">
			{/* Toolbar */}
			<div className="flex items-center gap-2">
				<h3 className="text-sm font-semibold">Multi-Container Logs</h3>
				{connected && (
					<Badge variant="outline" className="border-success text-success text-[10px]">
						Live
					</Badge>
				)}
				<div className="ml-auto flex items-center gap-2">
					<div className="relative">
						<Search className="absolute left-2 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
						<Input
							placeholder="Search..."
							value={search}
							onChange={(e) => setSearch(e.target.value)}
							className="h-8 w-48 pl-8 text-xs"
						/>
					</div>
					<Button
						variant="outline"
						size="icon"
						className="h-8 w-8"
						onClick={() => setFollowing(!following)}
						title={following ? "Pause auto-scroll" : "Resume auto-scroll"}
					>
						{following ? <Pause className="h-3.5 w-3.5" /> : <Play className="h-3.5 w-3.5" />}
					</Button>
					<Button
						variant="outline"
						size="icon"
						className="h-8 w-8"
						onClick={() => setMode(mode === "interleaved" ? "columns" : "interleaved")}
						title={mode === "interleaved" ? "Switch to columns" : "Switch to interleaved"}
					>
						{mode === "interleaved" ? (
							<Columns2 className="h-3.5 w-3.5" />
						) : (
							<List className="h-3.5 w-3.5" />
						)}
					</Button>
					{onClose && (
						<Button variant="ghost" size="icon" className="h-8 w-8" onClick={onClose}>
							<X className="h-3.5 w-3.5" />
						</Button>
					)}
				</div>
			</div>

			{/* Legend */}
			<div className="flex flex-wrap gap-2">
				{containerIds.map((id) => {
					const entry = logs.find((l) => l.container_id === id);
					const name = entry?.container_name || id.slice(0, 8);
					return (
						<span key={id} className={`text-xs font-mono ${getColor(id)}`}>
							{name}
						</span>
					);
				})}
			</div>

			{/* Logs */}
			{mode === "interleaved" ? (
				<ScrollArea className="h-96 rounded-md border bg-muted/30">
					<div className="p-2 font-mono text-xs leading-relaxed">
						{filteredLogs.map((entry, i) => (
							<div key={i} className={`flex gap-2 border-l-2 pl-2 ${getBorderColor(entry.container_id)}`}>
								<span className="shrink-0 text-muted-foreground">{formatTime(entry.timestamp)}</span>
								<span className={`shrink-0 ${getColor(entry.container_id)}`}>
									{entry.container_name || entry.container_id.slice(0, 8)}
								</span>
								<span className={entry.stream === "stderr" ? "text-red-400" : ""}>
									{entry.message}
								</span>
							</div>
						))}
						<div ref={logsEndRef} />
					</div>
				</ScrollArea>
			) : (
				<div className="grid gap-2" style={{ gridTemplateColumns: `repeat(${Math.min(containerIds.length, 3)}, 1fr)` }}>
					{Array.from(groupedLogs()).map(([cId, entries]) => {
						const name = entries[0]?.container_name || cId.slice(0, 8);
						return (
							<div key={cId} className="flex flex-col gap-1">
								<span className={`text-xs font-semibold ${getColor(cId)}`}>{name}</span>
								<ScrollArea className="h-80 rounded-md border bg-muted/30">
									<div className="p-2 font-mono text-[10px] leading-relaxed">
										{entries.map((entry, i) => (
											<div key={i} className={entry.stream === "stderr" ? "text-red-400" : ""}>
												<span className="text-muted-foreground">{formatTime(entry.timestamp)} </span>
												{entry.message}
											</div>
										))}
									</div>
								</ScrollArea>
							</div>
						);
					})}
				</div>
			)}
		</div>
	);
}
