import { Link } from "react-router";
import { Cpu, MemoryStick, ChevronRight, GripVertical } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { TagBadge } from "@/components/tag-badge";
import { SparklineChart, type SparklinePoint } from "@/components/dashboard/sparkline-chart";
import { relativeTime } from "@/lib/format";
import type { ServerRecord } from "@/types/pocketbase";

interface ServerCardProps {
	server: ServerRecord;
	cpuData: SparklinePoint[];
	memData: SparklinePoint[];
	editing?: boolean;
	tags?: { id: string; name: string; color: string }[];
}

const statusStyles: Record<
	string,
	{ variant: "default" | "secondary" | "destructive"; label: string }
> = {
	online: { variant: "default", label: "Online" },
	offline: { variant: "secondary", label: "Offline" },
	error: { variant: "destructive", label: "Error" },
};

export function ServerCard({ server, cpuData, memData, editing, tags }: ServerCardProps) {
	const style = statusStyles[server.status] ?? statusStyles.offline;

	const content = (
		<Card className="overflow-visible transition-colors hover:border-primary/40">
			{editing && (
				<div className="server-drag-handle flex items-center gap-1 border-b bg-muted/50 px-3 py-1 cursor-grab active:cursor-grabbing">
					<GripVertical className="h-3 w-3 text-muted-foreground" />
					<span className="text-[10px] font-medium text-muted-foreground">Drag to reorder</span>
				</div>
			)}
			<CardHeader className="pb-0">
				<div className="flex items-start justify-between">
					<div className="flex flex-col gap-1">
						<CardTitle className="text-sm">{server.name || server.hostname}</CardTitle>
						<span className="text-xs text-muted-foreground">
							{server.os}/{server.arch} &middot; v{server.agent_version}
						</span>
					</div>
					<div className="flex items-center gap-1">
						<Badge variant={style.variant}>{style.label}</Badge>
						<ChevronRight className="h-4 w-4 text-muted-foreground" />
					</div>
				</div>
			</CardHeader>

			<CardContent className="flex flex-col gap-3">
				<div className="flex items-center justify-between gap-4">
					<div className="flex flex-1 flex-col gap-1">
						<div className="flex items-center gap-1 text-xs text-muted-foreground">
							<Cpu className="h-3 w-3" />
							<span>CPU</span>
						</div>
						<SparklineChart data={cpuData} color="var(--color-primary)" height={36} />
					</div>
					<div className="flex flex-1 flex-col gap-1">
						<div className="flex items-center gap-1 text-xs text-muted-foreground">
							<MemoryStick className="h-3 w-3" />
							<span>RAM</span>
						</div>
						<SparklineChart data={memData} color="var(--color-chart-2)" height={36} />
					</div>
				</div>

				<div className="flex items-center justify-between">
					<span className="text-[11px] text-muted-foreground">
						Last seen {relativeTime(server.last_seen)}
					</span>
					{tags && tags.length > 0 && (
						<div className="flex flex-wrap justify-end gap-1">
							{tags.map((tag) => (
								<TagBadge key={tag.id} name={tag.name} color={tag.color} />
							))}
						</div>
					)}
				</div>
			</CardContent>
		</Card>
	);

	if (editing) {
		return content;
	}

	return (
		<Link to={`/servers/${server.id}`} className="block">
			{content}
		</Link>
	);
}

export function ServerCardSkeleton() {
	return (
		<Card>
			<CardHeader className="pb-0">
				<div className="flex items-start justify-between">
					<div className="flex flex-col gap-1">
						<Skeleton className="h-4 w-28" />
						<Skeleton className="h-3 w-36" />
					</div>
					<Skeleton className="h-5 w-14 rounded-full" />
				</div>
			</CardHeader>
			<CardContent className="flex flex-col gap-3">
				<div className="flex gap-4">
					<div className="flex flex-1 flex-col gap-1">
						<Skeleton className="h-3 w-10" />
						<Skeleton className="h-7 w-full" />
					</div>
					<div className="flex flex-1 flex-col gap-1">
						<Skeleton className="h-3 w-10" />
						<Skeleton className="h-7 w-full" />
					</div>
				</div>
				<Skeleton className="h-3 w-24" />
			</CardContent>
		</Card>
	);
}
