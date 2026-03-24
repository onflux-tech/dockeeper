import {
	AreaChart,
	Area,
	XAxis,
	YAxis,
	CartesianGrid,
	Tooltip,
	ResponsiveContainer,
} from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import type { MetricPoint } from "@/hooks/use-server-metrics";
import type { TimeRange } from "@/hooks/use-server-metrics";
import type { ValueType } from "recharts/types/component/DefaultTooltipContent";

interface MetricsChartsProps {
	data: MetricPoint[];
	range: TimeRange;
	isLoading: boolean;
}

function formatTime(ts: string, range: TimeRange): string {
	const d = new Date(ts);
	if (range === "7d") {
		return `${String(d.getMonth() + 1).padStart(2, "0")}/${String(d.getDate()).padStart(2, "0")}`;
	}
	return `${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`;
}

function formatBytes(bytes: number): string {
	if (bytes === 0) return "0 B";
	const units = ["B", "KB", "MB", "GB", "TB"];
	const i = Math.floor(Math.log(bytes) / Math.log(1024));
	const val = bytes / 1024 ** i;
	return `${val.toFixed(1)} ${units[i]}`;
}

function formatPercent(v: number): string {
	return `${v.toFixed(1)}%`;
}

function tooltipPercent(v: ValueType | undefined): string {
	return formatPercent(Number(v ?? 0));
}

function tooltipBytes(v: ValueType | undefined): string {
	return formatBytes(Number(v ?? 0));
}

function ChartSkeleton() {
	return (
		<Card>
			<CardHeader>
				<Skeleton className="h-4 w-24" />
			</CardHeader>
			<CardContent>
				<Skeleton className="h-48 w-full" />
			</CardContent>
		</Card>
	);
}

const chartColors = {
	cpu: "var(--color-primary)",
	mem: "var(--color-chart-2)",
	disk: "var(--color-chart-3)",
	netRx: "var(--color-chart-4)",
	netTx: "var(--color-chart-5)",
};

