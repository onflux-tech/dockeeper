import { useEffect, useMemo, useRef, useState } from "react";
import {
	Search,
	Download,
	Trash2,
	Circle,
	ArrowDown,
	ArrowUp,
	Clock,
	Regex,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import {
	Tooltip,
	TooltipContent,
	TooltipProvider,
	TooltipTrigger,
} from "@/components/ui/tooltip";
import { useContainerLogs, type LogEntry } from "@/hooks/use-container-logs";

// ── Types ────────────────────────────────────────────────────────────────────

interface LogViewerProps {
	containerId: string;
	containerName?: string;
}

type TimeMode = "off" | "time" | "full";
type LogLevel = "debug" | "info" | "warn" | "error" | "fatal" | "unknown";

const TAIL_OPTIONS = [50, 100, 500, 1000, 5000] as const;
const LS_TIME_MODE_KEY = "dk-log-time-mode";

// ── Log level detection ──────────────────────────────────────────────────────

function detectLogLevel(message: string): LogLevel {
	const first200 = message.substring(0, 200);

	// JSON structured logs: {"level":"info", ...}
	const jsonMatch = first200.match(/"level"\s*:\s*"(\w+)"/i);
	if (jsonMatch) {
		const lvl = jsonMatch[1].toLowerCase();
		if (lvl === "debug" || lvl === "trace") return "debug";
		if (lvl === "info") return "info";
		if (lvl === "warn" || lvl === "warning") return "warn";
		if (lvl === "error" || lvl === "err") return "error";
		if (lvl === "fatal" || lvl === "panic" || lvl === "critical") return "fatal";
	}

	// Common text patterns: [INFO], INFO:, level=info, etc.
	// Use word boundaries + common delimiters to avoid false positives.
	const lower = first200.toLowerCase();
	if (/(?:^|\[| )(?:debug|trace)(?:\]|:|,| )/i.test(lower)) return "debug";
	if (/(?:^|\[| )(?:info)(?:\]|:|,| )/i.test(lower)) return "info";
	if (/(?:^|\[| )(?:warn|warning)(?:\]|:|,| )/i.test(lower)) return "warn";
	if (/(?:^|\[| )(?:error|err)(?:\]|:|,| )/i.test(lower)) return "error";
	if (/(?:^|\[| )(?:fatal|panic|critical)(?:\]|:|,| )/i.test(lower)) return "fatal";

	return "unknown";
}

const levelStyles: Record<LogLevel, { label: string; className: string }> = {
	debug: { label: "DBG", className: "text-muted-foreground/60" },
	info: { label: "INF", className: "text-blue-500" },
	warn: { label: "WRN", className: "text-yellow-500" },
	error: { label: "ERR", className: "text-destructive" },
	fatal: { label: "FTL", className: "text-destructive font-bold" },
	unknown: { label: "", className: "" },
};

// ── Timestamp formatting ─────────────────────────────────────────────────────

function formatTimestamp(ts: string, mode: TimeMode): string {
	if (mode === "off" || !ts) return "";
	const d = new Date(ts);
	if (mode === "time") {
		return d.toLocaleTimeString("en-US", {
			hour12: false,
			hour: "2-digit",
			minute: "2-digit",
			second: "2-digit",
		});
	}
	// full
	return d.toLocaleString("en-US", {
		month: "2-digit",
		day: "2-digit",
		year: "numeric",
		hour: "2-digit",
		minute: "2-digit",
		second: "2-digit",
		hour12: true,
	});
}

// ── Search helpers ───────────────────────────────────────────────────────────

function parseSearchPattern(search: string): {
	isRegex: boolean;
	test: (text: string) => boolean;
	error: string | null;
} {
	if (!search) return { isRegex: false, test: () => true, error: null };

	// If starts with /, treat as regex: /pattern/flags
	if (search.startsWith("/")) {
		const lastSlash = search.lastIndexOf("/");
		if (lastSlash > 0) {
			const pattern = search.slice(1, lastSlash);
			const flags = search.slice(lastSlash + 1);
			try {
				const re = new RegExp(pattern, flags);
				return { isRegex: true, test: (text) => re.test(text), error: null };
			} catch (e) {
				return {
					isRegex: true,
					test: () => false,
					error: (e as Error).message,
				};
			}
		}
		// Incomplete regex, still treat as regex mode.
		return { isRegex: true, test: () => true, error: null };
	}

	// Plain text search.
	const lower = search.toLowerCase();
	return { isRegex: false, test: (text) => text.toLowerCase().includes(lower), error: null };
}

// ── LogLine Component ────────────────────────────────────────────────────────

function LogLine({
	entry,
	timeMode,
	index,
}: {
	entry: LogEntry;
	timeMode: TimeMode;
	index: number;
}) {
	const level = detectLogLevel(entry.message);
	const style = levelStyles[level];
	const ts = formatTimestamp(entry.timestamp, timeMode);

	return (
		<div
			className={`flex gap-2 py-0.5 px-2 font-mono text-xs leading-5 ${index % 2 === 0 ? "bg-transparent" : "bg-muted/20"} hover:bg-muted/40`}
		>
			{timeMode !== "off" && (
				<span className="shrink-0 text-muted-foreground/70 select-none tabular-nums">
					{ts}
				</span>
			)}
			{level !== "unknown" ? (
				<span className={`shrink-0 w-8 text-center select-none font-semibold ${style.className}`}>
					{style.label}
				</span>
			) : entry.stream === "stderr" ? (
				<span className="shrink-0 w-8 text-center select-none text-muted-foreground/50 text-[10px]">
					stderr
				</span>
			) : (
				<span className="shrink-0 w-8" />
			)}
			<span className="whitespace-pre-wrap break-words min-w-0 flex-1">
				{entry.message}
			</span>
		</div>
	);
}

// ── LogViewer Component ──────────────────────────────────────────────────────

export function LogViewer({ containerId, containerName }: LogViewerProps) {
	const [tail, setTail] = useState<number>(100);
	const [timeMode, setTimeMode] = useState<TimeMode>(() => {
		return (localStorage.getItem(LS_TIME_MODE_KEY) as TimeMode) || "time";
	});
	const [newestFirst, setNewestFirst] = useState(true);
	const [autoRefresh, setAutoRefresh] = useState(true);
	const [search, setSearch] = useState("");

	const { lines, connected, paused, bufferedCount, pause, resume, clear } =
		useContainerLogs(containerId, {
			follow: true,
			tail,
			maxLines: Math.max(tail, 5000),
		});

	const scrollRef = useRef<HTMLDivElement>(null);
	const [autoScroll, setAutoScroll] = useState(true);

	// Sync auto-refresh ↔ pause.
	useEffect(() => {
		if (autoRefresh && paused) resume();
		if (!autoRefresh && !paused) pause();
	}, [autoRefresh, paused, pause, resume]);

	// Persist time mode.
	useEffect(() => {
		localStorage.setItem(LS_TIME_MODE_KEY, timeMode);
	}, [timeMode]);

	// Search/filter.
	const searchPattern = useMemo(() => parseSearchPattern(search), [search]);

	const filtered = useMemo(() => {
		const result = search ? lines.filter((l) => searchPattern.test(l.message)) : lines;
		return newestFirst ? [...result].reverse() : result;
	}, [lines, search, searchPattern, newestFirst]);

	// Auto-scroll.
	useEffect(() => {
		if (autoScroll && autoRefresh && scrollRef.current) {
			if (newestFirst) {
				scrollRef.current.scrollTop = 0;
			} else {
				scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
			}
		}
	}, [filtered, autoScroll, autoRefresh, newestFirst]);

	const handleScroll = () => {
		if (!scrollRef.current) return;
		const { scrollTop, scrollHeight, clientHeight } = scrollRef.current;
		if (newestFirst) {
			setAutoScroll(scrollTop < 40);
		} else {
			setAutoScroll(scrollHeight - scrollTop - clientHeight < 40);
		}
	};

	// Cycle time mode: off → time → full → off.
	const cycleTimeMode = () => {
		setTimeMode((m) => {
			if (m === "off") return "time";
			if (m === "time") return "full";
			return "off";
		});
	};

	// Download logs.
	const downloadLogs = () => {
		const content = lines
			.map((l) => {
				const ts = timeMode !== "off" ? `${formatTimestamp(l.timestamp, timeMode)} ` : "";
				const level = detectLogLevel(l.message);
				const lvl = level !== "unknown" ? `[${level.toUpperCase()}] ` : "";
				return `${ts}${lvl}${l.message}`;
			})
			.join("\n");

		const blob = new Blob([content], { type: "text/plain" });
		const url = URL.createObjectURL(blob);
		const a = document.createElement("a");
		a.href = url;
		a.download = `${containerName ?? containerId}-logs-${Date.now()}.log`;
		a.click();
		URL.revokeObjectURL(url);
	};

	return (
		<div className="flex flex-col h-[calc(100vh-16rem)] min-h-[300px] border rounded-md overflow-hidden bg-background">
			{/* Toolbar */}
			<div className="flex items-center gap-2 px-3 py-2 border-b bg-muted/30 flex-wrap">
				{/* Connection status */}
				<Badge variant={connected ? "default" : "secondary"} className="text-xs shrink-0">
					<Circle
						className={`h-2 w-2 mr-1 ${connected ? "fill-current" : "fill-muted-foreground"}`}
					/>
					{connected ? "Live" : "Disconnected"}
				</Badge>

				{/* Tail selector */}
				<Select value={String(tail)} onValueChange={(v) => setTail(Number(v))}>
					<SelectTrigger size="sm" className="w-24 text-xs">
						<SelectValue />
					</SelectTrigger>
					<SelectContent>
						{TAIL_OPTIONS.map((n) => (
							<SelectItem key={n} value={String(n)} className="text-xs">
								{n} lines
							</SelectItem>
						))}
					</SelectContent>
				</Select>

				{/* Time toggle */}
				<TooltipProvider delayDuration={300}>
					<Tooltip>
						<TooltipTrigger asChild>
							<Button
								variant={timeMode === "off" ? "ghost" : "secondary"}
								size="sm"
								className="h-7 px-2 text-xs"
								onClick={cycleTimeMode}
							>
								<Clock className="h-3 w-3 mr-1" />
								{timeMode === "off" ? "Time" : timeMode === "time" ? "HH:MM" : "Full"}
							</Button>
						</TooltipTrigger>
						<TooltipContent>
							<p>Click to cycle: Off → Time → Full</p>
						</TooltipContent>
					</Tooltip>
				</TooltipProvider>

				{/* Order toggle */}
				<Button
					variant="ghost"
					size="sm"
					className="h-7 px-2 text-xs"
					onClick={() => setNewestFirst((p) => !p)}
				>
					{newestFirst ? (
						<>
							<ArrowDown className="h-3 w-3 mr-1" /> Newest
						</>
					) : (
						<>
							<ArrowUp className="h-3 w-3 mr-1" /> Oldest
						</>
					)}
				</Button>

				{/* Auto-refresh checkbox */}
				<label className="flex items-center gap-1.5 text-xs cursor-pointer shrink-0">
					<Checkbox
						checked={autoRefresh}
						onCheckedChange={(v) => setAutoRefresh(v === true)}
						className="h-3.5 w-3.5"
					/>
					Auto-refresh
				</label>

				<div className="flex-1" />

				{/* Search */}
				<div className="relative">
					{searchPattern.isRegex ? (
						<Regex
							className={`absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 ${searchPattern.error ? "text-destructive" : "text-primary"}`}
						/>
					) : (
						<Search className="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
					)}
					<Input
						placeholder="Search or /regex/i..."
						value={search}
						onChange={(e) => setSearch(e.target.value)}
						className={`h-7 w-52 pl-8 text-xs ${searchPattern.error ? "border-destructive" : ""}`}
					/>
				</div>

				{/* Download */}
				<TooltipProvider delayDuration={300}>
					<Tooltip>
						<TooltipTrigger asChild>
							<Button
								variant="ghost"
								size="icon"
								className="h-7 w-7"
								onClick={downloadLogs}
								disabled={lines.length === 0}
							>
								<Download className="h-3.5 w-3.5" />
							</Button>
						</TooltipTrigger>
						<TooltipContent>Download logs</TooltipContent>
					</Tooltip>
				</TooltipProvider>

				{/* Clear */}
				<TooltipProvider delayDuration={300}>
					<Tooltip>
						<TooltipTrigger asChild>
							<Button variant="ghost" size="icon" className="h-7 w-7" onClick={clear}>
								<Trash2 className="h-3.5 w-3.5" />
							</Button>
						</TooltipTrigger>
						<TooltipContent>Clear logs</TooltipContent>
					</Tooltip>
				</TooltipProvider>
			</div>

			{/* Log lines */}
			<div
				ref={scrollRef}
				onScroll={handleScroll}
				className="flex-1 min-h-0 overflow-auto font-mono text-xs"
			>
				{filtered.length === 0 ? (
					<div className="flex items-center justify-center h-full text-sm text-muted-foreground">
						{lines.length === 0 ? "Waiting for logs..." : "No matching log lines"}
					</div>
				) : (
					filtered.map((entry, i) => (
						<LogLine key={i} entry={entry} timeMode={timeMode} index={i} />
					))
				)}
			</div>

			{/* Paused bar */}
			{!autoRefresh && (
				<div className="flex items-center justify-center gap-2 py-1 text-xs text-muted-foreground bg-muted/50 border-t">
					<span>Paused</span>
					{bufferedCount > 0 && (
						<Badge variant="secondary" className="text-[10px] h-4">
							+{bufferedCount} new
						</Badge>
					)}
					<Button
						variant="link"
						size="sm"
						className="h-4 text-xs px-1"
						onClick={() => setAutoRefresh(true)}
					>
						Resume
					</Button>
				</div>
			)}

			{/* Search error */}
			{searchPattern.error && (
				<div className="px-3 py-1 text-xs text-destructive bg-destructive/10 border-t">
					Invalid regex: {searchPattern.error}
				</div>
			)}

			{/* Line count */}
			<div className="flex items-center justify-between px-3 py-1 text-[10px] text-muted-foreground border-t bg-muted/20">
				<span>{filtered.length} lines displayed</span>
				<span>{lines.length} total in buffer</span>
			</div>
		</div>
	);
}
