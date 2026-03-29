import { useState, useMemo } from "react";
import { useNavigate, useSearchParams } from "react-router";
import { Server, Plus, Search, Trash2, MoreHorizontal, ExternalLink, Pencil, Copy, Check, Terminal } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { SortableHead, type SortDir } from "@/components/ui/sortable-head";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuSeparator,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { PaginationControls } from "@/components/ui/pagination-controls";
import { PerPageSelector } from "@/components/ui/per-page-selector";
import { CascadeDeleteDialog } from "@/components/cascade-delete-dialog";
import { EditServerDialog } from "@/components/server/edit-server-dialog";
import { TagFilterSelect } from "@/components/tag-filter-select";
import { useCollection } from "@/hooks/use-collection";
import { useListFilters } from "@/hooks/use-list-filters";
import { useDeleteServer, useQuickInstallCommand } from "@/hooks/use-servers";
import { useRealtime } from "@/hooks/use-realtime";
import { formatUptime } from "@/lib/format";
import { containsFilter, andFilters } from "@/lib/pocketbase-filters";
import type { ServerRecord, ContainerRecord } from "@/types/pocketbase";

type ServerSortKey = "name" | "status" | "last_seen";

const SERVER_SORT_MAP: Record<ServerSortKey, string> = {
	name: "name",
	status: "status",
	last_seen: "last_seen",
};

function statusVariant(status: string): "default" | "secondary" | "destructive" | "outline" {
	switch (status) {
		case "online":
			return "default";
		case "offline":
			return "destructive";
		case "pending":
			return "outline";
		default:
			return "secondary";
	}
}

function metricColor(value: number): string {
	if (value >= 80) return "text-destructive";
	if (value >= 50) return "text-yellow-500";
	return "text-foreground";
}

