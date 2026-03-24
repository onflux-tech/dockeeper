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
import type { ContainerMetricPoint } from "@/hooks/use-container-metrics";
import type { ValueType } from "recharts/types/component/DefaultTooltipContent";

interface ContainerMetricsProps {
	data: ContainerMetricPoint[];
	isLoading: boolean;
}

function formatTime(ts: string): string {
	const d = new Date(ts);
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

const chartColors = {
	cpu: "var(--color-primary)",
	mem: "var(--color-chart-2)",
	blockRead: "var(--color-chart-3)",
	blockWrite: "var(--color-chart-5)",
	netRx: "var(--color-chart-4)",
	netTx: "var(--color-chart-5)",
};

const tooltipStyle = {
	contentStyle: {
		backgroundColor: "var(--color-popover)",
		borderColor: "var(--color-border)",
		color: "var(--color-popover-foreground)",
	},
	labelStyle: { color: "var(--color-muted-foreground)" },
};

export function ContainerMetrics({ data, isLoading }: ContainerMetricsProps) {
	if (isLoading && data.length === 0) {
		return (
			<div className="grid gap-4 md:grid-cols-2">
				{Array.from({ length: 4 }).map((_, i) => (
					<Card key={i}>
						<CardHeader>
							<Skeleton className="h-4 w-24" />
						</CardHeader>
						<CardContent>
							<Skeleton className="h-48 w-full" />
						</CardContent>
					</Card>
				))}
			</div>
		);
	}

	if (data.length === 0) {
		return (
			<div className="text-sm text-muted-foreground text-center py-8">
				No metrics data available yet.
			</div>
		);
	}

	const chartData = data.map((p) => ({
		...p,
		time: formatTime(p.timestamp),
		mem_percent: p.mem_limit > 0 ? (p.mem_used / p.mem_limit) * 100 : 0,
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
							<CartesianGrid
								strokeDasharray="3 3"
								className="stroke-border"
							/>
							<XAxis
								dataKey="time"
								tick={{ fontSize: 11 }}
								className="text-muted-foreground"
							/>
							<YAxis
								domain={[0, "auto"]}
								tick={{ fontSize: 11 }}
								tickFormatter={formatPercent}
								className="text-muted-foreground"
							/>
							<Tooltip
								formatter={tooltipPercent}
								{...tooltipStyle}
							/>
							<defs>
								<linearGradient id="fillCpuC" x1="0" y1="0" x2="0" y2="1">
									<stop
										offset="0%"
										stopColor={chartColors.cpu}
										stopOpacity={0.3}
									/>
									<stop
										offset="100%"
										stopColor={chartColors.cpu}
										stopOpacity={0}
									/>
								</linearGradient>
							</defs>
							<Area
								type="monotone"
								dataKey="cpu_percent"
								stroke={chartColors.cpu}
								fill="url(#fillCpuC)"
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
							<CartesianGrid
								strokeDasharray="3 3"
								className="stroke-border"
							/>
							<XAxis
								dataKey="time"
								tick={{ fontSize: 11 }}
								className="text-muted-foreground"
							/>
							<YAxis
								tick={{ fontSize: 11 }}
								tickFormatter={formatBytes}
								className="text-muted-foreground"
							/>
							<Tooltip
								formatter={tooltipBytes}
								{...tooltipStyle}
							/>
							<defs>
								<linearGradient id="fillMemC" x1="0" y1="0" x2="0" y2="1">
									<stop
										offset="0%"
										stopColor={chartColors.mem}
										stopOpacity={0.3}
									/>
									<stop
										offset="100%"
										stopColor={chartColors.mem}
										stopOpacity={0}
									/>
								</linearGradient>
							</defs>
							<Area
								type="monotone"
								dataKey="mem_used"
								stroke={chartColors.mem}
								fill="url(#fillMemC)"
								strokeWidth={1.5}
								isAnimationActive={false}
								name="Memory"
							/>
						</AreaChart>
					</ResponsiveContainer>
				</CardContent>
			</Card>

			{/* Block I/O */}
			<Card>
				<CardHeader className="pb-2">
					<CardTitle className="text-sm">Block I/O</CardTitle>
				</CardHeader>
				<CardContent>
					<ResponsiveContainer width="100%" height={200}>
						<AreaChart data={chartData}>
							<CartesianGrid
								strokeDasharray="3 3"
								className="stroke-border"
							/>
							<XAxis
								dataKey="time"
								tick={{ fontSize: 11 }}
								className="text-muted-foreground"
							/>
							<YAxis
								tick={{ fontSize: 11 }}
								tickFormatter={formatBytes}
								className="text-muted-foreground"
							/>
							<Tooltip
								formatter={tooltipBytes}
								{...tooltipStyle}
							/>
							<defs>
								<linearGradient id="fillBlockRead" x1="0" y1="0" x2="0" y2="1">
									<stop
										offset="0%"
										stopColor={chartColors.blockRead}
										stopOpacity={0.3}
									/>
									<stop
										offset="100%"
										stopColor={chartColors.blockRead}
										stopOpacity={0}
									/>
								</linearGradient>
								<linearGradient id="fillBlockWrite" x1="0" y1="0" x2="0" y2="1">
									<stop
										offset="0%"
										stopColor={chartColors.blockWrite}
										stopOpacity={0.3}
									/>
									<stop
										offset="100%"
										stopColor={chartColors.blockWrite}
										stopOpacity={0}
									/>
								</linearGradient>
							</defs>
							<Area
								type="monotone"
								dataKey="block_read"
								stroke={chartColors.blockRead}
								fill="url(#fillBlockRead)"
								strokeWidth={1.5}
								isAnimationActive={false}
								name="Read"
							/>
							<Area
								type="monotone"
								dataKey="block_write"
								stroke={chartColors.blockWrite}
								fill="url(#fillBlockWrite)"
								strokeWidth={1.5}
								isAnimationActive={false}
								name="Write"
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
							<CartesianGrid
								strokeDasharray="3 3"
								className="stroke-border"
							/>
							<XAxis
								dataKey="time"
								tick={{ fontSize: 11 }}
								className="text-muted-foreground"
							/>
							<YAxis
								tick={{ fontSize: 11 }}
								tickFormatter={formatBytes}
								className="text-muted-foreground"
							/>
							<Tooltip
								formatter={tooltipBytes}
								{...tooltipStyle}
							/>
							<defs>
								<linearGradient id="fillNetRxC" x1="0" y1="0" x2="0" y2="1">
									<stop
										offset="0%"
										stopColor={chartColors.netRx}
										stopOpacity={0.3}
									/>
									<stop
										offset="100%"
										stopColor={chartColors.netRx}
										stopOpacity={0}
									/>
								</linearGradient>
								<linearGradient id="fillNetTxC" x1="0" y1="0" x2="0" y2="1">
									<stop
										offset="0%"
										stopColor={chartColors.netTx}
										stopOpacity={0.3}
									/>
									<stop
										offset="100%"
										stopColor={chartColors.netTx}
										stopOpacity={0}
									/>
								</linearGradient>
							</defs>
							<Area
								type="monotone"
								dataKey="net_rx"
								stroke={chartColors.netRx}
								fill="url(#fillNetRxC)"
								strokeWidth={1.5}
								isAnimationActive={false}
								name="RX"
							/>
							<Area
								type="monotone"
								dataKey="net_tx"
								stroke={chartColors.netTx}
								fill="url(#fillNetTxC)"
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
