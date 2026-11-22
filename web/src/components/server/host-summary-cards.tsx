import { ArrowDownUp, Cpu, HardDrive, MemoryStick } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
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
		<Card className="py-0">
			<CardContent className="flex items-center gap-4 p-6">
				<div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-md bg-muted text-muted-foreground">
					{icon}
				</div>
				<div className="min-w-0">
					<div className="text-2xl font-bold tabular-nums">{value}</div>
					<p className="text-xs leading-tight text-muted-foreground">
						{title} · {detail}
					</p>
				</div>
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
					icon={<Cpu className="h-4 w-4" />}
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
					icon={<ArrowDownUp className="h-4 w-4" />}
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
				icon={<Cpu className="h-4 w-4" />}
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
				icon={<ArrowDownUp className="h-4 w-4" />}
				value={`↓ ${formatBytes(latest.net_rx)}`}
				detail={`↑ ${formatBytes(latest.net_tx)}`}
			/>
		</div>
	);
}
