import { useQuery, keepPreviousData } from "@tanstack/react-query";
import pb from "@/lib/pb";
import type { CollectionName } from "@/types/pocketbase";

export type TimeRange = "1h" | "6h" | "24h" | "7d";

export interface MetricPoint {
	timestamp: string;
	cpu_percent: number;
	mem_percent: number;
	disk_used: number;
	disk_total: number;
	net_rx: number;
	net_tx: number;
}

interface RangeConfig {
	collection: CollectionName;
	hours: number;
}

const rangeMap: Record<TimeRange, RangeConfig> = {
	"1h": { collection: "server_metrics_raw", hours: 1 },
	"6h": { collection: "server_metrics_raw", hours: 6 },
	"24h": { collection: "server_metrics_hourly", hours: 24 },
	"7d": { collection: "server_metrics_daily", hours: 168 },
};

function cutoffTimestamp(hours: number): string {
	const d = new Date(Date.now() - hours * 3600 * 1000);
	return d.toISOString().replace("T", " ");
}

/**
 * Fetches server metrics for a given time range.
 * Automatically selects the best collection (raw / hourly / daily) based on range.
 * Uses `keepPreviousData` so charts don't flash when switching ranges.
 */
export function useServerMetrics(serverId: string | undefined, range: TimeRange) {
	const { collection, hours } = rangeMap[range];

	return useQuery<MetricPoint[]>({
		queryKey: ["server-metrics", serverId, range],
		queryFn: async () => {
			const cutoff = cutoffTimestamp(hours);
			const result = await pb.collection(collection).getList(1, 500, {
				filter: `server = "${serverId}" && timestamp >= "${cutoff}"`,
				sort: "timestamp",
			});
			return result.items.map((r) => ({
				timestamp: r.timestamp as string,
				cpu_percent: r.cpu_percent as number,
				mem_percent: r.mem_percent as number,
				disk_used: r.disk_used as number,
				disk_total: r.disk_total as number,
				net_rx: r.net_rx as number,
				net_tx: r.net_tx as number,
			}));
		},
		enabled: !!serverId,
		placeholderData: keepPreviousData,
	});
}
