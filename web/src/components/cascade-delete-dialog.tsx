import { useState } from "react";
import { AlertTriangle, Shield } from "lucide-react";
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
import { Checkbox } from "@/components/ui/checkbox";
import { Skeleton } from "@/components/ui/skeleton";
import {
	useCascadePreview,
	type CascadeEntityType,
	type ContainerCascadePreview,
	type VolumeCascadePreview,
	type ServerCascadePreview,
	type ResourceCount,
	type ResourceCountWithItems,
} from "@/hooks/use-cascade-preview";

// ── Cascade option types ────────────────────────────

export interface ContainerCascadeOptions {
	remove_alerts: boolean;
	remove_volumes: boolean;
	remove_backups: boolean;
	remove_notifications: boolean;
}

export interface VolumeCascadeOptions {
	keep_backup_records: boolean;
}

// ── Props ────────────────────────────────────────────

interface BaseProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	entityName: string;
}

interface ContainerDialogProps extends BaseProps {
	entityType: "container";
	entityId: string;
	onConfirm: (options: ContainerCascadeOptions) => void;
}

interface VolumeDialogProps extends BaseProps {
	entityType: "volume";
	entityId: string;
	onConfirm: (options: VolumeCascadeOptions) => void;
}

interface ServerDialogProps extends BaseProps {
	entityType: "server";
	entityId: string;
	onConfirm: () => void;
}

type CascadeDeleteDialogProps = ContainerDialogProps | VolumeDialogProps | ServerDialogProps;

// ── Component ────────────────────────────────────────

export function CascadeDeleteDialog(props: CascadeDeleteDialogProps) {
	const { open, onOpenChange, entityId } = props;

	return (
		<AlertDialog open={open} onOpenChange={onOpenChange}>
			{/* Key forces remount on entity change, resetting all state */}
			<CascadeDialogContent key={entityId} {...props} />
		</AlertDialog>
	);
}

function CascadeDialogContent(props: CascadeDeleteDialogProps) {
	const { entityType, entityId, entityName } = props;

	const { data: preview, isLoading } = useCascadePreview(entityType, entityId);

	// Container cascade options (defaults from spec: remove_alerts = true)
	const [removeAlerts, setRemoveAlerts] = useState(true);
	const [removeVolumes, setRemoveVolumes] = useState(false);
	const [removeBackups, setRemoveBackups] = useState(false);
	const [removeNotifications, setRemoveNotifications] = useState(false);

	// Volume cascade option
	const [keepBackupRecords, setKeepBackupRecords] = useState(true);

	function handleConfirm() {
		if (props.entityType === "container") {
			props.onConfirm({
				remove_alerts: removeAlerts,
				remove_volumes: removeVolumes,
				remove_backups: removeBackups,
				remove_notifications: removeNotifications,
			});
		} else if (props.entityType === "volume") {
			props.onConfirm({ keep_backup_records: keepBackupRecords });
		} else {
			props.onConfirm();
		}
	}

	const titleMap: Record<CascadeEntityType, string> = {
		container: "Remove container",
		volume: "Remove volume",
		server: "Delete server",
	};

	const actionLabel =
		entityType === "server" ? "Delete" : entityType === "volume" ? "Remove" : "Remove";

	return (
		<AlertDialogContent>
			<AlertDialogHeader>
				<AlertDialogTitle className="flex items-center gap-2">
					<AlertTriangle className="h-5 w-5 text-destructive" />
					{titleMap[entityType]}
				</AlertDialogTitle>
				<AlertDialogDescription asChild>
					<div className="space-y-3">
						<p>
							This will permanently remove{" "}
							<span className="font-mono font-semibold">{entityName}</span>.
							{entityType === "server" &&
								" All database records for this server will be permanently deleted."}
						</p>

						{/* Impact preview */}
						{isLoading ? (
							<PreviewSkeleton />
						) : preview ? (
							<ImpactPreview entityType={entityType} preview={preview} />
						) : null}

						{/* Cascade options */}
						{entityType === "container" && !isLoading && preview && (
							<ContainerOptions
								preview={preview as ContainerCascadePreview}
								removeAlerts={removeAlerts}
								setRemoveAlerts={setRemoveAlerts}
								removeVolumes={removeVolumes}
								setRemoveVolumes={setRemoveVolumes}
								removeBackups={removeBackups}
								setRemoveBackups={setRemoveBackups}
								removeNotifications={removeNotifications}
								setRemoveNotifications={setRemoveNotifications}
							/>
						)}

						{entityType === "volume" && !isLoading && preview && (
							<VolumeOptions
								preview={preview as VolumeCascadePreview}
								keepBackupRecords={keepBackupRecords}
								setKeepBackupRecords={setKeepBackupRecords}
							/>
						)}
					</div>
				</AlertDialogDescription>
			</AlertDialogHeader>
			<AlertDialogFooter>
				<AlertDialogCancel>Cancel</AlertDialogCancel>
				<AlertDialogAction
					onClick={handleConfirm}
					className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
					disabled={isLoading}
				>
					{actionLabel}
				</AlertDialogAction>
			</AlertDialogFooter>
		</AlertDialogContent>
	);
}

