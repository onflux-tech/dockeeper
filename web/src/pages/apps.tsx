import { useMemo } from "react";
import { Link } from "react-router";
import { LayoutGrid, MoreHorizontal, Search } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { PaginationControls } from "@/components/ui/pagination-controls";
import { PerPageSelector } from "@/components/ui/per-page-selector";
import { useCollection } from "@/hooks/use-collection";
import { useListFilters } from "@/hooks/use-list-filters";
import { useRealtime } from "@/hooks/use-realtime";
import { formatCpuMem, formatUptime } from "@/lib/format";
import { andFilters, containsAnyFilter, eqFilter } from "@/lib/pocketbase-filters";
import type { HostAppRecord } from "@/types/pocketbase";

const sourceOptions = ["all", "systemd", "pm2", "process"];
const statusOptions = ["all", "running", "stopped", "failed", "unknown"];

const statusVariants: Record<HostAppRecord["status"], "default" | "secondary" | "destructive"> = {
	running: "default",
	stopped: "secondary",
	failed: "destructive",
	unknown: "secondary",
};

export default function AppsPage() {
	const { filters, setSearch, setPage, setPerPage, setCustom, pbOptions } = useListFilters({
		defaultSort: "name",
		searchFields: ["name", "unit", "pm_id"],
		customFilters: { server: "server", source: "source", status: "status" },
	});

	const { data: appsResult, isLoading } = useCollection("host_apps", pbOptions);
	const { data: serversResult } = useCollection("servers", { perPage: 200 });

	const baseFilter = andFilters([
		containsAnyFilter(["name", "unit", "pm_id"], filters.search || undefined),
		eqFilter("server", filters.custom.server === "all" ? undefined : filters.custom.server),
		eqFilter("source", filters.custom.source === "all" ? undefined : filters.custom.source),
	]);
	const statusCount = (status: string) => ({
		perPage: 1,
		filter: andFilters([baseFilter, eqFilter("status", status)]),
		requestKey: null,
	});
	const { data: runningResult } = useCollection("host_apps", statusCount("running"));
	const { data: stoppedResult } = useCollection("host_apps", statusCount("stopped"));
	const { data: failedResult } = useCollection("host_apps", statusCount("failed"));

	useRealtime("host_apps");
	useRealtime("servers", 3000);

	const servers = useMemo(() => serversResult?.items ?? [], [serversResult]);
	const serverNames = useMemo(() => {
		const map = new Map<string, string>();
		for (const server of servers) {
			map.set(server.id, server.name || server.hostname);
		}
		return map;
	}, [servers]);

	const apps = appsResult?.items ?? [];
	const totalApps = appsResult?.totalItems ?? apps.length;

	return (
		<div className="flex flex-col gap-6">
			<div className="flex flex-col gap-1">
				<div className="flex items-center gap-2">
					<LayoutGrid className="h-5 w-5 text-muted-foreground" />
					<h1 className="text-2xl font-semibold">Applications</h1>
					{appsResult && (
						<Badge variant="secondary" className="ml-1">
							{totalApps}
						</Badge>
					)}
				</div>
				<p className="text-sm text-muted-foreground">
					{runningResult && stoppedResult && failedResult
						? `${runningResult.totalItems} running · ${stoppedResult.totalItems} stopped · ${failedResult.totalItems} failed`
						: "systemd units, PM2 processes and listening services across your servers."}
				</p>
			</div>

			<div className="flex flex-wrap items-center gap-3">
				<Select value={filters.custom.server} onValueChange={(value) => setCustom("server", value)}>
					<SelectTrigger className="w-48">
						<SelectValue placeholder="Server" />
					</SelectTrigger>
					<SelectContent>
						<SelectItem value="all">All servers</SelectItem>
						{servers.map((server) => (
							<SelectItem key={server.id} value={server.id}>
								{server.name || server.hostname}
							</SelectItem>
						))}
					</SelectContent>
				</Select>

				<Select value={filters.custom.source} onValueChange={(value) => setCustom("source", value)}>
					<SelectTrigger className="w-36">
						<SelectValue placeholder="Source" />
					</SelectTrigger>
					<SelectContent>
						{sourceOptions.map((source) => (
							<SelectItem key={source} value={source}>
								{source === "all" ? "All sources" : source}
							</SelectItem>
						))}
					</SelectContent>
				</Select>

				<Select value={filters.custom.status} onValueChange={(value) => setCustom("status", value)}>
					<SelectTrigger className="w-36">
						<SelectValue placeholder="Status" />
					</SelectTrigger>
					<SelectContent>
						{statusOptions.map((status) => (
							<SelectItem key={status} value={status}>
								{status === "all" ? "All statuses" : status}
							</SelectItem>
						))}
					</SelectContent>
				</Select>

				<div className="relative w-64">
					<Search className="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
					<Input
						placeholder="Search applications..."
						className="pl-8"
						value={filters.search}
						onChange={(event) => setSearch(event.target.value)}
					/>
				</div>
			</div>

			<div className="rounded-md border">
				<Table>
					<TableHeader>
						<TableRow>
							<TableHead>App</TableHead>
							<TableHead className="w-44">Server</TableHead>
							<TableHead className="w-24">Source</TableHead>
							<TableHead className="w-24">Status</TableHead>
							<TableHead className="w-28">Ports</TableHead>
							<TableHead className="w-32">CPU / Mem</TableHead>
							<TableHead className="w-20">Health</TableHead>
							<TableHead className="w-20">Restarts</TableHead>
							<TableHead className="w-24">Uptime</TableHead>
							<TableHead className="w-10" />
						</TableRow>
					</TableHeader>
					<TableBody>
						{isLoading ? (
							Array.from({ length: 5 }).map((_, row) => (
								<TableRow key={row}>
									{Array.from({ length: 10 }).map((_, col) => (
										<TableCell key={col}>
											<Skeleton className="h-4 w-full" />
										</TableCell>
									))}
								</TableRow>
							))
						) : apps.length === 0 ? (
							<TableRow>
								<TableCell colSpan={10} className="py-8 text-center text-muted-foreground">
									{filters.search
										? "No applications match your filters."
										: "No applications reported."}
								</TableCell>
							</TableRow>
						) : (
							apps.map((app) => (
								<TableRow key={app.id}>
									<TableCell className="text-sm font-medium">{app.name}</TableCell>
									<TableCell className="text-sm">
										<Link to={`/servers/${app.server}`} className="hover:underline">
											{serverNames.get(app.server) ?? app.server}
										</Link>
									</TableCell>
									<TableCell>
										<Badge variant="outline">{app.source}</Badge>
									</TableCell>
									<TableCell>
										<Badge variant={statusVariants[app.status] ?? "secondary"}>{app.status}</Badge>
									</TableCell>
									<TableCell className="text-xs tabular-nums text-muted-foreground">
										{app.ports?.length ? app.ports.join(", ") : "—"}
									</TableCell>
									<TableCell className="text-xs tabular-nums text-muted-foreground">
										{formatCpuMem(app.cpu_percent, app.mem_bytes)}
									</TableCell>
									<TableCell className="text-xs text-muted-foreground">—</TableCell>
									<TableCell className="text-xs tabular-nums text-muted-foreground">
										{app.restarts > 0 ? app.restarts : "—"}
									</TableCell>
									<TableCell className="text-xs text-muted-foreground whitespace-nowrap">
										{app.uptime_seconds > 0 ? formatUptime(app.uptime_seconds) : "—"}
									</TableCell>
									<TableCell>
										<Button
											variant="ghost"
											size="icon"
											className="h-8 w-8"
											disabled
											title="Read-only in this release"
										>
											<MoreHorizontal className="h-4 w-4" />
										</Button>
									</TableCell>
								</TableRow>
							))
						)}
					</TableBody>
				</Table>
			</div>

			<div className="flex items-center justify-between">
				<PerPageSelector value={filters.perPage} onChange={setPerPage} />
				<PaginationControls
					page={filters.page}
					totalPages={appsResult?.totalPages ?? 0}
					totalItems={totalApps}
					onPageChange={setPage}
					label="applications"
				/>
			</div>
		</div>
	);
}
