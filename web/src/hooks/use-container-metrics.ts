import { useQuery, keepPreviousData } from "@tanstack/react-query";
import pb from "@/lib/pb";

export interface ContainerMetricPoint {
	timestamp: string;
	cpu_percent: number;
	mem_used: number;
	mem_limit: number;
	net_rx: number;
	net_tx: number;
	block_read: number;
	block_write: number;
	pids: number;
}

function cutoffTimestamp(hours: number): string {
	const d = new Date(Date.now() - hours * 3600 * 1000);
	return d.toISOString().replace("T", " ");
}

export function useContainerMetrics(containerId: string | undefined) {
	return useQuery<ContainerMetricPoint[]>({
		queryKey: ["container-metrics", containerId],
		queryFn: async () => {
			const cutoff = cutoffTimestamp(1);
			const result = await pb
				.collection("container_metrics_raw")
				.getList(1, 200, {
					filter: `container = "${containerId}" && timestamp >= "${cutoff}"`,
					sort: "timestamp",
				});
			return result.items.map((r) => ({
				timestamp: r.timestamp as string,
				cpu_percent: r.cpu_percent as number,
				mem_used: r.mem_used as number,
				mem_limit: r.mem_limit as number,
				net_rx: r.net_rx as number,
				net_tx: r.net_tx as number,
				block_read: r.block_read as number,
				block_write: r.block_write as number,
				pids: r.pids as number,
			}));
		},
		enabled: !!containerId,
		placeholderData: keepPreviousData,
		refetchInterval: 10_000,
	});
}