export function MetricsCharts({ data, range, isLoading }: MetricsChartsProps) {
	if (isLoading && data.length === 0) {
		return (
			<div className="grid gap-4 md:grid-cols-2">
				{Array.from({ length: 4 }).map((_, i) => (
					<ChartSkeleton key={i} />
				))}
			</div>
		);
	}

	const chartData = data.map((p) => ({
		...p,
		time: formatTime(p.timestamp, range),
		mem_used_pct: p.disk_total > 0 ? (p.disk_used / p.disk_total) * 100 : 0,
	}));

	return (
		<div className="grid gap-4 md:grid-cols-2">
			{/* CPU */}
			<Card>
				<CardHeader className="pb-2">
					<CardTitle className="text-sm">CPU Usage</CardTitle>
				</CardHeader>
				<CardContent>
					<ResponsiveContainer width="100%" height={200}>
						<AreaChart data={chartData}>
							<CartesianGrid strokeDasharray="3 3" className="stroke-border" />
							<XAxis dataKey="time" tick={{ fontSize: 11 }} className="text-muted-foreground" />
							<YAxis
								domain={[0, 100]}
								tick={{ fontSize: 11 }}
								tickFormatter={formatPercent}
								className="text-muted-foreground"
							/>
							<Tooltip
								formatter={tooltipPercent}
								contentStyle={{
									backgroundColor: "var(--color-popover)",
									borderColor: "var(--color-border)",
									color: "var(--color-popover-foreground)",
								}}
								labelStyle={{ color: "var(--color-muted-foreground)" }}
							/>
							<defs>
								<linearGradient id="fillCpu" x1="0" y1="0" x2="0" y2="1">
									<stop offset="0%" stopColor={chartColors.cpu} stopOpacity={0.3} />
									<stop offset="100%" stopColor={chartColors.cpu} stopOpacity={0} />
								</linearGradient>
							</defs>
							<Area
								type="monotone"
								dataKey="cpu_percent"
								stroke={chartColors.cpu}
								fill="url(#fillCpu)"
								strokeWidth={1.5}
								isAnimationActive={false}
								name="CPU"
							/>
						</AreaChart>
					</ResponsiveContainer>
				</CardContent>
			</Card>

			{/* Memory */}
			<Card>
				<CardHeader className="pb-2">
					<CardTitle className="text-sm">Memory Usage</CardTitle>
				</CardHeader>
				<CardContent>
					<ResponsiveContainer width="100%" height={200}>
						<AreaChart data={chartData}>
							<CartesianGrid strokeDasharray="3 3" className="stroke-border" />
							<XAxis dataKey="time" tick={{ fontSize: 11 }} className="text-muted-foreground" />
							<YAxis
								domain={[0, 100]}
								tick={{ fontSize: 11 }}
								tickFormatter={formatPercent}
								className="text-muted-foreground"
							/>
							<Tooltip
								formatter={tooltipPercent}
								contentStyle={{
									backgroundColor: "var(--color-popover)",
									borderColor: "var(--color-border)",
									color: "var(--color-popover-foreground)",
								}}
								labelStyle={{ color: "var(--color-muted-foreground)" }}
							/>
							<defs>
								<linearGradient id="fillMem" x1="0" y1="0" x2="0" y2="1">
									<stop offset="0%" stopColor={chartColors.mem} stopOpacity={0.3} />
									<stop offset="100%" stopColor={chartColors.mem} stopOpacity={0} />
								</linearGradient>
							</defs>
							<Area
								type="monotone"
								dataKey="mem_percent"
								stroke={chartColors.mem}
								fill="url(#fillMem)"
								strokeWidth={1.5}
								isAnimationActive={false}
								name="Memory"
							/>
						</AreaChart>
					</ResponsiveContainer>
				</CardContent>
			</Card>

			{/* Disk */}
			<Card>
				<CardHeader className="pb-2">
					<CardTitle className="text-sm">Disk Usage</CardTitle>
				</CardHeader>
				<CardContent>
					<ResponsiveContainer width="100%" height={200}>
						<AreaChart data={chartData}>
							<CartesianGrid strokeDasharray="3 3" className="stroke-border" />
							<XAxis dataKey="time" tick={{ fontSize: 11 }} className="text-muted-foreground" />
							<YAxis
								tick={{ fontSize: 11 }}
								tickFormatter={formatBytes}
								className="text-muted-foreground"
							/>
							<Tooltip
								formatter={tooltipBytes}
								contentStyle={{
									backgroundColor: "var(--color-popover)",
									borderColor: "var(--color-border)",
									color: "var(--color-popover-foreground)",
								}}
								labelStyle={{ color: "var(--color-muted-foreground)" }}
							/>
							<defs>
								<linearGradient id="fillDisk" x1="0" y1="0" x2="0" y2="1">
									<stop offset="0%" stopColor={chartColors.disk} stopOpacity={0.3} />
									<stop offset="100%" stopColor={chartColors.disk} stopOpacity={0} />
								</linearGradient>
							</defs>
							<Area
								type="monotone"
								dataKey="disk_used"
								stroke={chartColors.disk}
								fill="url(#fillDisk)"
								strokeWidth={1.5}
								isAnimationActive={false}
								name="Disk Used"
							/>
						</AreaChart>
					</ResponsiveContainer>
				</CardContent>
			</Card>

			{/* Network I/O */}
			<Card>
				<CardHeader className="pb-2">
					<CardTitle className="text-sm">Network I/O</CardTitle>
				</CardHeader>
				<CardContent>
					<ResponsiveContainer width="100%" height={200}>
						<AreaChart data={chartData}>
							<CartesianGrid strokeDasharray="3 3" className="stroke-border" />
							<XAxis dataKey="time" tick={{ fontSize: 11 }} className="text-muted-foreground" />
							<YAxis
								tick={{ fontSize: 11 }}
								tickFormatter={formatBytes}
								className="text-muted-foreground"
							/>
							<Tooltip
								formatter={tooltipBytes}
								contentStyle={{
									backgroundColor: "var(--color-popover)",
									borderColor: "var(--color-border)",
									color: "var(--color-popover-foreground)",
								}}
								labelStyle={{ color: "var(--color-muted-foreground)" }}
							/>
							<defs>
								<linearGradient id="fillRx" x1="0" y1="0" x2="0" y2="1">
									<stop offset="0%" stopColor={chartColors.netRx} stopOpacity={0.3} />
									<stop offset="100%" stopColor={chartColors.netRx} stopOpacity={0} />
								</linearGradient>
								<linearGradient id="fillTx" x1="0" y1="0" x2="0" y2="1">
									<stop offset="0%" stopColor={chartColors.netTx} stopOpacity={0.3} />
									<stop offset="100%" stopColor={chartColors.netTx} stopOpacity={0} />
								</linearGradient>
							</defs>
							<Area
								type="monotone"
								dataKey="net_rx"
								stroke={chartColors.netRx}
								fill="url(#fillRx)"
								strokeWidth={1.5}
								isAnimationActive={false}
								name="RX"
							/>
							<Area
								type="monotone"
								dataKey="net_tx"
								stroke={chartColors.netTx}
								fill="url(#fillTx)"
								strokeWidth={1.5}
								isAnimationActive={false}
								name="TX"
							/>
						</AreaChart>
					</ResponsiveContainer>
				</CardContent>
			</Card>
		</div>
	);
}
