import { useState } from "react";
import { Link } from "react-router";
import {
	MoreHorizontal,
	Play,
	RotateCcw,
	Square,
	Pause,
	CirclePlay,
	Trash2,
	ArrowDownToLine,
} from "lucide-react";
import { toast } from "sonner";
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
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuSeparator,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Skeleton } from "@/components/ui/skeleton";
import { SortableHead } from "@/components/ui/sortable-head";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import {
	CascadeDeleteDialog,
	type ContainerCascadeOptions,
} from "@/components/cascade-delete-dialog";
import { BatchProgressDialog } from "@/components/batch-progress-dialog";
import { TagBadge } from "@/components/tag-badge";
import {
	useContainerActions,
	useRemoveContainerWithCascade,
	executeBulkRemoveWithCascade,
} from "@/hooks/use-container-actions";
import { useBatchSubmit } from "@/hooks/use-batch";
import { useTags } from "@/hooks/use-tags";
import { formatUptimeFromDate, formatBytes } from "@/lib/format";
import type { ContainerRecord } from "@/types/pocketbase";

// ── Types ────────────────────────────────────────────────────────────────────

interface ContainerTableProps {
	containers: ContainerRecord[];
	isLoading: boolean;
	showServer?: boolean;
	sortKey?: SortKey;
	sortDir?: SortDir;
	onSortChange?: (key: SortKey) => void;
}

export type SortKey = "name" | "status" | "image" | "created" | "uptime" | "cpu" | "ram";
export type SortDir = "asc" | "desc";

// ── Constants ────────────────────────────────────────────────────────────────

const statusColors: Record<string, "default" | "secondary" | "destructive"> = {
	running: "default",
	exited: "secondary",
	created: "secondary",
	restarting: "default",
	paused: "secondary",
	dead: "destructive",
	removing: "destructive",
	removed: "destructive",
};

// ── Component ────────────────────────────────────────────────────────────────

