import { useMemo, useState } from "react";
import { Archive, Plus, Pencil, Trash2, Play, Loader2, Search, MoreHorizontal } from "lucide-react";
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
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { PaginationControls } from "@/components/ui/pagination-controls";
import { useCollection } from "@/hooks/use-collection";
import { useRealtime } from "@/hooks/use-realtime";
import { useDeletePolicy, useTriggerBackup, useUpdatePolicy } from "@/hooks/use-backups";
import { PolicyDialog } from "@/components/backups/policy-dialog";
import { describeCron } from "@/lib/format";
import { andFilters, containsAnyFilter, eqFilter } from "@/lib/pocketbase-filters";
import { RecordsTable } from "@/components/backups/records-table";
import type {
	ServerRecord,
	VolumeRecord,
	BackupPolicyRecord,
	BackupRecordRecord,
} from "@/types/pocketbase";

export default function BackupsPage() {
	const [policiesPage, setPoliciesPage] = useState(1);
	const [recordsPage, setRecordsPage] = useState(1);
	const [serverFilter, setServerFilter] = useState<string>("all");
	const [search, setSearch] = useState("");
	const [policyOpen, setPolicyOpen] = useState(false);
	const [editPolicy, setEditPolicy] = useState<BackupPolicyRecord | null>(null);
	const [deleteTarget, setDeleteTarget] = useState<BackupPolicyRecord | null>(null);
	const [selectedPolicyId, setSelectedPolicyId] = useState<string>("");

	const serverId = serverFilter !== "all" ? serverFilter : undefined;

	const { data: serversResult } = useCollection("servers", { perPage: 100 });
	const { data: volumesResult } = useCollection("volumes", {
		perPage: 500,
		...(serverId ? { filter: `server = '${serverId}'` } : {}),
	});

	useRealtime("backup_policies");
	useRealtime("backup_records");

	const { data: policiesResult, isLoading: policiesLoading } = useCollection("backup_policies", {
		page: policiesPage,
		perPage: 25,
		sort: "-created",
		filter: andFilters([
			serverId ? eqFilter("server", serverId) : "",
			containsAnyFilter(["volume", "schedule", "s3_bucket"], search),
		]),
	});
	const { data: recordsResult, isLoading: recordsLoading } = useCollection("backup_records", {
		page: recordsPage,
		perPage: 25,
		sort: "-started_at",
		filter: andFilters([
			selectedPolicyId ? eqFilter("policy", selectedPolicyId) : "",
			!selectedPolicyId && serverId ? eqFilter("server", serverId) : "",
		]),
	});

	const deleteMut = useDeletePolicy();
	const triggerMut = useTriggerBackup();
	const updateMut = useUpdatePolicy();

	const servers = (serversResult?.items ?? []) as ServerRecord[];
	const volumes = useMemo(
		() => (volumesResult?.items ?? []) as VolumeRecord[],
		[volumesResult?.items],
	);
	const policies = useMemo(
		() => (policiesResult?.items ?? []) as BackupPolicyRecord[],
		[policiesResult?.items],
	);
	const records = useMemo(
		() => (recordsResult?.items ?? []) as BackupRecordRecord[],
		[recordsResult?.items],
	);

	const serverName = (sid: string) => {
		const s = servers.find((sv) => sv.id === sid);
		return s?.hostname ?? "—";
	};

	const handleToggleEnabled = (policy: BackupPolicyRecord) => {
		updateMut.mutate({
			id: policy.id,
			fields: { enabled: !policy.enabled },
		});
	};

	const runningCount = useMemo(
		() => records.filter((r) => r.status === "running").length,
		[records],
	);

	return (
		<div className="flex flex-col gap-6">
			{/* Header */}
			<div className="flex items-center justify-between">
				<div className="flex items-center gap-2">
					<Archive className="h-5 w-5 text-muted-foreground" />
					<h1 className="text-2xl font-semibold">Backups</h1>
					{!policiesLoading && (
						<Badge variant="secondary" className="ml-1">
							{policiesResult?.totalItems ?? 0}
						</Badge>
					)}
					{runningCount > 0 && (
						<Badge variant="secondary" className="gap-1">
							<Loader2 className="h-3 w-3 animate-spin" />
							{runningCount} running
						</Badge>
					)}
				</div>
				<Button
					size="sm"
					onClick={() => {
						setEditPolicy(null);
						setPolicyOpen(true);
					}}
					disabled={serverFilter === "all"}
				>
					<Plus className="mr-1.5 h-3.5 w-3.5" />
					New Policy
				</Button>
			</div>

			{/* Filters */}
			<div className="flex items-center gap-3">
				<Select
					value={serverFilter}
					onValueChange={(v) => {
						setServerFilter(v);
						setSelectedPolicyId("");
						setPoliciesPage(1);
						setRecordsPage(1);
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
				<div className="relative w-64">
					<Search className="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
					<Input
						placeholder="Search policies..."
						value={search}
						onChange={(e) => {
							setSearch(e.target.value);
							setPoliciesPage(1);
						}}
						className="pl-8"
					/>
				</div>
			</div>

			{/* Policies Table */}
			<div>
				<h2 className="mb-3 text-lg font-medium">Policies</h2>
				{policiesLoading ? (
					<div className="space-y-2">
						{[1, 2, 3].map((i) => (
							<Skeleton key={i} className="h-10 w-full" />
						))}
					</div>
				) : policies.length === 0 ? (
					<p className="text-sm text-muted-foreground">
						No backup policies found.{" "}
						{serverFilter === "all"
							? "Select a server to create one."
							: 'Click "New Policy" to create one.'}
					</p>
				) : (
					<div className="rounded-md border">
						<Table>
							<TableHeader>
								<TableRow>
									<TableHead>Volume</TableHead>
									<TableHead>Schedule</TableHead>
									<TableHead>Retention</TableHead>
									<TableHead>S3 Bucket</TableHead>
									<TableHead>Status</TableHead>
									{serverFilter === "all" && <TableHead>Server</TableHead>}
									<TableHead className="w-12" />
								</TableRow>
							</TableHeader>
							<TableBody>
								{policies.map((policy) => (
									<TableRow
										key={policy.id}
										className={
											selectedPolicyId === policy.id
												? "bg-muted/50"
												: "cursor-pointer hover:bg-muted/30"
										}
										onClick={() => {
											setSelectedPolicyId(selectedPolicyId === policy.id ? "" : policy.id);
											setRecordsPage(1);
										}}
									>
										<TableCell className="font-medium">{policy.volume}</TableCell>
										<TableCell>
											<div className="flex flex-col">
												<span className="font-mono text-xs">{policy.schedule}</span>
												<span className="text-[10px] text-muted-foreground">
													{describeCron(policy.schedule)}
												</span>
											</div>
										</TableCell>
										<TableCell className="text-xs">{policy.retention_days}d</TableCell>
										<TableCell className="font-mono text-xs text-muted-foreground">
											{policy.s3_bucket || "—"}
										</TableCell>
										<TableCell>
											<Badge
												variant={policy.enabled ? "default" : "outline"}
												className="cursor-pointer"
												onClick={(e) => {
													e.stopPropagation();
													handleToggleEnabled(policy);
												}}
											>
												{policy.enabled ? "Enabled" : "Disabled"}
											</Badge>
										</TableCell>
										{serverFilter === "all" && (
											<TableCell className="text-xs text-muted-foreground">
												{serverName(policy.server)}
											</TableCell>
										)}
										<TableCell>
											<DropdownMenu>
												<DropdownMenuTrigger asChild>
													<Button
														variant="ghost"
														size="icon"
														className="h-8 w-8"
														onClick={(e) => e.stopPropagation()}
													>
														<MoreHorizontal className="h-4 w-4" />
													</Button>
												</DropdownMenuTrigger>
												<DropdownMenuContent align="end">
													<DropdownMenuItem
														onClick={(e) => {
															e.stopPropagation();
															triggerMut.mutate(policy.id);
														}}
														disabled={triggerMut.isPending}
													>
														<Play className="mr-2 h-3.5 w-3.5" />
														Trigger Backup
													</DropdownMenuItem>
													<DropdownMenuItem
														onClick={(e) => {
															e.stopPropagation();
															setEditPolicy(policy);
															setPolicyOpen(true);
														}}
													>
														<Pencil className="mr-2 h-3.5 w-3.5" />
														Edit
													</DropdownMenuItem>
													<DropdownMenuItem
														className="text-destructive"
														onClick={(e) => {
															e.stopPropagation();
															setDeleteTarget(policy);
														}}
													>
														<Trash2 className="mr-2 h-3.5 w-3.5" />
														Delete
													</DropdownMenuItem>
												</DropdownMenuContent>
											</DropdownMenu>
										</TableCell>
									</TableRow>
								))}
							</TableBody>
						</Table>
					</div>
				)}

				<PaginationControls
					page={policiesPage}
					totalPages={policiesResult?.totalPages ?? 0}
					totalItems={policiesResult?.totalItems ?? 0}
					onPageChange={setPoliciesPage}
					label="policies"
				/>
			</div>

			{/* Backup Records */}
			<div>
				<h2 className="mb-3 text-lg font-medium">
					Backup Records
					{selectedPolicyId && (
						<span className="ml-2 text-sm font-normal text-muted-foreground">
							(filtered by selected policy —{" "}
							<button
								className="underline"
								onClick={() => {
									setSelectedPolicyId("");
									setRecordsPage(1);
								}}
							>
								show all
							</button>
							)
						</span>
					)}
				</h2>
				<RecordsTable records={records} isLoading={recordsLoading} />
				<PaginationControls
					page={recordsPage}
					totalPages={recordsResult?.totalPages ?? 0}
					totalItems={recordsResult?.totalItems ?? 0}
					onPageChange={setRecordsPage}
					label="records"
				/>
			</div>

			{/* Policy Dialog (Create/Edit) */}
			{(serverFilter !== "all" || editPolicy) && (
				<PolicyDialog
					key={editPolicy?.id ?? "create"}
					open={policyOpen}
					onOpenChange={(open) => {
						setPolicyOpen(open);
						if (!open) setEditPolicy(null);
					}}
					serverID={editPolicy?.server ?? serverFilter}
					volumes={volumes}
					editPolicy={editPolicy}
				/>
			)}

			{/* Delete Policy Confirmation */}
			<AlertDialog
				open={!!deleteTarget}
				onOpenChange={(open) => {
					if (!open) setDeleteTarget(null);
				}}
			>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>Delete Backup Policy</AlertDialogTitle>
						<AlertDialogDescription>
							Are you sure you want to delete the backup policy for volume{" "}
							<span className="font-mono font-semibold">{deleteTarget?.volume}</span>? Existing
							backup records will remain but no new backups will be scheduled.
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel>Cancel</AlertDialogCancel>
						<AlertDialogAction
							onClick={() => {
								if (deleteTarget) {
									deleteMut.mutate(deleteTarget.id);
									setDeleteTarget(null);
								}
							}}
						>
							Delete
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
		</div>
	);
}