// ── Subcomponents ────────────────────────────────────

function PreviewSkeleton() {
	return (
		<div className="space-y-2 rounded-md border p-3">
			<Skeleton className="h-4 w-32" />
			<Skeleton className="h-3 w-48" />
			<Skeleton className="h-3 w-40" />
			<Skeleton className="h-3 w-36" />
		</div>
	);
}

function ImpactPreview({
	entityType,
	preview,
}: {
	entityType: CascadeEntityType;
	preview: ContainerCascadePreview | VolumeCascadePreview | ServerCascadePreview;
}) {
	const impactRows: { label: string; count: ResourceCount | ResourceCountWithItems }[] = [];

	if (entityType === "container") {
		const p = preview as ContainerCascadePreview;
		if (p.impact.alert_rules.deletable > 0 || p.impact.alert_rules.protected > 0)
			impactRows.push({ label: "Alert rules", count: p.impact.alert_rules });
		if (p.impact.volumes.deletable > 0 || p.impact.volumes.protected > 0)
			impactRows.push({ label: "Volumes", count: p.impact.volumes });
		if (p.impact.notification_logs.deletable > 0)
			impactRows.push({ label: "Notification logs", count: p.impact.notification_logs });
		if (p.impact.backup_policies.deletable > 0)
			impactRows.push({ label: "Backup policies", count: p.impact.backup_policies });
		if (p.impact.terminal_sessions.deletable > 0)
			impactRows.push({ label: "Terminal sessions", count: p.impact.terminal_sessions });
		if (p.impact.metrics_records.deletable > 0)
			impactRows.push({ label: "Metrics records", count: p.impact.metrics_records });
	} else if (entityType === "volume") {
		const p = preview as VolumeCascadePreview;
		if (p.impact.backup_policies.deletable > 0)
			impactRows.push({ label: "Backup policies", count: p.impact.backup_policies });
		if (p.impact.backup_records.deletable > 0)
			impactRows.push({ label: "Backup records", count: p.impact.backup_records });
	} else {
		const p = preview as ServerCascadePreview;
		const entries: [string, ResourceCount][] = [
			["Containers", p.impact.containers],
			["Volumes", p.impact.volumes],
			["Images", p.impact.images],
			["Networks", p.impact.networks],
			["Compose stacks", p.impact.compose_stacks],
			["Alert rules", p.impact.alert_rules],
			["Backup policies", p.impact.backup_policies],
			["Backup records", p.impact.backup_records],
			["Notification logs", p.impact.notification_logs],
			["Terminal sessions", p.impact.terminal_sessions],
			["Traefik routes", p.impact.traefik_routes],
			["Metrics records", p.impact.metrics_records],
		];
		for (const [label, count] of entries) {
			if (count.deletable > 0 || count.protected > 0) impactRows.push({ label, count });
		}
	}

	// Volume in_use_by
	const volumeInUseBy =
		entityType === "volume" ? ((preview as VolumeCascadePreview).in_use_by ?? []) : [];

	if (impactRows.length === 0 && volumeInUseBy.length === 0) return null;

	return (
		<div className="rounded-md border bg-muted/30 p-3">
			<p className="mb-2 text-xs font-medium text-foreground">Impact summary</p>
			<div className="space-y-1">
				{impactRows.map(({ label, count }) => (
					<div key={label} className="flex items-center justify-between text-xs">
						<span>{label}</span>
						<div className="flex items-center gap-1.5">
							{count.deletable > 0 && (
								<span className="text-destructive">{count.deletable} to delete</span>
							)}
							{count.protected > 0 && (
								<Badge variant="outline" className="h-5 text-[10px] gap-1">
									<Shield className="h-3 w-3" />
									{count.protected} protected
								</Badge>
							)}
						</div>
					</div>
				))}

				{/* Volume-specific: protected items with reason */}
				{entityType === "container" && isVolumePreview(preview) && (
					<VolumeItems preview={preview as ContainerCascadePreview} />
				)}

				{/* Volume in_use_by containers */}
				{volumeInUseBy.length > 0 && (
					<div className="mt-2 rounded border border-yellow-500/30 bg-yellow-500/10 px-2 py-1.5 text-xs">
						<span className="font-medium">In use by:</span> {volumeInUseBy.join(", ")}
					</div>
				)}
			</div>
		</div>
	);
}