export function ContainerTable({
	containers,
	isLoading,
	showServer,
	sortKey: externalSortKey,
	sortDir: externalSortDir,
	onSortChange,
}: ContainerTableProps) {
	const [localSortKey, setLocalSortKey] = useState<SortKey>("name");
	const [localSortDir, setLocalSortDir] = useState<SortDir>("asc");
	const [selected, setSelected] = useState<Set<string>>(new Set());
	const [removeTarget, setRemoveTarget] = useState<{ ids: string[]; name: string } | null>(null);
	const [updateTarget, setUpdateTarget] = useState<string | null>(null);
	const [bulkLoading, setBulkLoading] = useState(false);
	const sortKey = externalSortKey ?? localSortKey;
	const sortDir = externalSortDir ?? localSortDir;

	const actions = useContainerActions();
	const removeCascade = useRemoveContainerWithCascade();
	const batchSubmit = useBatchSubmit();
	const [batchJobId, setBatchJobId] = useState<string | null>(null);
	const [batchAction, setBatchAction] = useState("");
	const [batchDialogOpen, setBatchDialogOpen] = useState(false);

	const { data: tagsData } = useTags("container");
	const allTags = tagsData?.items ?? [];

	// Build a map of resource_id → tags for quick lookup.
	const tagsByContainer = new Map<string, typeof allTags>();
	for (const tag of allTags) {
		const existing = tagsByContainer.get(tag.resource_id) ?? [];
		existing.push(tag);
		tagsByContainer.set(tag.resource_id, existing);
	}

	// Container name map for batch progress dialog.
	const containerNameMap = new Map(containers.map((c) => [c.id, c.name]));

	// ── Data is pre-sorted and pre-filtered by parent ───────────────────
	const processed = containers;

	// ── Selection helpers ────────────────────────────────────────────────
	const allSelected = processed.length > 0 && processed.every((c) => selected.has(c.id));

	function toggleAll() {
		if (allSelected) {
			setSelected(new Set());
		} else {
			setSelected(new Set(processed.map((c) => c.id)));
		}
	}

	function toggleOne(id: string) {
		setSelected((prev) => {
			const next = new Set(prev);
			if (next.has(id)) next.delete(id);
			else next.add(id);
			return next;
		});
	}

	function toggleSort(key: SortKey) {
		if (onSortChange) {
			onSortChange(key);
		} else {
			if (localSortKey === key) {
				setLocalSortDir((d) => (d === "asc" ? "desc" : "asc"));
			} else {
				setLocalSortKey(key);
				setLocalSortDir("asc");
			}
		}
	}

	// ── Bulk action via batch API ───────────────────────────────────────
	async function handleBulk(action: "start" | "stop" | "restart" | "remove") {
		const ids = [...selected];
		if (action === "remove") {
			setRemoveTarget({ ids, name: `${ids.length} containers` });
			return;
		}
		setBulkLoading(true);
		try {
			const result = await batchSubmit.mutateAsync({
				action,
				container_ids: ids,
			});
			setBatchJobId(result.job_id);
			setBatchAction(action);
			setBatchDialogOpen(true);
			setSelected(new Set());
		} catch {
			toast.error(`Failed to submit batch ${action}`);
		} finally {
			setBulkLoading(false);
		}
	}

	async function confirmRemove(cascadeOptions: ContainerCascadeOptions) {
		if (!removeTarget) return;
		if (removeTarget.ids.length === 1) {
			removeCascade.mutate({
				containerId: removeTarget.ids[0],
				cascadeOptions,
			});
		} else {
			setBulkLoading(true);
			const { succeeded, failed } = await executeBulkRemoveWithCascade(
				removeTarget.ids,
				cascadeOptions,
			);
			setBulkLoading(false);
			if (failed === 0) {
				toast.success(`Removed ${succeeded} containers`);
			} else {
				toast.warning(`Remove: ${succeeded} succeeded, ${failed} failed`);
			}
		}
		setSelected(new Set());
		setRemoveTarget(null);
	}

	// ── Loading skeleton ─────────────────────────────────────────────────
	if (isLoading) {
		return (
			<div className="flex flex-col gap-3">
				<div className="flex gap-2">
					<Skeleton className="h-9 w-64" />
					<Skeleton className="h-9 w-32" />
				</div>
				<div className="rounded-md border overflow-x-auto">
					<Table>
						<TableHeader>
							<TableRow>
								{Array.from({ length: 7 }).map((_, i) => (
									<TableHead key={i}>
										<Skeleton className="h-4 w-16" />
									</TableHead>
								))}
							</TableRow>
						</TableHeader>
						<TableBody>
							{Array.from({ length: 5 }).map((_, i) => (
								<TableRow key={i}>
									{Array.from({ length: 7 }).map((_, j) => (
										<TableCell key={j}>
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

	const colCount = (showServer ? 10 : 9) + 1; // +1 for checkbox

	return (
		<div className="flex flex-col gap-3">
			{/* Bulk actions bar */}
			{selected.size > 0 && (
				<div className="flex items-center gap-2 rounded-md border bg-muted/50 px-3 py-2 text-sm">
					<span className="font-medium">{selected.size} selected</span>
					<Button
						size="sm"
						variant="outline"
						disabled={bulkLoading}
						onClick={() => handleBulk("start")}
					>
						<Play className="mr-1 h-3 w-3" /> Start
					</Button>
					<Button
						size="sm"
						variant="outline"
						disabled={bulkLoading}
						onClick={() => handleBulk("stop")}
					>
						<Square className="mr-1 h-3 w-3" /> Stop
					</Button>
					<Button
						size="sm"
						variant="outline"
						disabled={bulkLoading}
						onClick={() => handleBulk("restart")}
					>
						<RotateCcw className="mr-1 h-3 w-3" /> Restart
					</Button>
					<Button
						size="sm"
						variant="destructive"
						disabled={bulkLoading}
						onClick={() => handleBulk("remove")}
					>
						<Trash2 className="mr-1 h-3 w-3" /> Remove
					</Button>
					<Button size="sm" variant="ghost" onClick={() => setSelected(new Set())}>
						Clear
					</Button>
				</div>
			)}

			{/* Table */}
			<div className="rounded-md border">
				<Table className="table-fixed">
					<TableHeader>
						<TableRow>
							<TableHead className="w-10">
								<Checkbox checked={allSelected} onCheckedChange={toggleAll} />
							</TableHead>
							<SortableHead
								className="w-24"
								label="Status"
								sortKey="status"
								currentKey={sortKey}
								currentDir={sortDir}
								onToggle={toggleSort}
							/>
							<SortableHead
								label="Name"
								sortKey="name"
								currentKey={sortKey}
								currentDir={sortDir}
								onToggle={toggleSort}
							/>
							<SortableHead
								className="w-24"
								label="Uptime"
								sortKey="uptime"
								currentKey={sortKey}
								currentDir={sortDir}
								onToggle={toggleSort}
							/>
							<SortableHead
								className="w-16"
								label="CPU%"
								sortKey="cpu"
								currentKey={sortKey}
								currentDir={sortDir}
								onToggle={toggleSort}
							/>
							<SortableHead
								className="w-20"
								label="RAM"
								sortKey="ram"
								currentKey={sortKey}
								currentDir={sortDir}
								onToggle={toggleSort}
							/>
							<SortableHead
								label="Image"
								sortKey="image"
								currentKey={sortKey}
								currentDir={sortDir}
								onToggle={toggleSort}
							/>
							<TableHead className="w-24">Tags</TableHead>
							{showServer && <TableHead className="w-28">Server</TableHead>}
							<TableHead className="w-10" />
						</TableRow>
					</TableHeader>
					<TableBody>
						{processed.length === 0 ? (
							<TableRow>
								<TableCell colSpan={colCount} className="py-8 text-center text-muted-foreground">
									No containers found.
								</TableCell>
							</TableRow>
						) : (
							processed.map((c) => (
								<TableRow key={c.id}>
									<TableCell>
										<Checkbox
											checked={selected.has(c.id)}
											onCheckedChange={() => toggleOne(c.id)}
										/>
									</TableCell>
									<TableCell>
										<Badge variant={statusColors[c.status] ?? "secondary"}>{c.status}</Badge>
									</TableCell>
									<TableCell className="overflow-hidden text-sm font-medium">
										<Link to={`/containers/${c.id}`} className="block truncate hover:underline">
											{c.name}
										</Link>
									</TableCell>
									<TableCell className="text-xs text-muted-foreground whitespace-nowrap">
										{c.status === "running" ? formatUptimeFromDate(c.started_at) : "—"}
									</TableCell>
									<TableCell className="text-xs tabular-nums font-medium">
										{c.current_cpu != null && c.current_cpu > 0
											? `${c.current_cpu.toFixed(1)}%`
											: "—"}
									</TableCell>
									<TableCell className="text-xs tabular-nums text-muted-foreground">
										{c.current_ram != null && c.current_ram > 0 ? formatBytes(c.current_ram) : "—"}
									</TableCell>
									<TableCell className="overflow-hidden truncate text-sm text-muted-foreground">
										{c.image}
									</TableCell>
									<TableCell>
										{(tagsByContainer.get(c.id)?.length ?? 0) > 0 ? (
											<div className="flex flex-wrap gap-1">
												{tagsByContainer.get(c.id)!.map((tag) => (
													<TagBadge key={tag.id} name={tag.name} color={tag.color} />
												))}
											</div>
										) : (
											<span className="text-xs text-muted-foreground">—</span>
										)}
									</TableCell>
									{showServer && (
										<TableCell className="text-sm text-muted-foreground">{c.server}</TableCell>
									)}
									<TableCell>
										<RowActions
											container={c}
											actions={actions}
											onRemove={() => setRemoveTarget({ ids: [c.id], name: c.name })}
											onUpdate={() => setUpdateTarget(c.id)}
										/>
									</TableCell>
								</TableRow>
							))
						)}
					</TableBody>
				</Table>
			</div>

			{/* Remove confirmation dialog */}
			{removeTarget && (
				<CascadeDeleteDialog
					open={removeTarget !== null}
					onOpenChange={(open) => {
						if (!open) setRemoveTarget(null);
					}}
					entityType="container"
					entityId={removeTarget.ids[0]}
					entityName={removeTarget.name}
					onConfirm={confirmRemove}
				/>
			)}

			{/* Update confirmation dialog */}
			<AlertDialog
				open={updateTarget !== null}
				onOpenChange={(open) => {
					if (!open) setUpdateTarget(null);
				}}
			>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>Update container?</AlertDialogTitle>
						<AlertDialogDescription>
							This will pull the latest image and recreate the container. The container will be
							briefly unavailable during the update.
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel>Cancel</AlertDialogCancel>
						<AlertDialogAction
							onClick={() => {
								if (updateTarget) actions.update.mutate(updateTarget);
								setUpdateTarget(null);
							}}
						>
							Update
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>

			{/* Batch progress dialog */}
			<BatchProgressDialog
				jobId={batchJobId}
				action={batchAction}
				open={batchDialogOpen}
				onOpenChange={(open) => {
					setBatchDialogOpen(open);
					if (!open) setBatchJobId(null);
				}}
				containerNames={containerNameMap}
			/>
		</div>
	);
}

// ── Subcomponents ────────────────────────────────────────────────────────────

function RowActions({
	container,
	actions,
	onRemove,
	onUpdate,
}: {
	container: ContainerRecord;
	actions: ReturnType<typeof useContainerActions>;
	onRemove: () => void;
	onUpdate: () => void;
}) {
	const isRunning = container.status === "running";
	const isPaused = container.status === "paused";
	const isStopped = container.status === "exited" || container.status === "created";

	return (
		<DropdownMenu>
			<DropdownMenuTrigger asChild>
				<Button variant="ghost" size="icon" className="h-8 w-8">
					<MoreHorizontal className="h-4 w-4" />
				</Button>
			</DropdownMenuTrigger>
			<DropdownMenuContent align="end">
				{isStopped && (
					<DropdownMenuItem onClick={() => actions.start.mutate(container.id)}>
						<Play className="mr-2 h-4 w-4" />
						Start
					</DropdownMenuItem>
				)}
				{isRunning && (
					<>
						<DropdownMenuItem onClick={() => actions.stop.mutate(container.id)}>
							<Square className="mr-2 h-4 w-4" />
							Stop
						</DropdownMenuItem>
						<DropdownMenuItem onClick={() => actions.pause.mutate(container.id)}>
							<Pause className="mr-2 h-4 w-4" />
							Pause
						</DropdownMenuItem>
					</>
				)}
				{isPaused && (
					<DropdownMenuItem onClick={() => actions.unpause.mutate(container.id)}>
						<CirclePlay className="mr-2 h-4 w-4" />
						Unpause
					</DropdownMenuItem>
				)}
				<DropdownMenuItem onClick={() => actions.restart.mutate(container.id)}>
					<RotateCcw className="mr-2 h-4 w-4" />
					Restart
				</DropdownMenuItem>
				<DropdownMenuItem onClick={onUpdate}>
					<ArrowDownToLine className="mr-2 h-4 w-4" />
					Update
				</DropdownMenuItem>
				<DropdownMenuSeparator />
				<DropdownMenuItem onClick={onRemove} className="text-destructive focus:text-destructive">
					<Trash2 className="mr-2 h-4 w-4" />
					Remove
				</DropdownMenuItem>
			</DropdownMenuContent>
		</DropdownMenu>
	);
}
