import { useMemo, useState } from "react";
import { HardDrive, Plus, Trash2, Search } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { CascadeDeleteDialog, type VolumeCascadeOptions } from "@/components/cascade-delete-dialog";
import { useCollection } from "@/hooks/use-collection";
import { useRealtime } from "@/hooks/use-realtime";
import { useRemoveVolume, useInspectVolume } from "@/hooks/use-volumes";
import { CreateVolumeDialog } from "@/components/volumes/create-volume-dialog";
import { PaginationControls } from "@/components/ui/pagination-controls";
import { andFilters, containsAnyFilter, eqFilter } from "@/lib/pocketbase-filters";
import type { VolumeRecord, ServerRecord } from "@/types/pocketbase";

function formatSize(bytes: number): string {
	if (!bytes || bytes <= 0) return "—";
	const units = ["B", "KB", "MB", "GB", "TB"];
	let idx = 0;
	let size = bytes;
	while (size >= 1024 && idx < units.length - 1) {
		size /= 1024;
		idx++;
	}
	return `${size.toFixed(idx === 0 ? 0 : 1)} ${units[idx]}`;
}

export default function VolumesPage() {
	const [serverFilter, setServerFilter] = useState<string>("all");
	const [search, setSearch] = useState("");
	const [createOpen, setCreateOpen] = useState(false);
	const [removeTarget, setRemoveTarget] = useState<VolumeRecord | null>(null);
	const [inspectData, setInspectData] = useState<Record<string, unknown> | null>(null);
	const [page, setPage] = useState(1);

	const { data: serversResult } = useCollection("servers", { perPage: 100 });
	const { data: volumesResult, isLoading } = useCollection("volumes", {
		page,
		perPage: 25,
		sort: "name",
		filter: andFilters([
			serverFilter !== "all" ? eqFilter("server", serverFilter) : "",
			containsAnyFilter(["name", "driver"], search),
		]),
	});

	useRealtime("volumes");

	const servers = (serversResult?.items ?? []) as ServerRecord[];
	const volumes = useMemo(
		() => (volumesResult?.items ?? []) as VolumeRecord[],
		[volumesResult?.items],
	);
	const totalVolumes = volumesResult?.totalItems ?? 0;

	const removeMut = useRemoveVolume();
	const inspectMut = useInspectVolume();

	const serverName = (serverId: string) => {
		const s = servers.find((sv) => sv.id === serverId);
		return s?.hostname ?? "—";
	};

	const handleInspect = (vol: VolumeRecord) => {
		inspectMut.mutate(vol.id, {
			onSuccess: (data) => setInspectData(data as unknown as Record<string, unknown>),
		});
	};

	return (
		<div className="flex flex-col gap-6">
			{/* Header */}
			<div className="flex items-center justify-between">
				<div className="flex items-center gap-2">
					<HardDrive className="h-5 w-5 text-muted-foreground" />
					<h1 className="text-2xl font-semibold">Volumes</h1>
					{!isLoading && <span className="text-sm text-muted-foreground">({totalVolumes})</span>}
				</div>
				<Button size="sm" onClick={() => setCreateOpen(true)} disabled={serverFilter === "all"}>
					<Plus className="mr-1.5 h-3.5 w-3.5" />
					Create Volume
				</Button>
			</div>

			{/* Filters */}
			<div className="flex items-center gap-3">
				<Select
					value={serverFilter}
					onValueChange={(value) => {
						setServerFilter(value);
						setPage(1);
					}}
				>
					<SelectTrigger className="w-48">
						<SelectValue placeholder="All Servers" />
					</SelectTrigger>
					<SelectContent>
						<SelectItem value="all">All Servers</SelectItem>
						{servers.map((s) => (
							<SelectItem key={s.id} value={s.id}>
								{s.hostname}
							</SelectItem>
						))}
					</SelectContent>
				</Select>
				<Input
					placeholder="Search volumes..."
					value={search}
					onChange={(e) => {
						setSearch(e.target.value);
						setPage(1);
					}}
					className="max-w-xs"
				/>
			</div>

			{/* Table */}
			{isLoading ? (
				<div className="space-y-2">
					{[1, 2, 3, 4].map((i) => (
						<Skeleton key={i} className="h-10 w-full" />
					))}
				</div>
			) : volumes.length === 0 ? (
				<p className="text-sm text-muted-foreground">No volumes found.</p>
			) : (
				<div className="rounded-md border">
					<Table>
						<TableHeader>
							<TableRow>
								<TableHead>Name</TableHead>
								<TableHead>Driver</TableHead>
								<TableHead>Mountpoint</TableHead>
								<TableHead>Size</TableHead>
								<TableHead>In Use</TableHead>
								{serverFilter === "all" && <TableHead>Server</TableHead>}
								<TableHead className="w-24" />
							</TableRow>
						</TableHeader>
						<TableBody>
							{volumes.map((vol) => (
								<TableRow key={vol.id}>
									<TableCell className="font-medium">{vol.name}</TableCell>
									<TableCell>
										<Badge variant="secondary">{vol.driver || "local"}</Badge>
									</TableCell>
									<TableCell className="max-w-48 truncate font-mono text-xs text-muted-foreground">
										{vol.mountpoint || "—"}
									</TableCell>
									<TableCell className="text-xs">{formatSize(vol.size)}</TableCell>
									<TableCell>
										{vol.in_use ? (
											<Badge variant="default">In Use</Badge>
										) : (
											<Badge variant="outline">Unused</Badge>
										)}
									</TableCell>
									{serverFilter === "all" && (
										<TableCell className="text-xs text-muted-foreground">
											{serverName(vol.server)}
										</TableCell>
									)}
									<TableCell>
										<div className="flex items-center gap-1">
											<Button
												variant="ghost"
												size="icon"
												className="h-7 w-7"
												onClick={() => handleInspect(vol)}
												disabled={inspectMut.isPending}
											>
												<Search className="h-3.5 w-3.5" />
											</Button>
											<Button
												variant="ghost"
												size="icon"
												className="h-7 w-7 text-destructive"
												onClick={() => setRemoveTarget(vol)}
											>
												<Trash2 className="h-3.5 w-3.5" />
											</Button>
										</div>
									</TableCell>
								</TableRow>
							))}
						</TableBody>
					</Table>
				</div>
			)}

			<PaginationControls
				page={page}
				totalPages={volumesResult?.totalPages ?? 0}
				totalItems={totalVolumes}
				onPageChange={setPage}
				label="volumes"
			/>

			{/* Create Dialog */}
			{serverFilter !== "all" && (
				<CreateVolumeDialog
					open={createOpen}
					onOpenChange={setCreateOpen}
					serverID={serverFilter}
				/>
			)}

			{/* Inspect Dialog */}
			<Dialog
				open={!!inspectData}
				onOpenChange={(open) => {
					if (!open) setInspectData(null);
				}}
			>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>Volume Details</DialogTitle>
						<DialogDescription>Raw inspect data from Docker.</DialogDescription>
					</DialogHeader>
					{inspectData && (
						<div className="max-h-80 overflow-y-auto rounded-md border bg-muted/30 p-3 font-mono text-xs leading-relaxed">
							<pre>{JSON.stringify(inspectData, null, 2)}</pre>
						</div>
					)}
				</DialogContent>
			</Dialog>

			{/* Remove Confirmation */}
			{removeTarget && (
				<CascadeDeleteDialog
					open={!!removeTarget}
					onOpenChange={(open) => {
						if (!open) setRemoveTarget(null);
					}}
					entityType="volume"
					entityId={removeTarget.id}
					entityName={removeTarget.name}
					onConfirm={(cascadeOptions: VolumeCascadeOptions) => {
						removeMut.mutate({
							id: removeTarget.id,
							force: removeTarget.in_use,
							cascade_options: cascadeOptions,
						});
						setRemoveTarget(null);
					}}
				/>
			)}
		</div>
	);
}
