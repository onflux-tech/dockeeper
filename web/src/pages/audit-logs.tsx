import { useState } from "react";
import { ScrollText, Search, ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
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
import { useAuditLogs } from "@/hooks/use-audit-logs";
import { ServerFilterSelect } from "@/components/server-filter-select";
import { relativeTime } from "@/lib/format";

const TARGET_TYPES = [
	{ value: "", label: "All types" },
	{ value: "container", label: "Container" },
	{ value: "image", label: "Image" },
	{ value: "network", label: "Network" },
	{ value: "volume", label: "Volume" },
	{ value: "compose_stack", label: "Compose" },
	{ value: "backup_policy", label: "Backup" },
	{ value: "alert_rule", label: "Alert Rule" },
	{ value: "notification_channel", label: "Channel" },
	{ value: "setting", label: "Setting" },
	{ value: "terminal", label: "Terminal" },
];

function actionColor(action: string): "default" | "secondary" | "destructive" {
	if (action.includes("delete") || action.includes("remove")) return "destructive";
	if (action.includes("create") || action.includes("trigger")) return "default";
	return "secondary";
}

export default function AuditLogsPage() {
	const [page, setPage] = useState(1);
	const [search, setSearch] = useState("");
	const [targetType, setTargetType] = useState("");
	const [serverId, setServerId] = useState("");
	const [debouncedSearch, setDebouncedSearch] = useState("");

	const { data, isLoading } = useAuditLogs({
		page,
		perPage: 25,
		target_type: targetType || undefined,
		search: debouncedSearch || undefined,
		server_id: serverId || undefined,
	});

	const handleSearchKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
		if (e.key === "Enter") {
			setDebouncedSearch(search);
			setPage(1);
		}
	};

	return (
		<div className="flex flex-col gap-4">
			{/* Header */}
			<div className="flex items-center gap-2">
				<ScrollText className="h-5 w-5 text-muted-foreground" />
				<h1 className="text-2xl font-semibold">Audit Logs</h1>
			</div>

			{/* Filters */}
			<div className="flex flex-wrap items-center gap-3">
				<div className="relative w-64">
					<Search className="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
					<Input
						placeholder="Search actions, targets…"
						className="pl-8"
						value={search}
						onChange={(e) => setSearch(e.target.value)}
						onKeyDown={handleSearchKeyDown}
					/>
				</div>
				<Select
					value={targetType}
					onValueChange={(v) => {
						setTargetType(v === "all" ? "" : v);
						setPage(1);
					}}
				>
					<SelectTrigger className="w-44">
						<SelectValue placeholder="All types" />
					</SelectTrigger>
					<SelectContent>
						<SelectItem value="all">All types</SelectItem>
						{TARGET_TYPES.filter((t) => t.value).map((t) => (
							<SelectItem key={t.value} value={t.value}>
								{t.label}
							</SelectItem>
						))}
					</SelectContent>
				</Select>
				<ServerFilterSelect
					value={serverId || "all"}
					onValueChange={(v) => {
						setServerId(v === "all" ? "" : v);
						setPage(1);
					}}
				/>
			</div>

			{/* Table */}
			<div className="rounded-md border">
				<Table>
					<TableHeader>
						<TableRow>
							<TableHead>Action</TableHead>
							<TableHead>Target Type</TableHead>
							<TableHead>Target ID</TableHead>
							<TableHead>User</TableHead>
							<TableHead>IP Address</TableHead>
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
						) : data?.items.length === 0 ? (
							<TableRow>
								<TableCell colSpan={6} className="h-32 text-center text-muted-foreground">
									No audit logs found.
								</TableCell>
							</TableRow>
						) : (
							data?.items.map((log) => (
								<TableRow key={log.id}>
									<TableCell>
										<Badge variant={actionColor(log.action)}>{log.action}</Badge>
									</TableCell>
									<TableCell className="text-muted-foreground">{log.target_type || "—"}</TableCell>
									<TableCell className="max-w-[120px] truncate font-mono text-xs">
										{log.target_id || "—"}
									</TableCell>
									<TableCell className="max-w-[120px] truncate font-mono text-xs">
										{log.user || "system"}
									</TableCell>
									<TableCell className="text-xs text-muted-foreground">
										{log.ip_address || "—"}
									</TableCell>
									<TableCell className="whitespace-nowrap text-xs text-muted-foreground">
										{relativeTime(log.timestamp)}
									</TableCell>
								</TableRow>
							))
						)}
					</TableBody>
				</Table>
			</div>

			{/* Pagination */}
			{data && data.totalPages > 1 && (
				<div className="flex items-center justify-between">
					<p className="text-sm text-muted-foreground">
						Page {data.page} of {data.totalPages} ({data.totalItems} total)
					</p>
					<div className="flex items-center gap-2">
						<Button
							variant="outline"
							size="sm"
							disabled={page <= 1}
							onClick={() => setPage((p) => Math.max(1, p - 1))}
						>
							<ChevronLeft className="h-4 w-4" />
							Previous
						</Button>
						<Button
							variant="outline"
							size="sm"
							disabled={page >= data.totalPages}
							onClick={() => setPage((p) => p + 1)}
						>
							Next
							<ChevronRight className="h-4 w-4" />
						</Button>
					</div>
				</div>
			)}
		</div>
	);
}
