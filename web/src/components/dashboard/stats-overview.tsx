import { Server, Activity, AlertTriangle, Container } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import type { ServerRecord } from "@/types/pocketbase";

interface StatsOverviewProps {
	servers: ServerRecord[];
	totalContainers: number;
	isLoading: boolean;
}

interface StatCardProps {
	label: string;
	value: number;
	icon: React.ReactNode;
}

function StatCard({ label, value, icon }: StatCardProps) {
	return (
		<Card>
			<CardContent className="flex items-center gap-4 py-0">
				<div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-muted">
					{icon}
				</div>
				<div className="flex flex-col">
					<span className="text-2xl font-bold tabular-nums">{value}</span>
					<span className="text-xs text-muted-foreground">{label}</span>
				</div>
			</CardContent>
		</Card>
	);
}

function StatCardSkeleton() {
	return (
		<Card>
			<CardContent className="flex items-center gap-4 py-0">
				<Skeleton className="h-10 w-10 rounded-lg" />
				<div className="flex flex-col gap-1">
					<Skeleton className="h-7 w-12" />
					<Skeleton className="h-3 w-20" />
				</div>
			</CardContent>
		</Card>
	);
}

export function StatsOverview({ servers, totalContainers, isLoading }: StatsOverviewProps) {
	if (isLoading) {
		return (
			<div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
				{Array.from({ length: 4 }).map((_, i) => (
					<StatCardSkeleton key={i} />
				))}
			</div>
		);
	}

	const online = servers.filter((s) => s.status === "online").length;
	const offline = servers.filter((s) => s.status !== "online").length;

	return (
		<div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
			<StatCard
				label="Total Servers"
				value={servers.length}
				icon={<Server className="h-5 w-5 text-muted-foreground" />}
			/>
			<StatCard
				label="Online"
				value={online}
				icon={<Activity className="h-5 w-5 text-success" />}
			/>
			<StatCard
				label="Offline / Error"
				value={offline}
				icon={<AlertTriangle className="h-5 w-5 text-destructive" />}
			/>
			<StatCard
				label="Containers"
				value={totalContainers}
				icon={<Container className="h-5 w-5 text-muted-foreground" />}
			/>
		</div>
	);
}
