import { Activity, HardDrive, MemoryStick, Network } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatBytes } from "@/lib/format";
import type { MetricPoint } from "@/hooks/use-server-metrics";

interface HostSummaryCardsProps {
	metrics: MetricPoint[];
}

interface SummaryCardProps {
	title: string;
	icon: React.ReactNode;
	value: string;
	detail: string;
}

function SummaryCard({ title, icon, value, detail }: SummaryCardProps) {
	return (
		<Card>
			<CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
				<CardTitle className="text-sm font-medium">{title}</CardTitle>
				<span className="text-muted-foreground">{icon}</span>
			</CardHeader>
			<CardContent>
				<div className="text-2xl font-semibold tabular-nums">{value}</div>
				<p className="text-xs text-muted-foreground">{detail}</p>
			</CardContent>
		</Card>
	);
}

export function HostSummaryCards({ metrics }: HostSummaryCardsProps) {
	const latest = metrics[metrics.length - 1];

	if (!latest) {
		return (
			<div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
				<SummaryCard
					title="CPU"
					icon={<Activity className="h-4 w-4" />}
					value="—"
					detail="No metrics yet"
				/>
				<SummaryCard
					title="Memory"
					icon={<MemoryStick className="h-4 w-4" />}
					value="—"
					detail="No metrics yet"
				/>
				<SummaryCard
					title="Disk"
					icon={<HardDrive className="h-4 w-4" />}
					value="—"
					detail="No metrics yet"
				/>
				<SummaryCard
					title="Network"
					icon={<Network className="h-4 w-4" />}
					value="—"
					detail="No metrics yet"
				/>
			</div>
		);
	}

	return (
		<div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
			<SummaryCard
				title="CPU"
				icon={<Activity className="h-4 w-4" />}
				value={`${latest.cpu_percent.toFixed(1)}%`}
				detail={`${metrics.length} samples in range`}
			/>
			<SummaryCard
				title="Memory"
				icon={<MemoryStick className="h-4 w-4" />}
				value={`${latest.mem_percent.toFixed(1)}%`}
				detail={`${formatBytes(latest.mem_available)} available`}
			/>
			<SummaryCard
				title="Disk"
				icon={<HardDrive className="h-4 w-4" />}
				value={formatBytes(latest.disk_used)}
				detail={`of ${formatBytes(latest.disk_total)}`}
			/>
			<SummaryCard
				title="Network"
				icon={<Network className="h-4 w-4" />}
				value={`↓ ${formatBytes(latest.net_rx)}`}
				detail={`↑ ${formatBytes(latest.net_tx)}`}
			/>
		</div>
	);
}
