import type { DiskPartition } from "@/types/pocketbase";

export interface DiskUsage {
	used: number;
	total: number;
}

interface DiskSample {
	disk_used: number;
	disk_total: number;
	partitions?: DiskPartition[];
}

export function diskUsage(sample: DiskSample | undefined, view: string): DiskUsage | null {
	if (!sample) return null;

	if (view === "all") {
		const partitions = sample.partitions ?? [];
		if (partitions.length > 0) {
			const total = partitions.reduce<DiskUsage>(
				(acc, partition) => ({
					used: acc.used + partition.used,
					total: acc.total + partition.total,
				}),
				{ used: 0, total: 0 },
			);
			if (!partitions.some((partition) => partition.mountpoint === "/")) {
				return {
					used: total.used + sample.disk_used,
					total: total.total + sample.disk_total,
				};
			}
			return total;
		}
		return { used: sample.disk_used, total: sample.disk_total };
	}

	const partition = sample.partitions?.find((p) => p.mountpoint === view);
	return partition ? { used: partition.used, total: partition.total } : null;
}

export function buildDiskOptions(partitions: DiskPartition[]): { value: string; label: string }[] {
	const options = [{ value: "all", label: "All disks (sum)" }];
	const seen = new Set<string>();
	for (const partition of partitions) {
		if (seen.has(partition.mountpoint)) continue;
		seen.add(partition.mountpoint);
		options.push({ value: partition.mountpoint, label: partition.mountpoint });
	}
	return options;
}

export function latestPartitions(samples: { partitions?: DiskPartition[] }[]): DiskPartition[] {
	for (let i = samples.length - 1; i >= 0; i--) {
		const partitions = samples[i].partitions;
		if (partitions?.length) return partitions;
	}
	return [];
}
