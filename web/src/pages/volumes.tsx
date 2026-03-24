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
	AlertDialog,
	AlertDialogAction,
	AlertDialogCancel,
	AlertDialogContent,
	AlertDialogDescription,
	AlertDialogFooter,
	AlertDialogHeader,
	AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { useCollection } from "@/hooks/use-collection";
import { useRealtime } from "@/hooks/use-realtime";
import { useRemoveVolume, useInspectVolume } from "@/hooks/use-volumes";
import { CreateVolumeDialog } from "@/components/volumes/create-volume-dialog";
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

	const { data: serversResult } = useCollection("servers", { perPage: 100 });
	const { data: volumesResult, isLoading } = useCollection("volumes", {
		perPage: 200,
		sort: "name",
		...(serverFilter !== "all"
			? { filter: `server = '${serverFilter}'` }
			: {}),
	});

	useRealtime("volumes");

	const servers = (serversResult?.items ?? []) as ServerRecord[];
	const volumes = useMemo(
		() => (volumesResult?.items ?? []) as VolumeRecord[],
		[volumesResult?.items],
	);

	const removeMut = useRemoveVolume();
	const inspectMut = useInspectVolume();

	const filtered = useMemo(() => {
		if (!search) return volumes;
		const q = search.toLowerCase();
		return volumes.filter(
			(v) =>
				v.name.toLowerCase().includes(q) ||
				(v.driver ?? "").toLowerCase().includes(q),
		);
	}, [volumes, search]);

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
					{!isLoading && (
						<span className="text-sm text-muted-foreground">
							({filtered.length})
						</span>
					)}
				</div>
				<Button
					size="sm"
					onClick={() => setCreateOpen(true)}
					disabled={serverFilter === "all"}
				>
					<Plus className="mr-1.5 h-3.5 w-3.5" />
					Create Volume
				</Button>
			</div>

			{/* Filters */}
			<div className="flex items-center gap-3">
				<Select value={serverFilter} onValueChange={setServerFilter}>
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
					onChange={(e) => setSearch(e.target.value)}
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
			) : filtered.length === 0 ? (
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
							{filtered.map((vol) => (
								<TableRow key={vol.id}>
									<TableCell className="font-medium">{vol.name}</TableCell>
									<TableCell>
										<Badge variant="secondary">{vol.driver || "local"}</Badge>
									</TableCell>
									<TableCell className="max-w-48 truncate font-mono text-xs text-muted-foreground">
										{vol.mountpoint || "—"}
									</TableCell>
									<TableCell className="text-xs">
										{formatSize(vol.size)}
									</TableCell>
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
			<AlertDialog
				open={!!removeTarget}
				onOpenChange={(open) => {
					if (!open) setRemoveTarget(null);
				}}
			>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>Remove Volume</AlertDialogTitle>
						<AlertDialogDescription>
							{removeTarget?.in_use ? (
								<>
									<span className="font-semibold text-destructive">
										Warning:
									</span>{" "}
									Volume{" "}
									<span className="font-mono font-semibold">
										{removeTarget?.name}
									</span>{" "}
									is currently in use by one or more containers. Removing it may
									cause data loss.
								</>
							) : (
								<>
									Are you sure you want to remove the volume{" "}
									<span className="font-mono font-semibold">
										{removeTarget?.name}
									</span>
									? This action cannot be undone and all data will be lost.
								</>
							)}
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel>Cancel</AlertDialogCancel>
						<AlertDialogAction
							onClick={() => {
								if (removeTarget) {
									removeMut.mutate({
										id: removeTarget.id,
										force: removeTarget.in_use,
									});
									setRemoveTarget(null);
								}
							}}
						>
							{removeTarget?.in_use ? "Force Remove" : "Remove"}
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
		</div>
	);
}
