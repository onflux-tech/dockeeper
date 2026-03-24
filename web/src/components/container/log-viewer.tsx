import { useEffect, useMemo, useRef, useState } from "react";
import { Search, Pause, Play, Trash2, Circle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { useContainerLogs, type LogEntry } from "@/hooks/use-container-logs";

interface LogViewerProps {
	containerId: string;
}

function LogLine({ entry }: { entry: LogEntry }) {
	const ts = entry.timestamp
		? new Date(entry.timestamp).toLocaleTimeString("en-US", {
				hour12: false,
				hour: "2-digit",
				minute: "2-digit",
				second: "2-digit",
			})
		: "";

	return (
		<div className="flex gap-2 py-px font-mono text-xs leading-5 hover:bg-muted/30">
			<span className="shrink-0 text-muted-foreground select-none w-18 text-right">{ts}</span>
			<span
				className={`shrink-0 w-12 text-center select-none ${entry.stream === "stderr" ? "text-destructive" : "text-muted-foreground"}`}
			>
				{entry.stream === "stderr" ? "err" : "out"}
			</span>
			<span className="whitespace-pre-wrap break-all">{entry.message}</span>
		</div>
	);
}

export function LogViewer({ containerId }: LogViewerProps) {
	const { lines, connected, paused, pause, resume, clear } = useContainerLogs(containerId, {
		follow: true,
		tail: 100,
	});
	const [search, setSearch] = useState("");
	const [autoScroll, setAutoScroll] = useState(true);
	const scrollRef = useRef<HTMLDivElement>(null);

	const filtered = useMemo(() => {
		if (!search) return lines;
		const lower = search.toLowerCase();
		return lines.filter((l) => l.message.toLowerCase().includes(lower));
	}, [lines, search]);

	useEffect(() => {
		if (autoScroll && !paused && scrollRef.current) {
			scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
		}
	}, [filtered, autoScroll, paused]);

	const handleScroll = () => {
		if (!scrollRef.current) return;
		const { scrollTop, scrollHeight, clientHeight } = scrollRef.current;
		const atBottom = scrollHeight - scrollTop - clientHeight < 40;
		setAutoScroll(atBottom);
	};

	return (
		<Card className="flex flex-col">
			<CardHeader className="flex flex-row items-center justify-between gap-2 pb-2">
				<CardTitle className="text-sm flex items-center gap-2">
					Logs
					<Badge variant={connected ? "default" : "secondary"} className="text-xs">
						<Circle
							className={`h-2 w-2 mr-1 ${connected ? "fill-current" : "fill-muted-foreground"}`}
						/>
						{connected ? "Live" : "Disconnected"}
					</Badge>
				</CardTitle>
				<div className="flex items-center gap-2">
					<div className="relative">
						<Search className="absolute left-2 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
						<Input
							placeholder="Search logs..."
							value={search}
							onChange={(e) => setSearch(e.target.value)}
							className="h-8 w-48 pl-8 text-xs"
						/>
					</div>
					<Button
						variant="ghost"
						size="icon"
						className="h-8 w-8"
						onClick={paused ? resume : pause}
						title={paused ? "Resume" : "Pause"}
					>
						{paused ? <Play className="h-3.5 w-3.5" /> : <Pause className="h-3.5 w-3.5" />}
					</Button>
					<Button variant="ghost" size="icon" className="h-8 w-8" onClick={clear} title="Clear">
						<Trash2 className="h-3.5 w-3.5" />
					</Button>
				</div>
			</CardHeader>
			<CardContent className="flex-1 p-0">
				<div
					ref={scrollRef}
					onScroll={handleScroll}
					className="h-96 overflow-auto bg-background/50 px-3 py-2 border-t"
				>
					{filtered.length === 0 ? (
						<div className="flex items-center justify-center h-full text-sm text-muted-foreground">
							{lines.length === 0 ? "Waiting for logs..." : "No matching log lines"}
						</div>
					) : (
						filtered.map((entry, i) => <LogLine key={i} entry={entry} />)
					)}
				</div>
				{paused && (
					<div className="text-center py-1 text-xs text-muted-foreground bg-muted/50 border-t">
						Paused —{" "}
						{lines.length - filtered.length > 0 ? "new lines buffered" : "click play to resume"}
					</div>
				)}
			</CardContent>
		</Card>
	);
}
