import { useState } from "react";
import { ServerCog } from "lucide-react";
import { useCollection } from "@/hooks/use-collection";
import { useRealtime } from "@/hooks/use-realtime";
import { StatsOverview } from "@/components/dashboard/stats-overview";
import {
	ServerCard,
	ServerCardSkeleton,
} from "@/components/dashboard/server-card";
import type { SparklinePoint } from "@/components/dashboard/sparkline-chart";
import type { ServerMetricsRawRecord } from "@/types/pocketbase";

/**
 * Build sparkline data for a given server from raw metrics.
 * Returns the last `limit` points sorted chronologically.
 */
function buildSparkline(
	metrics: ServerMetricsRawRecord[],
	serverId: string,
	field: "cpu_percent" | "mem_percent",
	limit = 30,
): SparklinePoint[] {
	return metrics
		.filter((m) => m.server === serverId)
		.sort((a, b) => a.timestamp.localeCompare(b.timestamp))
		.slice(-limit)
		.map((m) => ({ value: m[field] }));
}

export default function DashboardPage() {
	// ── data fetching ────────────────────────────────────────────────────
	const [fiveMinAgo] = useState(() => {
		const d = new Date(Date.now() - 5 * 60 * 1000);
		return d.toISOString().replace("T", " ");
	});

	const { data: serversResult, isLoading: loadingServers } =
		useCollection("servers");
	const { data: containersResult, isLoading: loadingContainers } =
		useCollection("containers");
	const { data: metricsResult } = useCollection("server_metrics_raw", {
		filter: `timestamp >= "${fiveMinAgo}"`,
		sort: "timestamp",
		perPage: 500,
	});

	// ── realtime ─────────────────────────────────────────────────────────
	useRealtime("servers");
	useRealtime("server_metrics_raw");

	const servers = serversResult?.items ?? [];
	const containers = containersResult?.items ?? [];
	const metrics = metricsResult?.items ?? [];
	const isLoading = loadingServers || loadingContainers;

	// ── render ────────────────────────────────────────────────────────────
	return (
		<div className="flex flex-col gap-6">
			<h1 className="text-2xl font-semibold">Dashboard</h1>

			<StatsOverview
				servers={servers}
				containers={containers}
				isLoading={isLoading}
			/>

			<div>
				<h2 className="mb-3 text-lg font-medium">Servers</h2>

				{isLoading ? (
					<div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
						{Array.from({ length: 6 }).map((_, i) => (
							<ServerCardSkeleton key={i} />
						))}
					</div>
				) : servers.length === 0 ? (
					<div className="flex flex-col items-center justify-center gap-2 rounded-lg border border-dashed py-16 text-center">
						<ServerCog className="h-10 w-10 text-muted-foreground/50" />
						<p className="text-sm text-muted-foreground">
							No servers registered yet.
						</p>
						<p className="text-xs text-muted-foreground/70">
							Enroll an agent to see your infrastructure here.
						</p>
					</div>
				) : (
					<div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
						{servers.map((server) => (
							<ServerCard
								key={server.id}
								server={server}
								cpuData={buildSparkline(metrics, server.id, "cpu_percent")}
								memData={buildSparkline(metrics, server.id, "mem_percent")}
							/>
						))}
					</div>
				)}
			</div>
		</div>
	);
}
