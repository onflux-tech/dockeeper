import { useState } from "react";
import { useParams, Link } from "react-router";
import { ArrowLeft, Pencil } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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
import { EditServerDialog } from "@/components/server/edit-server-dialog";
import { PaginationControls } from "@/components/ui/pagination-controls";
import { TagSelector } from "@/components/tag-selector";
import { andFilters, containsAnyFilter, eqFilter } from "@/lib/pocketbase-filters";

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
	const [containerSearch, setContainerSearch] = useState("");
	const [containerStatus, setContainerStatus] = useState("all");
	const [editOpen, setEditOpen] = useState(false);

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
		enabled: !!id,
	});

	useRealtime("servers");
	useRealtime("server_metrics_raw");
	useRealtime("containers");

	const containers = containersResult?.items ?? [];
	const metrics = metricsData ?? [];

	const statusVariant =
		server?.status === "online"
			? "default"
			: server?.status === "error"
				? "destructive"
				: "secondary";

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
						{id && server && (
							<TagSelector resourceType="server" resourceId={id} />
						)}
						{server && (
							<Button variant="outline" size="sm" onClick={() => setEditOpen(true)}>
								<Pencil className="mr-1.5 h-3.5 w-3.5" />
								Edit
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

					<MetricsCharts data={metrics} range={range} isLoading={loadingMetrics} />
				</TabsContent>

				{/* Containers Tab */}
				<TabsContent value="containers" className="mt-4">
					<ContainerTable
						containers={containers}
						isLoading={loadingContainers}
						search={containerSearch}
						onSearchChange={(value) => {
							setContainerSearch(value);
							setContainersPage(1);
						}}
						statusFilter={containerStatus}
						onStatusFilterChange={(value) => {
							setContainerStatus(value);
							setContainersPage(1);
						}}
					/>
					<PaginationControls
						className="mt-4 flex items-center justify-between gap-3"
						page={containersPage}
						totalPages={containersResult?.totalPages ?? 0}
						totalItems={containersResult?.totalItems ?? containers.length}
						onPageChange={setContainersPage}
						label="containers"
					/>
				</TabsContent>
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
