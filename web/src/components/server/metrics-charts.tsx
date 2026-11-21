import { useMemo, useState } from "react";
import {
	AreaChart,
	Area,
	BarChart,
	Bar,
	LineChart,
	Line,
	XAxis,
	YAxis,
	CartesianGrid,
	Tooltip,
	ResponsiveContainer,
	Legend,
} from "recharts";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { buildDiskOptions, diskUsage, latestPartitions } from "@/lib/disk-view";
import type { MetricPoint } from "@/hooks/use-server-metrics";
import type { TimeRange } from "@/hooks/use-server-metrics";
import type { ValueType } from "recharts/types/component/DefaultTooltipContent";

interface MetricsChartsProps {
	data: MetricPoint[];
	range: TimeRange;
	isLoading: boolean;
	serverId: string;
}

const DISK_VIEW_KEY = "dockeeper.disk-view";

function readDiskView(serverId: string): string {
	if (!serverId) return "all";
	try {
		return localStorage.getItem(`${DISK_VIEW_KEY}.${serverId}`) ?? "all";
	} catch {
		return "all";
	}
}

function writeDiskView(serverId: string, view: string) {
	if (!serverId) return;
	try {
		localStorage.setItem(`${DISK_VIEW_KEY}.${serverId}`, view);
	} catch {
		return;
	}
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

function formatTemp(v: number): string {
	return `${v.toFixed(0)}°C`;
}

function tooltipPercent(v: ValueType | undefined): string {
	return formatPercent(Number(v ?? 0));
}

function tooltipBytes(v: ValueType | undefined): string {
	return formatBytes(Number(v ?? 0));
}

function tooltipTemp(v: ValueType | undefined): string {
	return formatTemp(Number(v ?? 0));
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

const tooltipStyle = {
	contentStyle: {
		backgroundColor: "var(--color-popover)",
		borderColor: "var(--color-border)",
		borderRadius: "8px",
		color: "var(--color-popover-foreground)",
	},
	itemStyle: { color: "var(--color-popover-foreground)" },
	labelStyle: { color: "var(--color-muted-foreground)" },
	wrapperStyle: { zIndex: 10 },
};

const chartColors = {
	cpu: "var(--color-primary)",
	mem: "var(--color-chart-2)",
	disk: "var(--color-chart-3)",
	netRx: "var(--color-chart-4)",
	netTx: "var(--color-chart-5)",
	diskRead: "var(--color-chart-4)",
	diskWrite: "var(--color-chart-5)",
	swap: "var(--color-chart-3)",
	swapTotal: "var(--color-muted-foreground)",
};

const TEMP_COLORS = [
	"var(--color-primary)",
	"var(--color-chart-2)",
	"var(--color-chart-3)",
	"var(--color-chart-4)",
	"var(--color-chart-5)",
	"var(--color-destructive)",
];

export function MetricsCharts({ data, range, isLoading, serverId }: MetricsChartsProps) {
	const [diskView, setDiskView] = useState(() => readDiskView(serverId));

	// Extract unique temperature sensor names and pick the most relevant ones.
	// Group by hardware prefix (coretemp → max across cores, etc.) to avoid clutter.
	const tempSensors = useMemo(() => {
		const names = new Set<string>();
		for (const p of data) {
			if (p.temperatures) {
				for (const t of p.temperatures) names.add(t.name);
			}
		}
		return [...names];
	}, [data]);

	// Aggregate sensors: keep one representative per hardware group.
	// e.g. coretemp_core_0..N → "CPU Cores (max)" using the hottest core per tick.
	const { aggregatedSensors, aggregatedTempData } = useMemo(() => {
		if (tempSensors.length === 0)
			return {
				aggregatedSensors: [] as string[],
				aggregatedTempData: [] as Record<string, unknown>[],
			};

		// Group sensors by prefix (everything before the last _N suffix).
		const groups = new Map<string, string[]>();
		for (const name of tempSensors) {
			const match = name.match(/^(.+?)_\d+$/);
			const prefix = match ? match[1] : name;
			const existing = groups.get(prefix) ?? [];
			existing.push(name);
			groups.set(prefix, existing);
		}

		// Build display names: groups with >1 sensor get "(max)" label.
		const displayMap = new Map<string, { label: string; members: string[] }>();
		for (const [prefix, members] of groups) {
			if (members.length > 1) {
				displayMap.set(prefix, { label: `${prefix} (max)`, members });
			} else {
				displayMap.set(prefix, { label: members[0], members });
			}
		}

		const sensors = [...displayMap.values()].map((v) => v.label);
		const tempRows = data.map((p) => {
			const row: Record<string, unknown> = { time: formatTime(p.timestamp, range) };
			const tempMap = new Map<string, number>();
			for (const t of p.temperatures ?? []) {
				tempMap.set(t.name, t.celsius);
			}
			for (const [, { label, members }] of displayMap) {
				if (members.length > 1) {
					const vals = members
						.map((m) => tempMap.get(m))
						.filter((v): v is number => v !== undefined);
					row[label] = vals.length > 0 ? Math.max(...vals) : undefined;
				} else {
					row[label] = tempMap.get(members[0]);
				}
			}
			return row;
		});

		return { aggregatedSensors: sensors, aggregatedTempData: tempRows };
	}, [data, tempSensors, range]);

	// Build latest partitions data for bar chart.
	const partitionData = useMemo(() => {
		const latest = latestPartitions(data);
		if (!latest.length) return [];
		return latest.map((p) => ({
			name: p.mountpoint,
			device: p.device,
			used: p.percent,
			total: p.total,
			usedBytes: p.used,
			available: p.available,
		}));
	}, [data]);

	const diskOptions = useMemo(() => buildDiskOptions(latestPartitions(data)), [data]);

	const activeDiskView = diskOptions.some((option) => option.value === diskView) ? diskView : "all";

	function handleDiskViewChange(value: string) {
		setDiskView(value);
		writeDiskView(serverId, value);
	}

	const latestDisk = useMemo(() => {
		for (let i = data.length - 1; i >= 0; i--) {
			const usage = diskUsage(data[i], activeDiskView);
			if (usage && usage.total > 0) return usage;
		}
		return null;
	}, [data, activeDiskView]);

	if (isLoading && data.length === 0) {
		return (
			<div className="grid gap-4 md:grid-cols-2">
				{Array.from({ length: 8 }).map((_, i) => (
					<ChartSkeleton key={i} />
				))}
			</div>
		);
	}

	const chartData = data.map((p) => {
		const usage = diskUsage(p, activeDiskView);
		return {
			...p,
			time: formatTime(p.timestamp, range),
			disk_used: usage?.used ?? null,
		};
	});

	// Check if we have data for the new charts.
	const hasDiskIO = data.some((p) => p.disk_io_read > 0 || p.disk_io_write > 0);
	const hasSwap = data.some((p) => p.swap_total > 0);
	const hasTemps = aggregatedSensors.length > 0;
	const hasPartitions = partitionData.length > 0;

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
							<Tooltip formatter={tooltipPercent} {...tooltipStyle} />
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
							<Tooltip formatter={tooltipPercent} {...tooltipStyle} />
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

			{/* Disk Usage */}
			<Card>
				<CardHeader className="pb-2">
					<div className="flex items-center justify-between gap-2">
						<CardTitle className="text-sm">Disk Usage</CardTitle>
						{diskOptions.length > 1 && (
							<div className="w-[170px]">
								<Select value={activeDiskView} onValueChange={handleDiskViewChange}>
									<SelectTrigger className="text-xs" aria-label="Disk view">
										<SelectValue />
									</SelectTrigger>
									<SelectContent>
										{diskOptions.map((option) => (
											<SelectItem key={option.value} value={option.value}>
												{option.label}
											</SelectItem>
										))}
									</SelectContent>
								</Select>
							</div>
						)}
					</div>
				</CardHeader>
				<CardContent>
					{latestDisk && (
						<p className="mb-2 text-xs text-muted-foreground">
							{formatBytes(latestDisk.used)} / {formatBytes(latestDisk.total)} ·{" "}
							{Math.round((latestDisk.used / latestDisk.total) * 100)}%
						</p>
					)}
					<ResponsiveContainer width="100%" height={200}>
						<AreaChart data={chartData}>
							<CartesianGrid strokeDasharray="3 3" className="stroke-border" />
							<XAxis dataKey="time" tick={{ fontSize: 11 }} className="text-muted-foreground" />
							<YAxis
								tick={{ fontSize: 11 }}
								tickFormatter={formatBytes}
								className="text-muted-foreground"
							/>
							<Tooltip formatter={tooltipBytes} {...tooltipStyle} />
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
								connectNulls
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
							<Tooltip formatter={tooltipBytes} {...tooltipStyle} />
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

			{/* Disk I/O */}
			{hasDiskIO && (
				<Card>
					<CardHeader className="pb-2">
						<CardTitle className="text-sm">Disk I/O</CardTitle>
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
								<Tooltip formatter={tooltipBytes} {...tooltipStyle} />
								<Legend />
								<defs>
									<linearGradient id="fillDiskRead" x1="0" y1="0" x2="0" y2="1">
										<stop offset="0%" stopColor={chartColors.diskRead} stopOpacity={0.3} />
										<stop offset="100%" stopColor={chartColors.diskRead} stopOpacity={0} />
									</linearGradient>
									<linearGradient id="fillDiskWrite" x1="0" y1="0" x2="0" y2="1">
										<stop offset="0%" stopColor={chartColors.diskWrite} stopOpacity={0.3} />
										<stop offset="100%" stopColor={chartColors.diskWrite} stopOpacity={0} />
									</linearGradient>
								</defs>
								<Area
									type="monotone"
									dataKey="disk_io_read"
									stroke={chartColors.diskRead}
									fill="url(#fillDiskRead)"
									strokeWidth={1.5}
									isAnimationActive={false}
									name="Read"
								/>
								<Area
									type="monotone"
									dataKey="disk_io_write"
									stroke={chartColors.diskWrite}
									fill="url(#fillDiskWrite)"
									strokeWidth={1.5}
									isAnimationActive={false}
									name="Write"
								/>
							</AreaChart>
						</ResponsiveContainer>
					</CardContent>
				</Card>
			)}

			{/* Swap */}
			{hasSwap && (
				<Card>
					<CardHeader className="pb-2">
						<CardTitle className="text-sm">Swap Usage</CardTitle>
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
								<Tooltip formatter={tooltipBytes} {...tooltipStyle} />
								<Legend />
								<defs>
									<linearGradient id="fillSwap" x1="0" y1="0" x2="0" y2="1">
										<stop offset="0%" stopColor={chartColors.swap} stopOpacity={0.3} />
										<stop offset="100%" stopColor={chartColors.swap} stopOpacity={0} />
									</linearGradient>
								</defs>
								<Area
									type="monotone"
									dataKey="swap_used"
									stroke={chartColors.swap}
									fill="url(#fillSwap)"
									strokeWidth={1.5}
									isAnimationActive={false}
									name="Used"
								/>
								<Area
									type="monotone"
									dataKey="swap_total"
									stroke={chartColors.swapTotal}
									fill="none"
									strokeWidth={1}
									strokeDasharray="4 4"
									isAnimationActive={false}
									name="Total"
								/>
							</AreaChart>
						</ResponsiveContainer>
					</CardContent>
				</Card>
			)}

			{/* Temperature */}
			{hasTemps && (
				<Card>
					<CardHeader className="pb-2">
						<CardTitle className="text-sm">Temperature</CardTitle>
					</CardHeader>
					<CardContent>
						<ResponsiveContainer width="100%" height={250}>
							<LineChart data={aggregatedTempData}>
								<CartesianGrid strokeDasharray="3 3" className="stroke-border" />
								<XAxis dataKey="time" tick={{ fontSize: 11 }} className="text-muted-foreground" />
								<YAxis
									tick={{ fontSize: 11 }}
									tickFormatter={formatTemp}
									className="text-muted-foreground"
									domain={["auto", "auto"]}
								/>
								<Tooltip formatter={tooltipTemp} {...tooltipStyle} wrapperStyle={{ zIndex: 10 }} />
								<Legend wrapperStyle={{ fontSize: 11, paddingTop: 8 }} />
								{aggregatedSensors.map((name, i) => (
									<Line
										key={name}
										type="monotone"
										dataKey={name}
										stroke={TEMP_COLORS[i % TEMP_COLORS.length]}
										strokeWidth={1.5}
										dot={false}
										isAnimationActive={false}
										name={name}
									/>
								))}
							</LineChart>
						</ResponsiveContainer>
					</CardContent>
				</Card>
			)}

			{/* Partitions */}
			{hasPartitions && (
				<Card>
					<CardHeader className="pb-2">
						<CardTitle className="text-sm">Disk Partitions</CardTitle>
					</CardHeader>
					<CardContent>
						<ResponsiveContainer width="100%" height={200}>
							<BarChart data={partitionData} layout="vertical">
								<CartesianGrid strokeDasharray="3 3" className="stroke-border" />
								<XAxis
									type="number"
									domain={[0, 100]}
									tick={{ fontSize: 11 }}
									tickFormatter={formatPercent}
									className="text-muted-foreground"
								/>
								<YAxis
									type="category"
									dataKey="name"
									tick={{ fontSize: 10 }}
									width={80}
									className="text-muted-foreground"
								/>
								<Tooltip
									cursor={{ fill: "var(--color-muted)", opacity: 0.15 }}
									formatter={(v, _name, entry) => {
										const p = entry?.payload;
										if (p) {
											return `${formatPercent(Number(v ?? 0))} (${formatBytes(p.usedBytes)} / ${formatBytes(p.total)})`;
										}
										return formatPercent(Number(v ?? 0));
									}}
									labelFormatter={(label) => {
										const p = partitionData.find((d) => d.name === String(label));
										return p ? `${String(label)} (${p.device})` : String(label);
									}}
									{...tooltipStyle}
								/>
								<Bar
									dataKey="used"
									fill={chartColors.disk}
									radius={[0, 4, 4, 0]}
									isAnimationActive={false}
									name="Used %"
								/>
							</BarChart>
						</ResponsiveContainer>
					</CardContent>
				</Card>
			)}
		</div>
	);
}
