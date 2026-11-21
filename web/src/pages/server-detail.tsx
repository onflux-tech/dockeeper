import { useState } from "react";
import { useParams, Link } from "react-router";
import { ArrowLeft, Pencil, Download, Search } from "lucide-react";
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useRecord } from "@/hooks/use-record";
import { useCollection } from "@/hooks/use-collection";
import { useRealtime } from "@/hooks/use-realtime";
import { useServerMetrics, type TimeRange } from "@/hooks/use-server-metrics";
import { ServerInfo } from "@/components/server/server-info";
import { MetricsCharts } from "@/components/server/metrics-charts";
import { ContainerTable } from "@/components/server/container-table";
import { HostNotice } from "@/components/server/host-notice";
import { HostSummaryCards } from "@/components/server/host-summary-cards";
import { HostAppsTable } from "@/components/server/host-apps-table";
import { HostProcessesTable } from "@/components/server/host-processes-table";
import { useHostApps, useHostProcesses } from "@/hooks/use-host-inventory";
import { CONTAINER_SORT_FIELDS, type SortDir, type SortKey } from "@/lib/container-constants";
import { statusOptions } from "@/lib/container-constants";
import { EditServerDialog } from "@/components/server/edit-server-dialog";
import { PaginationControls } from "@/components/ui/pagination-controls";
import { TagSelector } from "@/components/tag-selector";
import { andFilters, containsAnyFilter, eqFilter } from "@/lib/pocketbase-filters";
import { useAgentUpdate, useLatestRelease } from "@/hooks/use-settings";
import { isVersionNewer } from "@/lib/version";

const timeRangeLabels: Record<TimeRange, string> = {
	"1h": "Last 1 hour",
	"6h": "Last 6 hours",
	"24h": "Last 24 hours",
	"7d": "Last 7 days",
};

