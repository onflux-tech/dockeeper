import { useState } from "react";
import { Activity, ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { ServerFilterSelect } from "@/components/server-filter-select";
import { useEvents, type EventRecord } from "@/hooks/use-events";
import { relativeTime } from "@/lib/format";

const EVENT_TYPES = [
	{ value: "container.created", label: "Container Created" },
	{ value: "container.started", label: "Container Started" },
	{ value: "container.stopped", label: "Container Stopped" },
	{ value: "container.crashed", label: "Container Crashed" },
	{ value: "container.removed", label: "Container Removed" },
	{ value: "container.restarting", label: "Container Restarting" },
	{ value: "server.online", label: "Server Online" },
	{ value: "server.offline", label: "Server Offline" },
	{ value: "server.enrolled", label: "Server Enrolled" },
	{ value: "server.removed", label: "Server Removed" },
	{ value: "compose.deployed", label: "Compose Deployed" },
	{ value: "compose.stopped", label: "Compose Stopped" },
	{ value: "compose.failed", label: "Compose Failed" },
	{ value: "backup.completed", label: "Backup Completed" },
	{ value: "backup.failed", label: "Backup Failed" },
	{ value: "image.pulled", label: "Image Pulled" },
	{ value: "image.removed", label: "Image Removed" },
];

const SEVERITIES = [
	{ value: "info", label: "Info" },
	{ value: "warning", label: "Warning" },
	{ value: "critical", label: "Critical" },
];

const PAGE_SIZE = 50;

function severityVariant(s: string): "default" | "secondary" | "destructive" {
	if (s === "critical") return "destructive";
	if (s === "warning") return "secondary";
	return "default";
}

function eventTypeLabel(t: string): string {
	return t.replace(".", " ").replace(/\b\w/g, (c) => c.toUpperCase());
}

function EventDetailsCell({ event }: { event: EventRecord }) {
	if (!event.details || Object.keys(event.details).length === 0) {
		return <span className="text-muted-foreground">—</span>;
	}

	const entries = Object.entries(event.details).slice(0, 3);
	return (
		<div className="flex flex-wrap gap-1">
			{entries.map(([k, v]) => (
				<span key={k} className="inline-flex items-center gap-0.5 rounded bg-muted px-1.5 py-0.5 text-xs">
					<span className="text-muted-foreground">{k}:</span>
					<span className="max-w-[100px] truncate">{String(v)}</span>
				</span>
			))}
			{Object.keys(event.details).length > 3 && (
				<span className="text-xs text-muted-foreground">+{Object.keys(event.details).length - 3}</span>
			)}
		</div>
	);
}

export default function EventsPage() {
	const [eventType, setEventType] = useState("");
	const [severity, setSeverity] = useState("");
	const [serverId, setServerId] = useState("");
	const [offset, setOffset] = useState(0);

	const { data, isLoading } = useEvents({
		type: eventType || undefined,
		severity: severity || undefined,
		server_id: serverId || undefined,
		limit: PAGE_SIZE,
		offset,
	});

	const events = data ?? [];
	const hasNext = events.length === PAGE_SIZE;
	const page = Math.floor(offset / PAGE_SIZE) + 1;

	return (
		<div className="flex flex-col gap-6">
			{/* Header */}
			<div className="flex items-center gap-2">
				<Activity className="h-5 w-5 text-muted-foreground" />
				<h1 className="text-2xl font-semibold">Events</h1>
				{!isLoading && (
					<Badge variant="secondary" className="ml-1">
						{events.length}
					</Badge>
				)}
			</div>

			{/* Filters */}
			<div className="flex flex-wrap items-center gap-3">
				<Select
					value={eventType || "all"}
					onValueChange={(v) => {
						setEventType(v === "all" ? "" : v);
						setOffset(0);
					}}
				>
					<SelectTrigger className="w-52">
						<SelectValue placeholder="All event types" />
					</SelectTrigger>
					<SelectContent>
						<SelectItem value="all">All event types</SelectItem>
						{EVENT_TYPES.map((t) => (
							<SelectItem key={t.value} value={t.value}>
								{t.label}
							</SelectItem>
						))}
					</SelectContent>
				</Select>

				<Select
					value={severity || "all"}
					onValueChange={(v) => {
						setSeverity(v === "all" ? "" : v);
						setOffset(0);
					}}
				>
					<SelectTrigger className="w-36">
						<SelectValue placeholder="All severities" />
					</SelectTrigger>
					<SelectContent>
						<SelectItem value="all">All severities</SelectItem>
						{SEVERITIES.map((s) => (
							<SelectItem key={s.value} value={s.value}>
								{s.label}
							</SelectItem>
						))}
					</SelectContent>
				</Select>

				<ServerFilterSelect
					value={serverId || "all"}
					onValueChange={(v) => {
						setServerId(v === "all" ? "" : v);
						setOffset(0);
					}}
				/>
			</div>

			{/* Table */}
			<div className="rounded-md border">
				<Table>
					<TableHeader>
						<TableRow>
							<TableHead>Event</TableHead>
							<TableHead>Severity</TableHead>
							<TableHead>Server</TableHead>
							<TableHead>Container</TableHead>
							<TableHead>Details</TableHead>
							<TableHead>When</TableHead>
						</TableRow>
					</TableHeader>
					<TableBody>
						{isLoading ? (
							Array.from({ length: 8 }).map((_, i) => (
								<TableRow key={i}>
									{Array.from({ length: 6 }).map((_, j) => (
										<TableCell key={j}>
											<Skeleton className="h-4 w-full" />
										</TableCell>
									))}
								</TableRow>
							))
						) : events.length === 0 ? (
							<TableRow>
								<TableCell colSpan={6} className="h-32 text-center text-muted-foreground">
									No events found.
								</TableCell>
							</TableRow>
						) : (
							events.map((evt) => (
								<TableRow key={evt.id}>
									<TableCell>
										<span className="whitespace-nowrap text-sm font-medium">
											{eventTypeLabel(evt.event_type)}
										</span>
									</TableCell>
									<TableCell>
										<Badge variant={severityVariant(evt.severity)}>
											{evt.severity}
										</Badge>
									</TableCell>
									<TableCell className="whitespace-nowrap text-sm">
										{evt.server_name || evt.server_id || "—"}
									</TableCell>
									<TableCell className="max-w-[140px] truncate text-sm">
										{evt.container_name || "—"}
									</TableCell>
									<TableCell>
										<EventDetailsCell event={evt} />
									</TableCell>
									<TableCell className="whitespace-nowrap text-xs text-muted-foreground">
										{relativeTime(evt.timestamp)}
									</TableCell>
								</TableRow>
							))
						)}
					</TableBody>
				</Table>
			</div>

			{/* Pagination */}
			<div className="flex items-center justify-between">
				<p className="text-sm text-muted-foreground">Page {page}</p>
				<div className="flex items-center gap-2">
					<Button
						variant="outline"
						size="sm"
						disabled={offset === 0}
						onClick={() => setOffset((o) => Math.max(0, o - PAGE_SIZE))}
					>
						<ChevronLeft className="h-4 w-4" />
						Previous
					</Button>
					<Button
						variant="outline"
						size="sm"
						disabled={!hasNext}
						onClick={() => setOffset((o) => o + PAGE_SIZE)}
					>
						Next
						<ChevronRight className="h-4 w-4" />
					</Button>
				</div>
			</div>
		</div>
	);
}