function isVolumePreview(
	preview: ContainerCascadePreview | VolumeCascadePreview | ServerCascadePreview,
): preview is ContainerCascadePreview {
	return "volumes" in (preview as ContainerCascadePreview).impact;
}

function VolumeItems({ preview }: { preview: ContainerCascadePreview }) {
	const items = preview.impact.volumes.items;
	if (!items || items.length === 0) return null;

	return (
		<div className="mt-1.5 space-y-1 pl-1">
			{items.map((item) => (
				<div key={item.id} className="flex items-center gap-2 text-[11px]">
					<span className="font-mono">{item.name}</span>
					{item.eligible ? (
						<Badge variant="destructive" className="h-4 text-[9px]">
							Will delete
						</Badge>
					) : (
						<Badge variant="outline" className="h-4 text-[9px] gap-1">
							<Shield className="h-2.5 w-2.5" />
							In use
						</Badge>
					)}
					{item.reason && <span className="text-muted-foreground">{item.reason}</span>}
				</div>
			))}
		</div>
	);
}

function ContainerOptions({
	preview,
	removeAlerts,
	setRemoveAlerts,
	removeVolumes,
	setRemoveVolumes,
	removeBackups,
	setRemoveBackups,
	removeNotifications,
	setRemoveNotifications,
}: {
	preview: ContainerCascadePreview;
	removeAlerts: boolean;
	setRemoveAlerts: (v: boolean) => void;
	removeVolumes: boolean;
	setRemoveVolumes: (v: boolean) => void;
	removeBackups: boolean;
	setRemoveBackups: (v: boolean) => void;
	removeNotifications: boolean;
	setRemoveNotifications: (v: boolean) => void;
}) {
	const hasAlerts = preview.impact.alert_rules.deletable > 0;
	const hasVolumes = preview.impact.volumes.deletable > 0;
	const hasBackups = preview.impact.backup_policies.deletable > 0;
	const hasNotifications = preview.impact.notification_logs.deletable > 0;

	if (!hasAlerts && !hasVolumes && !hasBackups && !hasNotifications) return null;

	return (
		<div className="space-y-2.5 rounded-md border p-3">
			<p className="text-xs font-medium text-foreground">Cascade options</p>
			{hasAlerts && (
				<CascadeCheckbox
					id="remove-alerts"
					checked={removeAlerts}
					onCheckedChange={setRemoveAlerts}
					label="Delete alert rules"
					count={preview.impact.alert_rules.deletable}
				/>
			)}
			{hasVolumes && (
				<CascadeCheckbox
					id="remove-volumes"
					checked={removeVolumes}
					onCheckedChange={setRemoveVolumes}
					label="Remove exclusive volumes from host"
					count={preview.impact.volumes.deletable}
				/>
			)}
			{hasBackups && (
				<CascadeCheckbox
					id="remove-backups"
					checked={removeBackups}
					onCheckedChange={setRemoveBackups}
					label="Delete backup data for removed volumes"
					count={preview.impact.backup_policies.deletable}
				/>
			)}
			{hasNotifications && (
				<CascadeCheckbox
					id="remove-notifications"
					checked={removeNotifications}
					onCheckedChange={setRemoveNotifications}
					label="Delete notification logs"
					count={preview.impact.notification_logs.deletable}
				/>
			)}
		</div>
	);
}

function VolumeOptions({
	preview,
	keepBackupRecords,
	setKeepBackupRecords,
}: {
	preview: VolumeCascadePreview;
	keepBackupRecords: boolean;
	setKeepBackupRecords: (v: boolean) => void;
}) {
	const hasBackups = preview.impact.backup_records.deletable > 0;
	if (!hasBackups) return null;

	return (
		<div className="space-y-2.5 rounded-md border p-3">
			<p className="text-xs font-medium text-foreground">Cascade options</p>
			<CascadeCheckbox
				id="keep-backups"
				checked={keepBackupRecords}
				onCheckedChange={setKeepBackupRecords}
				label="Preserve backup records (detach from policy)"
				count={preview.impact.backup_records.deletable}
			/>
		</div>
	);
}

function CascadeCheckbox({
	id,
	checked,
	onCheckedChange,
	label,
	count,
}: {
	id: string;
	checked: boolean;
	onCheckedChange: (v: boolean) => void;
	label: string;
	count: number;
}) {
	return (
		<div className="flex items-center gap-2">
			<Checkbox id={id} checked={checked} onCheckedChange={(v) => onCheckedChange(v === true)} />
			<label htmlFor={id} className="text-xs cursor-pointer select-none">
				{label}
				{count > 0 && <span className="ml-1 text-muted-foreground">({count})</span>}
			</label>
		</div>
	);
}
