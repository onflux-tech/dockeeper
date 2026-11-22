import { Skeleton } from "@/components/ui/skeleton";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { formatBytes } from "@/lib/format";
import type { HostProcessRecord } from "@/types/pocketbase";

interface HostProcessesTableProps {
	processes: HostProcessRecord[];
	isLoading: boolean;
	limit?: number;
}

export function HostProcessesTable({ processes, isLoading, limit }: HostProcessesTableProps) {
	const visibleProcesses = limit ? processes.slice(0, limit) : processes;

	if (isLoading) {
		return (
			<div className="rounded-md border">
				<Table>
					<TableHeader>
						<TableRow>
							{Array.from({ length: 6 }).map((_, index) => (
								<TableHead key={index}>
									<Skeleton className="h-4 w-16" />
								</TableHead>
							))}
						</TableRow>
					</TableHeader>
					<TableBody>
						{Array.from({ length: 4 }).map((_, row) => (
							<TableRow key={row}>
								{Array.from({ length: 6 }).map((_, col) => (
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
						<TableHead className="w-24">PID</TableHead>
						<TableHead>Process</TableHead>
						<TableHead className="w-48">App</TableHead>
						<TableHead className="w-24">CPU</TableHead>
						<TableHead className="w-28">Mem</TableHead>
						<TableHead className="w-24">Threads</TableHead>
					</TableRow>
				</TableHeader>
				<TableBody>
					{visibleProcesses.length === 0 ? (
						<TableRow>
							<TableCell colSpan={6} className="py-8 text-center text-muted-foreground">
								No processes reported.
							</TableCell>
						</TableRow>
					) : (
						visibleProcesses.map((process) => (
							<TableRow key={process.id}>
								<TableCell className="text-xs tabular-nums text-muted-foreground">
									{process.pid}
								</TableCell>
								<TableCell className="text-sm font-medium">{process.name}</TableCell>
								<TableCell className="text-sm text-muted-foreground">
									{process.app || "—"}
								</TableCell>
								<TableCell className="text-xs tabular-nums text-muted-foreground">
									{process.cpu_percent > 0 ? `${process.cpu_percent.toFixed(1)}%` : "—"}
								</TableCell>
								<TableCell className="text-xs tabular-nums text-muted-foreground">
									{process.mem_bytes > 0 ? formatBytes(process.mem_bytes) : "—"}
								</TableCell>
								<TableCell className="text-xs tabular-nums text-muted-foreground">
									{process.threads > 0 ? process.threads : "—"}
								</TableCell>
							</TableRow>
						))
					)}
				</TableBody>
			</Table>
			{limit && processes.length > limit ? (
				<div className="border-t px-3 py-2 text-xs text-muted-foreground">
					Showing {visibleProcesses.length} of {processes.length}
				</div>
			) : null}
		</div>
	);
}
