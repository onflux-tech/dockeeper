import { useState } from "react";
import { RotateCcw, Loader2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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
import { Input } from "@/components/ui/input";
import { useRestoreBackup } from "@/hooks/use-backups";
import { relativeTime } from "@/lib/format";
import type { BackupRecordRecord } from "@/types/pocketbase";

interface RecordsTableProps {
	records: BackupRecordRecord[];
	isLoading: boolean;
}

const STATUS_VARIANT: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
	success: "default",
	running: "secondary",
	failed: "destructive",
	pending: "outline",
};

function formatBytes(bytes: number): string {
	if (!bytes || bytes <= 0) return "—";
	const units = ["B", "KB", "MB", "GB", "TB"];
	let i = 0;
	let size = bytes;
	while (size >= 1024 && i < units.length - 1) {
		size /= 1024;
		i++;
	}
	return `${size.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

export function RecordsTable({ records, isLoading }: RecordsTableProps) {
	const [restoreTarget, setRestoreTarget] = useState<BackupRecordRecord | null>(null);
	const [confirmText, setConfirmText] = useState("");
	const restoreMut = useRestoreBackup();

	const handleRestore = () => {
		if (!restoreTarget || confirmText !== "RESTORE") return;
		restoreMut.mutate(restoreTarget.id, {
			onSuccess: () => {
				setRestoreTarget(null);
				setConfirmText("");
			},
		});
	};

	if (isLoading) {
		return (
			<div className="flex items-center gap-2 py-8 text-sm text-muted-foreground">
				<Loader2 className="h-4 w-4 animate-spin" />
				Loading records...
			</div>
		);
	}

	if (records.length === 0) {
		return <p className="py-4 text-sm text-muted-foreground">No backup records yet.</p>;
	}

	return (
		<>
			<div className="rounded-md border">
				<Table>
					<TableHeader>
						<TableRow>
							<TableHead>Volume</TableHead>
							<TableHead>Status</TableHead>
							<TableHead>Size</TableHead>
							<TableHead>S3 Key</TableHead>
							<TableHead>Started</TableHead>
							<TableHead>Completed</TableHead>
							<TableHead className="w-12" />
						</TableRow>
					</TableHeader>
					<TableBody>
						{records.map((rec) => (
							<TableRow key={rec.id}>
								<TableCell className="font-medium">{rec.volume_name}</TableCell>
								<TableCell>
									<div className="flex items-center gap-1.5">
										{rec.status === "running" && <Loader2 className="h-3 w-3 animate-spin" />}
										<Badge variant={STATUS_VARIANT[rec.status] ?? "outline"}>{rec.status}</Badge>
									</div>
								</TableCell>
								<TableCell className="font-mono text-xs">{formatBytes(rec.size)}</TableCell>
								<TableCell
									className="max-w-[200px] truncate font-mono text-xs text-muted-foreground"
									title={rec.s3_key}
								>
									{rec.s3_key || "—"}
								</TableCell>
								<TableCell className="text-xs text-muted-foreground">
									{relativeTime(rec.started_at)}
								</TableCell>
								<TableCell className="text-xs text-muted-foreground">
									{rec.completed_at ? relativeTime(rec.completed_at) : "—"}
								</TableCell>
								<TableCell>
									{rec.status === "success" && (
										<Button
											variant="ghost"
											size="icon"
											className="h-7 w-7"
											title="Restore this backup"
											onClick={() => setRestoreTarget(rec)}
										>
											<RotateCcw className="h-3.5 w-3.5" />
										</Button>
									)}
								</TableCell>
							</TableRow>
						))}
					</TableBody>
				</Table>
			</div>

			{/* Double-confirmation restore dialog */}
			<AlertDialog
				open={!!restoreTarget}
				onOpenChange={(open) => {
					if (!open) {
						setRestoreTarget(null);
						setConfirmText("");
					}
				}}
			>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>Restore Backup</AlertDialogTitle>
						<AlertDialogDescription asChild>
							<div className="space-y-3">
								<p>
									This will restore the volume{" "}
									<span className="font-mono font-semibold">{restoreTarget?.volume_name}</span> from
									backup. The current volume data will be{" "}
									<span className="font-semibold text-destructive">overwritten</span>.
								</p>
								<p className="text-xs">
									S3 key: <span className="font-mono">{restoreTarget?.s3_key}</span>
								</p>
								<div className="space-y-1.5">
									<label className="text-xs font-medium">
										Type <span className="font-mono">RESTORE</span> to confirm
									</label>
									<Input
										value={confirmText}
										onChange={(e) => setConfirmText(e.target.value)}
										placeholder="RESTORE"
										className="font-mono"
									/>
								</div>
							</div>
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel>Cancel</AlertDialogCancel>
						<AlertDialogAction
							disabled={confirmText !== "RESTORE" || restoreMut.isPending}
							onClick={(e) => {
								e.preventDefault();
								handleRestore();
							}}
						>
							{restoreMut.isPending ? "Restoring..." : "Confirm Restore"}
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
		</>
	);
}
