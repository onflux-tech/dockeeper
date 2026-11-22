import type { ReactNode } from "react";
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
	limit?: number;
	title?: string;
	subtitle?: string;
	action?: ReactNode;
}

const statusVariants: Record<HostAppRecord["status"], "default" | "secondary" | "destructive"> = {
	running: "default",
	stopped: "secondary",
	failed: "destructive",
	unknown: "secondary",
};

export function HostAppsTable({
	apps,
	isLoading,
	serverID,
	serverOnline,
	limit,
	title,
	subtitle,
	action,
}: HostAppsTableProps) {
	const visibleApps = limit ? apps.slice(0, limit) : apps;

	if (isLoading) {
		return (
			<div className="rounded-md border">
				<div className="px-6">
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
			</div>
		);
	}

	return (
		<div className="rounded-md border">
			{title ? (
				<div className="flex items-start justify-between gap-3 p-6">
					<div>
						<h2 className="text-base font-semibold">{title}</h2>
						{subtitle ? <p className="text-sm text-muted-foreground">{subtitle}</p> : null}
					</div>
					{action}
				</div>
			) : null}
			<div className="px-6">
				<Table>
					<TableHeader>
						<TableRow>
							<TableHead className="text-sm font-medium text-foreground uppercase">App</TableHead>
							<TableHead className="w-24 text-sm font-medium text-foreground uppercase">
								Source
							</TableHead>
							<TableHead className="w-24 text-sm font-medium text-foreground uppercase">
								Status
							</TableHead>
							<TableHead className="w-32 text-sm font-medium text-foreground uppercase">
								Ports
							</TableHead>
							<TableHead className="w-32 text-sm font-medium text-foreground uppercase">
								CPU / Mem
							</TableHead>
							<TableHead className="w-20 text-sm font-medium text-foreground uppercase">
								Health
							</TableHead>
							<TableHead className="w-10" />
						</TableRow>
					</TableHeader>
					<TableBody>
						{visibleApps.length === 0 ? (
							<TableRow>
								<TableCell colSpan={7} className="py-8 text-center text-muted-foreground">
									No applications reported.
								</TableCell>
							</TableRow>
						) : (
							visibleApps.map((app) => (
								<TableRow key={app.id}>
									<TableCell>
										<div className="flex flex-col">
											<span className="text-sm font-medium">{app.name}</span>
											<span className="text-xs text-muted-foreground">
												{app.source}
												{app.exec_path ? ` · ${app.exec_path}` : ""}
											</span>
										</div>
									</TableCell>
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
			{limit && apps.length > limit ? (
				<div className="border-t px-6 py-3 text-xs text-muted-foreground">
					Showing {visibleApps.length} of {apps.length}
				</div>
			) : null}
		</div>
	);
}
