import { useState, useCallback, useMemo, useRef } from "react";
import {
	ResponsiveGridLayout,
	useContainerWidth,
	verticalCompactor,
} from "react-grid-layout";
import type { Layout, LayoutItem } from "react-grid-layout";
import {
	HeartPulse,
	Bell,
	ServerCog,
	GripVertical,
	Lock,
	Unlock,
	RotateCcw,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import { useCollection } from "@/hooks/use-collection";
import { useRealtime } from "@/hooks/use-realtime";
import { useHealthChecks } from "@/hooks/use-health-checks";
import {
	useDashboardLayout,
	useSaveDashboardLayout,
	useResetDashboardLayout,
} from "@/hooks/use-dashboard";
import { StatsOverview } from "@/components/dashboard/stats-overview";
import { ServerCard, ServerCardSkeleton } from "@/components/dashboard/server-card";
import type { SparklinePoint } from "@/components/dashboard/sparkline-chart";
import { useTags } from "@/hooks/use-tags";
import type { ServerMetricsRawRecord, ServerRecord } from "@/types/pocketbase";

import "react-grid-layout/css/styles.css";
import "./dashboard.css";

// ── Constants ────────────────────────────────────────────

const LS_SERVER_ORDER_KEY = "dk-server-card-order";

// Widget defs for the bottom RGL grid
const WIDGET_DEFS = [
	{ id: "health-status", label: "Health Status", minW: 2, minH: 3 },
	{ id: "recent-events", label: "Recent Events", minW: 2, minH: 3 },
	{ id: "alerts-summary", label: "Alerts Summary", minW: 2, minH: 3 },
] as const;

// Column counts per breakpoint (must match the RGL cols prop).
const COLS: Record<string, number> = { lg: 12, md: 9, sm: 6, xs: 4 };

const widgetMap = new Map<string, (typeof WIDGET_DEFS)[number]>(WIDGET_DEFS.map((d) => [d.id, d]));

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

// ── Server card order persistence (localStorage) ─────────

function getSavedServerOrder(): string[] {
	try {
		const raw = localStorage.getItem(LS_SERVER_ORDER_KEY);
		return raw ? JSON.parse(raw) : [];
	} catch {
		return [];
	}
}

function saveServerOrder(order: string[]) {
	localStorage.setItem(LS_SERVER_ORDER_KEY, JSON.stringify(order));
}

function orderServers(servers: ServerRecord[]): ServerRecord[] {
	const saved = getSavedServerOrder();
	if (saved.length === 0) return servers;
	const map = new Map(servers.map((s) => [s.id, s]));
	const ordered: ServerRecord[] = [];
	for (const id of saved) {
		const s = map.get(id);
		if (s) {
			ordered.push(s);
			map.delete(id);
		}
	}
	for (const s of map.values()) {
		ordered.push(s);
	}
	return ordered;
}

// ── Main page ─────────────────────────────────────────────

export default function DashboardPage() {
	const [fiveMinAgo] = useState(() => {
		const d = new Date(Date.now() - 5 * 60 * 1000);
		return d.toISOString().replace("T", " ");
	});

	const { width, containerRef, mounted } = useContainerWidth({
		measureBeforeMount: true,
	});

	const { data: layoutData, isLoading: loadingLayout } = useDashboardLayout();
	const saveMut = useSaveDashboardLayout();
	const resetMut = useResetDashboardLayout();
	const [editing, setEditing] = useState(false);

	const { data: serversResult, isLoading: loadingServers } = useCollection("servers", {
		sort: "name",
		perPage: 50,
	});
	const { data: containersResult, isLoading: loadingContainers } = useCollection("containers");
	const { data: metricsResult } = useCollection("server_metrics_raw", {
		filter: `timestamp >= "${fiveMinAgo}"`,
		sort: "timestamp",
		perPage: 500,
	});
	const { data: healthChecks } = useHealthChecks();

	useRealtime("servers");
	useRealtime("server_metrics_raw");

	const { data: tagsData } = useTags("server");
	const allTags = tagsData?.items ?? [];
	const tagsByServer = new Map<string, { id: string; name: string; color: string }[]>();
	for (const tag of allTags) {
		const existing = tagsByServer.get(tag.resource_id) ?? [];
		existing.push({ id: tag.id, name: tag.name, color: tag.color });
		tagsByServer.set(tag.resource_id, existing);
	}

	const servers = serversResult?.items ?? [];
	const containers = containersResult?.items ?? [];
	const metrics = metricsResult?.items ?? [];
	const isLoading = loadingServers || loadingContainers;

	// Ordered servers for the card grid
	const [serverOrderVersion, setServerOrderVersion] = useState(0);
	const orderedServers = useMemo(
		() => orderServers(servers),
		// eslint-disable-next-line react-hooks/exhaustive-deps
		[servers, serverOrderVersion],
	);

	// Build layout items for bottom widgets (filter out server-cards and stats-overview)
	// Generate a layout per breakpoint so 3 widgets always fit in one row.
	const responsiveLayouts = useMemo(() => {
		if (!layoutData?.widgets) return {};
		const filtered = layoutData.widgets.filter(
			(w) => w.id !== "stats-overview" && w.id !== "server-cards",
		);

		const result: Record<string, LayoutItem[]> = {};
		for (const [bp, cols] of Object.entries(COLS)) {
			const perWidget = Math.floor(cols / Math.max(filtered.length, 1));
			result[bp] = filtered.map((w, idx) => {
				const def = widgetMap.get(w.id);
				const minH = def?.minH ?? 3;
				const minW = def?.minW ?? 2;
				const widgetW = bp === "lg" ? Math.max(w.w, minW) : Math.max(perWidget, minW);
				return {
					i: w.id,
					x: bp === "lg" ? w.x : idx * perWidget,
					y: bp === "lg" ? w.y : 0,
					w: widgetW,
					h: Math.max(w.h, minH),
					minW,
					minH,
					maxW: cols,
					static: !editing,
				};
			});
		}
		return result;
	}, [layoutData, editing]);

	const layoutItems = responsiveLayouts.lg ?? [];

	const handleLayoutChange = useCallback(
		(newLayout: Layout) => {
			if (!editing) return;
			const widgets = newLayout.map((item) => ({
				id: item.i,
				x: item.x,
				y: item.y,
				w: item.w,
				h: item.h,
			}));
			saveMut.mutate({ widgets });
		},
		[editing, saveMut],
	);

	const handleReset = () => {
		resetMut.mutate(undefined, {
			onSuccess: () => setEditing(false),
		});
	};

	return (
		<div className="flex flex-col gap-4" ref={containerRef}>
			{loadingLayout ? (
				<>
					<Skeleton className="h-8 w-48" />
					<div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
						{[1, 2, 3, 4].map((i) => (
							<Skeleton key={i} className="h-24" />
						))}
					</div>
				</>
			) : (
				<>
					<div className="flex items-center justify-between">
						<h1 className="text-2xl font-semibold">Dashboard</h1>
						<div className="flex items-center gap-2">
							{editing && (
								<Button variant="outline" size="sm" onClick={handleReset}>
									<RotateCcw className="mr-1.5 h-3.5 w-3.5" />
									Reset
								</Button>
							)}
							<Button
								variant={editing ? "default" : "outline"}
								size="sm"
								onClick={() => setEditing((v) => !v)}
							>
								{editing ? (
									<>
										<Lock className="mr-1.5 h-3.5 w-3.5" />
										Lock
									</>
								) : (
									<>
										<Unlock className="mr-1.5 h-3.5 w-3.5" />
										Customize
									</>
								)}
							</Button>
						</div>
					</div>

					{/* Fixed stats — flat cards, no wrapper */}
					<StatsOverview
						servers={servers}
						containers={containers as never}
						isLoading={isLoading}
					/>

					{/* Server cards — borderless, 3-col grid, scroll, drag to reorder */}
					<ServerCardGrid
						servers={orderedServers}
						metrics={metrics}
						isLoading={isLoading}
						editing={editing}
						onOrderChange={() => setServerOrderVersion((v) => v + 1)}
						tagsByServer={tagsByServer}
					/>

					{/* Bottom widgets — RGL grid with resize */}
					{mounted && layoutItems.length > 0 && (
						<ResponsiveGridLayout
							className={editing ? "layout dashboard-editing" : "layout"}
							width={width}
							layouts={responsiveLayouts}
							breakpoints={{ lg: 1200, md: 996, sm: 768, xs: 480 }}
							cols={COLS}
							rowHeight={40}
							margin={[16, 16] as const}
							containerPadding={[0, 0] as const}
							dragConfig={{
								handle: ".widget-drag-handle",
								enabled: editing,
								bounded: true,
							}}
							resizeConfig={{
								enabled: editing,
								handles: ["se", "s", "e"] as const,
							}}
							compactor={verticalCompactor}
							onLayoutChange={handleLayoutChange}
						>
							{layoutItems.map((item) => (
								<div key={item.i}>
									<WidgetWrapper id={item.i} editing={editing}>
										<WidgetContent
											id={item.i}
											healthChecks={healthChecks ?? []}
										/>
									</WidgetWrapper>
								</div>
							))}
						</ResponsiveGridLayout>
					)}
				</>
			)}
		</div>
	);
}

// ── Server Card Grid (borderless, drag to reorder) ───────

function ServerCardGrid({
	servers,
	metrics,
	isLoading,
	editing,
	onOrderChange,
	tagsByServer,
}: {
	servers: ServerRecord[];
	metrics: ServerMetricsRawRecord[];
	isLoading: boolean;
	editing: boolean;
	onOrderChange: () => void;
	tagsByServer: Map<string, { id: string; name: string; color: string }[]>;
}) {
	const dragItem = useRef<string | null>(null);
	const dragOverItem = useRef<string | null>(null);
	const [draggingId, setDraggingId] = useState<string | null>(null);

	const handleDragStart = (e: React.DragEvent, id: string) => {
		dragItem.current = id;
		setDraggingId(id);
		e.dataTransfer.effectAllowed = "move";
	};

	const handleDragOver = (e: React.DragEvent, id: string) => {
		e.preventDefault();
		e.dataTransfer.dropEffect = "move";
		dragOverItem.current = id;
	};

	const handleDrop = () => {
		if (!dragItem.current || !dragOverItem.current) return;
		if (dragItem.current === dragOverItem.current) {
			setDraggingId(null);
			return;
		}

		const currentOrder = servers.map((s) => s.id);
		const fromIdx = currentOrder.indexOf(dragItem.current);
		const toIdx = currentOrder.indexOf(dragOverItem.current);
		if (fromIdx === -1 || toIdx === -1) return;

		const newOrder = [...currentOrder];
		const [moved] = newOrder.splice(fromIdx, 1);
		newOrder.splice(toIdx, 0, moved);
		saveServerOrder(newOrder);
		onOrderChange();

		dragItem.current = null;
		dragOverItem.current = null;
		setDraggingId(null);
	};

	const handleDragEnd = () => {
		setDraggingId(null);
	};

	const needsScroll = servers.length > 6;
	const gridRef = useRef<HTMLDivElement>(null);

	if (isLoading) {
		return (
			<div className="grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-3">
				{[1, 2, 3].map((i) => (
					<ServerCardSkeleton key={i} />
				))}
			</div>
		);
	}

	if (servers.length === 0) {
		return (
			<div className="flex flex-col items-center justify-center gap-2 rounded-lg border border-dashed py-12 text-center">
				<ServerCog className="h-8 w-8 text-muted-foreground/50" />
				<p className="text-sm text-muted-foreground">No servers registered yet.</p>
			</div>
		);
	}

	return (
		<div
			ref={gridRef}
			className={needsScroll ? "server-card-scroll overflow-y-auto overflow-x-hidden" : undefined}
		>
			<div className="grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-3">
				{servers.map((server) => (
					<div
						key={server.id}
						draggable={editing}
						onDragStart={(e) => handleDragStart(e, server.id)}
						onDragOver={(e) => handleDragOver(e, server.id)}
						onDrop={handleDrop}
						onDragEnd={handleDragEnd}
						className={draggingId === server.id ? "opacity-50" : undefined}
					>
						<ServerCard
							server={server}
							cpuData={buildSparkline(metrics, server.id, "cpu_percent")}
							memData={buildSparkline(metrics, server.id, "mem_percent")}
							editing={editing}
							tags={tagsByServer.get(server.id)}
						/>
					</div>
				))}
			</div>
		</div>
	);
}

// ── Widget Wrapper (for RGL grid items) ──────────────────

function WidgetWrapper({
	id,
	editing,
	children,
}: {
	id: string;
	editing: boolean;
	children: React.ReactNode;
}) {
	const def = widgetMap.get(id);
	return (
		<Card className="flex h-full flex-col overflow-hidden">
			{editing && (
				<div className="widget-drag-handle flex items-center gap-1 border-b bg-muted/50 px-3 py-1.5 cursor-grab active:cursor-grabbing">
					<GripVertical className="h-3.5 w-3.5 text-muted-foreground" />
					<span className="text-xs font-medium text-muted-foreground">
						{def?.label ?? id}
					</span>
				</div>
			)}
			<div className="min-h-0 flex-1 overflow-hidden p-3">{children}</div>
		</Card>
	);
}

// ── Widget Content ────────────────────────────────────────

function WidgetContent({
	id,
	healthChecks,
}: {
	id: string;
	healthChecks: { status: string; name: string; last_check_at: string }[];
}) {
	switch (id) {
		case "health-status": {
			const healthy = healthChecks.filter((c) => c.status === "healthy").length;
			const unhealthy = healthChecks.filter((c) => c.status === "unhealthy").length;
			return (
				<div className="flex flex-col gap-3">
					<div className="flex items-center gap-2">
						<HeartPulse className="h-4 w-4 text-muted-foreground" />
						<span className="text-sm font-medium">Health Checks</span>
					</div>
					<div className="flex gap-4">
						<div className="flex items-center gap-1.5">
							<div className="h-2.5 w-2.5 rounded-full bg-emerald-500" />
							<span className="text-sm">{healthy} healthy</span>
						</div>
						<div className="flex items-center gap-1.5">
							<div className="h-2.5 w-2.5 rounded-full bg-destructive" />
							<span className="text-sm">{unhealthy} unhealthy</span>
						</div>
					</div>
					{healthChecks
						.filter((c) => c.status === "unhealthy")
						.slice(0, 3)
						.map((c, i) => (
							<div key={i} className="flex items-center justify-between text-xs">
								<span className="truncate">{c.name}</span>
								<Badge variant="destructive" className="text-[10px]">
									Unhealthy
								</Badge>
							</div>
						))}
				</div>
			);
		}

		case "recent-events":
			return (
				<div className="flex flex-col gap-3">
					<div className="flex items-center gap-2">
						<Bell className="h-4 w-4 text-muted-foreground" />
						<span className="text-sm font-medium">Recent Events</span>
					</div>
					<p className="text-xs text-muted-foreground">
						View events on the Events page.
					</p>
				</div>
			);

		case "alerts-summary":
			return (
				<div className="flex flex-col gap-3">
					<div className="flex items-center gap-2">
						<Bell className="h-4 w-4 text-muted-foreground" />
						<span className="text-sm font-medium">Alerts</span>
					</div>
					<p className="text-xs text-muted-foreground">
						View active alerts on the Alerts page.
					</p>
				</div>
			);

		default:
			return <p className="text-sm text-muted-foreground">Unknown widget: {id}</p>;
	}
}
