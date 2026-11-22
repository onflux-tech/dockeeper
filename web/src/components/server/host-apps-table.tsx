import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { HostActionsMenu } from "@/components/server/host-actions-menu";
import { formatCpuMem } from "@/lib/format";
import type { HostAppRecord } from "@/types/pocketbase";

interface HostAppsTableProps {
	apps: HostAppRecord[];
	isLoading: boolean;
	serverID?: string;
	serverOnline?: boolean;
}

const statusVariants: Record<HostAppRecord["status"], "default" | "secondary" | "destructive"> = {
	running: "default",
	stopped: "secondary",
	failed: "destructive",
	unknown: "secondary",
};

export function HostAppsTable({ apps, isLoading, serverID, serverOnline }: HostAppsTableProps) {
	if (isLoading) {
		return (
			<div className="rounded-md border">
				<Table>
					<TableHeader>
						<TableRow>
							{Array.from({ length: 7 }).map((_, index) => (
								<TableHead key={index}>
									<Skeleton className="h-4 w-16" />
								</TableHead>
							))}
						</TableRow>
					</TableHeader>
					<TableBody>
						{Array.from({ length: 4 }).map((_, row) => (
							<TableRow key={row}>
								{Array.from({ length: 7 }).map((_, col) => (
									<TableCell key={col}>
										<Skeleton className="h-4 w-full" />
									</TableCell>
								))}
							</TableRow>
						))}
					</TableBody>
				</Table>
			</div>
		);
	}

	return (
		<div className="rounded-md border">
			<Table>
				<TableHeader>
					<TableRow>
						<TableHead>App</TableHead>
						<TableHead className="w-24">Source</TableHead>
						<TableHead className="w-24">Status</TableHead>
						<TableHead className="w-32">Ports</TableHead>
						<TableHead className="w-32">CPU / Mem</TableHead>
						<TableHead className="w-20">Health</TableHead>
						<TableHead className="w-10" />
					</TableRow>
				</TableHeader>
				<TableBody>
					{apps.length === 0 ? (
						<TableRow>
							<TableCell colSpan={7} className="py-8 text-center text-muted-foreground">
								No applications reported.
							</TableCell>
						</TableRow>
					) : (
						apps.map((app) => (
							<TableRow key={app.id}>
								<TableCell className="text-sm font-medium">{app.name}</TableCell>
								<TableCell>
									<Badge variant="outline">{app.source}</Badge>
								</TableCell>
								<TableCell>
									<Badge variant={statusVariants[app.status] ?? "secondary"}>{app.status}</Badge>
								</TableCell>
								<TableCell className="text-xs tabular-nums text-muted-foreground">
									{app.ports?.length ? app.ports.join(", ") : "—"}
								</TableCell>
								<TableCell className="text-xs tabular-nums text-muted-foreground">
									{formatCpuMem(app.cpu_percent, app.mem_bytes)}
								</TableCell>
								<TableCell className="text-xs text-muted-foreground">—</TableCell>
								<TableCell>
									{serverID ? (
										<HostActionsMenu serverID={serverID} app={app} serverOnline={serverOnline} />
									) : null}
								</TableCell>
							</TableRow>
						))
					)}
				</TableBody>
			</Table>
		</div>
	);
}