export default function ServersPage() {
	const { filters, setSearch, setPage, setPerPage, setSort, pbOptions } = useListFilters({
		defaultSort: "name",
		searchFields: ["name", "hostname", "ip_address"],
	});

	// Tag filter (needs contains ~ instead of eq =).
	const [searchParams, setSearchParams] = useSearchParams();
	const tagValue = searchParams.get("tag") ?? "all";

	function setTagFilter(value: string) {
		setSearchParams(
			(prev) => {
				const next = new URLSearchParams(prev);
				if (value === "all") {
					next.delete("tag");
				} else {
					next.set("tag", value);
				}
				next.delete("page");
				return next;
			},
			{ replace: true },
		);
	}

	const tagFilter = tagValue !== "all" ? containsFilter("tags", tagValue) : "";
	const mergedFilter = andFilters([pbOptions.filter, tagFilter]);
	const finalOptions = { ...pbOptions, filter: mergedFilter };

	const { data: serversResult, isLoading } = useCollection("servers", finalOptions);
	const deleteServer = useDeleteServer();
	const navigate = useNavigate();
	useRealtime("servers", 3000);

	// Fetch all containers for running/total count per server.
	const { data: containersResult } = useCollection("containers", { perPage: 500 });
	useRealtime("containers", 3000);
	const containerCounts = useMemo(() => {
		const map = new Map<string, { running: number; total: number }>();
		for (const c of containersResult?.items ?? []) {
			const ct = (c as ContainerRecord);
			const entry = map.get(ct.server) ?? { running: 0, total: 0 };
			entry.total++;
			if (ct.status === "running") entry.running++;
			map.set(ct.server, entry);
		}
		return map;
	}, [containersResult]);

	const [deleteTarget, setDeleteTarget] = useState<ServerRecord | null>(null);
	const [editTarget, setEditTarget] = useState<ServerRecord | null>(null);
	const [copied, setCopied] = useState(false);
	const [popoverOpen, setPopoverOpen] = useState(false);
	const servers = serversResult?.items ?? [];
	const totalServers = serversResult?.totalItems ?? 0;

	const { data: quickInstall } = useQuickInstallCommand();

	function copyCommand() {
		if (!quickInstall?.command) return;
		navigator.clipboard.writeText(quickInstall.command).then(() => {
			setCopied(true);
			setTimeout(() => {
				setCopied(false);
				setPopoverOpen(false);
			}, 600);
		});
	}

	// Derive sort state from URL
	const rawSort = filters.sort;
	const isDesc = rawSort.startsWith("-");
	const sortField = isDesc ? rawSort.slice(1) : rawSort;
	const sortKey = (Object.entries(SERVER_SORT_MAP).find(([, v]) => v === sortField)?.[0] ?? "name") as ServerSortKey;
	const sortDir: SortDir = isDesc ? "desc" : "asc";

	const handleSortChange = (key: ServerSortKey) => {
		const pbField = SERVER_SORT_MAP[key];
		if (sortKey === key) {
			setSort(sortDir === "asc" ? `-${pbField}` : pbField);
		} else {
			setSort(pbField);
		}
	};

	return (
		<div className="flex flex-col gap-6">
			<div className="flex items-center justify-between">
				<div className="flex items-center gap-2">
					<Server className="h-5 w-5 text-muted-foreground" />
					<h1 className="text-2xl font-semibold">Servers</h1>
					{serversResult && (
						<Badge variant="secondary" className="ml-1">
							{totalServers}
						</Badge>
					)}
				</div>
				<div className="flex items-center gap-2">
					{quickInstall?.command && (
						<Popover open={popoverOpen} onOpenChange={setPopoverOpen}>
							<PopoverTrigger asChild>
								<Button variant="outline" size="sm">
									<Terminal className="mr-2 h-4 w-4" />
									Quick Install
								</Button>
							</PopoverTrigger>
							<PopoverContent align="end" className="w-[640px]">
								<div className="flex flex-col gap-3">
									<div>
										<p className="text-sm font-medium">Quick Agent Install</p>
										<p className="text-xs text-muted-foreground">
											Run this command on any server to install the agent
										</p>
									</div>
									<div className="flex items-center gap-2">
										<code className="min-w-0 flex-1 truncate rounded bg-muted border px-3 py-2 text-xs font-mono select-all">
											{quickInstall.command}
										</code>
										<Button
											variant="outline"
											size="icon"
											className="shrink-0 h-8 w-8"
											onClick={copyCommand}
										>
											{copied ? (
												<Check className="h-3.5 w-3.5 text-success" />
											) : (
												<Copy className="h-3.5 w-3.5" />
											)}
										</Button>
									</div>
									<p className="text-[11px] text-muted-foreground leading-relaxed">
										Requires port <strong>50051</strong> open on the manager. Agents
										only make outbound connections — no inbound ports needed on
										agent servers.
									</p>
								</div>
							</PopoverContent>
						</Popover>
					)}
					<Button size="sm" onClick={() => navigate("/servers/create")}>
						<Plus className="mr-2 h-4 w-4" />
						Add Server
					</Button>
				</div>
			</div>

			<div className="flex items-center gap-3">
				<div className="relative w-64">
					<Search className="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
					<Input
						placeholder="Search servers…"
						className="pl-8"
						value={filters.search}
						onChange={(e) => {
							setSearch(e.target.value);
						}}
					/>
				</div>
				<TagFilterSelect
					resourceType="server"
					value={tagValue}
					onValueChange={setTagFilter}
				/>
			</div>

			<div className="rounded-md border">
				<Table>
					<TableHeader>
						<TableRow>
							<SortableHead label="Status" sortKey="status" currentKey={sortKey} currentDir={sortDir} onToggle={handleSortChange} />
							<SortableHead label="Name" sortKey="name" currentKey={sortKey} currentDir={sortDir} onToggle={handleSortChange} />
							<TableHead>Hostname</TableHead>
							<TableHead>Containers</TableHead>
							<TableHead>Uptime</TableHead>
							<TableHead>CPU%</TableHead>
							<TableHead>RAM%</TableHead>
							<TableHead>OS / Version</TableHead>
							<TableHead className="w-10" />
						</TableRow>
					</TableHeader>
					<TableBody>
						{isLoading ? (
							Array.from({ length: 5 }).map((_, i) => (
								<TableRow key={i}>
									{Array.from({ length: 9 }).map((_, j) => (
										<TableCell key={j}>
											<Skeleton className="h-4 w-full" />
										</TableCell>
									))}
								</TableRow>
							))
						) : servers.length === 0 ? (
							<TableRow>
								<TableCell colSpan={9} className="h-32 text-center text-muted-foreground">
									{filters.search
										? "No servers match your search."
										: 'No servers yet. Click "Add Server" to get started.'}
								</TableCell>
							</TableRow>
						) : (
							servers.map((server) => {
								const counts = containerCounts.get(server.id);
								return (
									<TableRow
										key={server.id}
										className="cursor-pointer"
										onClick={() => navigate(`/servers/${server.id}`)}
									>
										<TableCell>
											<Badge variant={statusVariant(server.status)}>{server.status}</Badge>
										</TableCell>
										<TableCell className="font-medium">
											{server.name}
											{server.description && (
												<p className="text-xs text-muted-foreground truncate max-w-[200px]">
													{server.description}
												</p>
											)}
										</TableCell>
										<TableCell className="font-mono text-xs">{server.hostname || "—"}</TableCell>
										<TableCell className="text-xs tabular-nums">
											{counts ? (
												<span>
													<span className="font-medium">{counts.running}</span>
													<span className="text-muted-foreground"> / {counts.total}</span>
												</span>
											) : (
												"—"
											)}
										</TableCell>
										<TableCell className="text-xs text-muted-foreground whitespace-nowrap">
											{server.uptime_seconds ? formatUptime(server.uptime_seconds) : "—"}
										</TableCell>
										<TableCell className={`text-xs tabular-nums font-medium ${metricColor(server.current_cpu ?? 0)}`}>
											{server.current_cpu != null && server.current_cpu > 0
												? `${server.current_cpu.toFixed(1)}%`
												: "—"}
										</TableCell>
										<TableCell className={`text-xs tabular-nums font-medium ${metricColor(server.current_ram ?? 0)}`}>
											{server.current_ram != null && server.current_ram > 0
												? `${server.current_ram.toFixed(1)}%`
												: "—"}
										</TableCell>
										<TableCell className="text-xs text-muted-foreground">
											{server.os && server.arch ? `${server.os}/${server.arch}` : "—"}
											{server.agent_version && (
												<span className="ml-1 text-muted-foreground/70">v{server.agent_version}</span>
											)}
										</TableCell>
										<TableCell>
											<DropdownMenu>
												<DropdownMenuTrigger asChild>
													<Button
														variant="ghost"
														size="icon"
														className="h-8 w-8"
														onClick={(e) => e.stopPropagation()}
													>
														<MoreHorizontal className="h-4 w-4" />
													</Button>
												</DropdownMenuTrigger>
												<DropdownMenuContent align="end">
													<DropdownMenuItem
														onClick={(e) => {
															e.stopPropagation();
															navigate(`/servers/${server.id}`);
														}}
													>
														<ExternalLink className="mr-2 h-4 w-4" />
														View Details
													</DropdownMenuItem>
													<DropdownMenuItem
														onClick={(e) => {
															e.stopPropagation();
															setEditTarget(server);
														}}
													>
														<Pencil className="mr-2 h-4 w-4" />
														Edit
													</DropdownMenuItem>
													<DropdownMenuSeparator />
													<DropdownMenuItem
														className="text-destructive focus:text-destructive"
														onClick={(e) => {
															e.stopPropagation();
															setDeleteTarget(server);
														}}
													>
														<Trash2 className="mr-2 h-4 w-4" />
														Delete Server
													</DropdownMenuItem>
												</DropdownMenuContent>
											</DropdownMenu>
										</TableCell>
									</TableRow>
								);
							})
						)}
					</TableBody>
				</Table>
			</div>

			<div className="flex items-center justify-between">
				<PerPageSelector value={filters.perPage} onChange={setPerPage} />
				<PaginationControls
					page={filters.page}
					totalPages={serversResult?.totalPages ?? 0}
					totalItems={totalServers}
					onPageChange={setPage}
					label="servers"
				/>
			</div>

			{deleteTarget && (
				<CascadeDeleteDialog
					open={deleteTarget !== null}
					onOpenChange={(open) => {
						if (!open) setDeleteTarget(null);
					}}
					entityType="server"
					entityId={deleteTarget.id}
					entityName={deleteTarget.name ?? deleteTarget.hostname}
					onConfirm={() => {
						deleteServer.mutate(deleteTarget.id);
						setDeleteTarget(null);
					}}
				/>
			)}

			{editTarget && (
				<EditServerDialog
					key={editTarget.id}
					open={editTarget !== null}
					onOpenChange={(open) => {
						if (!open) setEditTarget(null);
					}}
					server={editTarget}
				/>
			)}
		</div>
	);
}