export default function ServerDetailPage() {
	const { id } = useParams();
	const [range, setRange] = useState<TimeRange>("1h");
	const [containersPage, setContainersPage] = useState(1);
	const [containerSortKey, setContainerSortKey] = useState<SortKey>("name");
	const [containerSortDir, setContainerSortDir] = useState<SortDir>("asc");
	const [containerSearch, setContainerSearch] = useState("");
	const [containerStatus, setContainerStatus] = useState("all");
	const [editOpen, setEditOpen] = useState(false);
	const agentUpdate = useAgentUpdate();
	const { data: releaseInfo } = useLatestRelease();

	const { data: server, isLoading: loadingServer } = useRecord("servers", id);
	const { data: metricsData, isLoading: loadingMetrics } = useServerMetrics(id, range);
	const { data: containersResult, isLoading: loadingContainers } = useCollection("containers", {
		page: containersPage,
		perPage: 25,
		filter: andFilters([
			eqFilter("server", id),
			containsAnyFilter(["name", "image"], containerSearch),
			containerStatus !== "all" ? eqFilter("status", containerStatus) : "",
		]),
		sort: `${containerSortDir === "desc" ? "-" : ""}${CONTAINER_SORT_FIELDS[containerSortKey]}`,
		enabled: !!id,
	});

	useRealtime("servers");
	useRealtime("server_metrics_raw");
	useRealtime("containers");

	const hostOnly = server?.orchestrator === "none";
	const { data: hostApps, isLoading: loadingHostApps } = useHostApps(hostOnly ? id : undefined);
	const { data: hostProcesses, isLoading: loadingHostProcesses } = useHostProcesses(
		hostOnly ? id : undefined,
	);

	const containers = containersResult?.items ?? [];
	const metrics = metricsData ?? [];

	function handleContainerSort(key: SortKey) {
		setContainersPage(1);
		if (key === containerSortKey) {
			setContainerSortDir(containerSortDir === "asc" ? "desc" : "asc");
		} else {
			setContainerSortKey(key);
			setContainerSortDir("asc");
		}
	}

	const statusVariant =
		server?.status === "online"
			? "default"
			: server?.status === "error"
				? "destructive"
				: "secondary";

	const agentNeedsUpdate = releaseInfo?.latest
		? isVersionNewer(server?.agent_version ?? "", releaseInfo.latest)
		: false;

	return (
		<div className="flex flex-col gap-6">
			{/* Header */}
			<div className="flex items-center gap-3">
				<Button variant="ghost" size="icon" className="h-8 w-8" asChild>
					<Link to="/servers">
						<ArrowLeft className="h-4 w-4" />
					</Link>
				</Button>
				{loadingServer ? (
					<>
						<Skeleton className="h-7 w-40" />
						<Skeleton className="h-5 w-16 rounded-full" />
					</>
				) : (
					<>
						<h1 className="text-2xl font-semibold">
							{server?.name || server?.hostname || "Server"}
						</h1>
						{server && <Badge variant={statusVariant}>{server.status}</Badge>}
						{id && server && <TagSelector resourceType="server" resourceId={id} />}
						{server && (
							<Button variant="outline" size="sm" onClick={() => setEditOpen(true)}>
								<Pencil className="mr-1.5 h-3.5 w-3.5" />
								Edit
							</Button>
						)}
						{server?.status === "online" && agentNeedsUpdate && (
							<Button
								variant="outline"
								size="sm"
								disabled={agentUpdate.isPending}
								onClick={() => {
									if (id) {
										agentUpdate.mutate(id);
									}
								}}
							>
								<Download className="mr-1.5 h-3.5 w-3.5" />
								{agentUpdate.isPending ? "Updating..." : "Update Agent"}
							</Button>
						)}
					</>
				)}
			</div>

			{/* Tabs */}
			<Tabs defaultValue="overview">
				<TabsList>
					<TabsTrigger value="overview">Overview</TabsTrigger>
					<TabsTrigger value="containers">
						Containers ({containersResult?.totalItems ?? containers.length})
					</TabsTrigger>
					{hostOnly && <TabsTrigger value="host">Host</TabsTrigger>}
				</TabsList>

				{/* Overview Tab */}
				<TabsContent value="overview" className="flex flex-col gap-6 mt-4">
					<ServerInfo server={server} isLoading={loadingServer} />

					<div className="flex items-center justify-between">
						<h2 className="text-lg font-medium">Metrics</h2>
						<Select value={range} onValueChange={(v) => setRange(v as TimeRange)}>
							<SelectTrigger className="w-44">
								<SelectValue />
							</SelectTrigger>
							<SelectContent>
								{(Object.keys(timeRangeLabels) as TimeRange[]).map((r) => (
									<SelectItem key={r} value={r}>
										{timeRangeLabels[r]}
									</SelectItem>
								))}
							</SelectContent>
						</Select>
					</div>

					<MetricsCharts
						key={id}
						data={metrics}
						range={range}
						isLoading={loadingMetrics}
						serverId={id ?? ""}
					/>
				</TabsContent>

				{/* Containers Tab */}
				<TabsContent value="containers" className="flex flex-col gap-4 mt-4">
					{hostOnly ? (
						<div className="rounded-md border border-dashed p-8 text-center">
							<p className="text-sm font-medium">No container runtime</p>
							<p className="mt-1 text-sm text-muted-foreground">
								Docker is not available on this server. Check the Host tab for systemd units, PM2
								applications and processes.
							</p>
						</div>
					) : (
						<>
							<div className="flex items-center gap-3">
								<Select
									value={containerStatus}
									onValueChange={(value) => {
										setContainerStatus(value);
										setContainersPage(1);
									}}
								>
									<SelectTrigger className="w-36">
										<SelectValue placeholder="Status" />
									</SelectTrigger>
									<SelectContent>
										{statusOptions.map((s) => (
											<SelectItem key={s} value={s}>
												{s === "all" ? "All statuses" : s}
											</SelectItem>
										))}
									</SelectContent>
								</Select>
								<div className="relative w-64">
									<Search className="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
									<Input
										placeholder="Search containers..."
										className="pl-8"
										value={containerSearch}
										onChange={(e) => {
											setContainerSearch(e.target.value);
											setContainersPage(1);
										}}
									/>
								</div>
							</div>
							<ContainerTable
								containers={containers}
								isLoading={loadingContainers}
								sortKey={containerSortKey}
								sortDir={containerSortDir}
								onSortChange={handleContainerSort}
							/>
							<PaginationControls
								className="mt-4 flex items-center justify-between gap-3"
								page={containersPage}
								totalPages={containersResult?.totalPages ?? 0}
								totalItems={containersResult?.totalItems ?? containers.length}
								onPageChange={setContainersPage}
								label="containers"
							/>
						</>
					)}
				</TabsContent>

				{hostOnly && (
					<TabsContent value="host" className="flex flex-col gap-4 mt-4">
						<HostNotice />
						<HostSummaryCards metrics={metrics} />
						<div className="flex items-center justify-between">
							<h2 className="text-lg font-medium">Applications</h2>
							<Button variant="ghost" size="sm" asChild>
								<Link to="/apps">View all</Link>
							</Button>
						</div>
						<HostAppsTable apps={hostApps} isLoading={loadingHostApps} />
						<h2 className="text-lg font-medium">Top processes</h2>
						<HostProcessesTable processes={hostProcesses} isLoading={loadingHostProcesses} />
					</TabsContent>
				)}
			</Tabs>

			{server && (
				<EditServerDialog
					key={server.id + server.updated}
					open={editOpen}
					onOpenChange={setEditOpen}
					server={server}
				/>
			)}
		</div>
	);
}
