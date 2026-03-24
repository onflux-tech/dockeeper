import { useState } from "react";
import { Archive } from "lucide-react";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { useCreatePolicy, useUpdatePolicy } from "@/hooks/use-backups";
import { describeCron } from "@/lib/format";
import type { BackupPolicyRecord, VolumeRecord } from "@/types/pocketbase";

interface PolicyDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	serverID: string;
	volumes: VolumeRecord[];
	editPolicy?: BackupPolicyRecord | null;
}

export function PolicyDialog({
	open,
	onOpenChange,
	serverID,
	volumes,
	editPolicy,
}: PolicyDialogProps) {
	const isEdit = !!editPolicy;

	const [volume, setVolume] = useState(editPolicy?.volume ?? "");
	const [schedule, setSchedule] = useState(editPolicy?.schedule ?? "0 2 * * *");
	const [retentionDays, setRetentionDays] = useState(
		editPolicy ? String(editPolicy.retention_days) : "30",
	);
	const [s3Bucket, setS3Bucket] = useState(editPolicy?.s3_bucket ?? "");
	const [s3Prefix, setS3Prefix] = useState(editPolicy?.s3_prefix ?? "dockeeper-backups");
	const [enabled, setEnabled] = useState(editPolicy?.enabled !== false ? "yes" : "no");

	const createMut = useCreatePolicy();
	const updateMut = useUpdatePolicy();
	const isPending = createMut.isPending || updateMut.isPending;

	const handleSubmit = () => {
		if (!volume || !schedule.trim()) return;

		if (isEdit && editPolicy) {
			updateMut.mutate(
				{
					id: editPolicy.id,
					fields: {
						volume,
						schedule: schedule.trim(),
						retention_days: parseInt(retentionDays, 10) || 30,
						s3_bucket: s3Bucket.trim(),
						s3_prefix: s3Prefix.trim(),
						enabled: enabled === "yes",
					},
				},
				{ onSuccess: () => onOpenChange(false) },
			);
		} else {
			createMut.mutate(
				{
					server_id: serverID,
					volume,
					schedule: schedule.trim(),
					retention_days: parseInt(retentionDays, 10) || 30,
					s3_bucket: s3Bucket.trim(),
					s3_prefix: s3Prefix.trim(),
					enabled: enabled === "yes",
				},
				{ onSuccess: () => onOpenChange(false) },
			);
		}
	};

	const handleClose = (nextOpen: boolean) => {
		if (isPending) return;
		onOpenChange(nextOpen);
	};

	return (
		<Dialog open={open} onOpenChange={handleClose}>
			<DialogContent>
				<DialogHeader>
					<DialogTitle className="flex items-center gap-2">
						<Archive className="h-5 w-5" />
						{isEdit ? "Edit Policy" : "Create Backup Policy"}
					</DialogTitle>
					<DialogDescription>
						{isEdit
							? "Update the backup policy configuration."
							: "Create a scheduled backup policy for a volume."}
					</DialogDescription>
				</DialogHeader>

				<div className="space-y-4">
					<div className="space-y-1.5">
						<label className="text-xs font-medium">Volume</label>
						<Select
							value={volume}
							onValueChange={setVolume}
							disabled={isEdit}
						>
							<SelectTrigger>
								<SelectValue placeholder="Select a volume" />
							</SelectTrigger>
							<SelectContent>
								{volumes.map((v) => (
									<SelectItem key={v.id} value={v.name}>
										{v.name}
									</SelectItem>
								))}
							</SelectContent>
						</Select>
					</div>

					<div className="grid grid-cols-2 gap-3">
						<div className="space-y-1.5">
							<label className="text-xs font-medium">
								Schedule (cron)
							</label>
							<Input
								placeholder="0 2 * * *"
								value={schedule}
								onChange={(e) => setSchedule(e.target.value)}
							/>
							<p className="text-[10px] text-muted-foreground">
								{describeCron(schedule)}
							</p>
						</div>
						<div className="space-y-1.5">
							<label className="text-xs font-medium">
								Retention (days)
							</label>
							<Input
								type="number"
								min={1}
								max={365}
								value={retentionDays}
								onChange={(e) => setRetentionDays(e.target.value)}
							/>
						</div>
					</div>

					<div className="grid grid-cols-2 gap-3">
						<div className="space-y-1.5">
							<label className="text-xs font-medium">S3 Bucket</label>
							<Input
								placeholder="my-backups"
								value={s3Bucket}
								onChange={(e) => setS3Bucket(e.target.value)}
							/>
						</div>
						<div className="space-y-1.5">
							<label className="text-xs font-medium">S3 Prefix</label>
							<Input
								placeholder="dockeeper-backups"
								value={s3Prefix}
								onChange={(e) => setS3Prefix(e.target.value)}
							/>
						</div>
					</div>

					<div className="space-y-1.5">
						<label className="text-xs font-medium">Enabled</label>
						<Select value={enabled} onValueChange={setEnabled}>
							<SelectTrigger className="w-32">
								<SelectValue />
							</SelectTrigger>
							<SelectContent>
								<SelectItem value="yes">Enabled</SelectItem>
								<SelectItem value="no">Disabled</SelectItem>
							</SelectContent>
						</Select>
					</div>
				</div>

				<DialogFooter>
					<Button variant="outline" onClick={() => handleClose(false)}>
						Cancel
					</Button>
					<Button
						onClick={handleSubmit}
						disabled={!volume || !schedule.trim() || isPending}
					>
						{isPending
							? isEdit
								? "Saving..."
								: "Creating..."
							: isEdit
								? "Save Changes"
								: "Create Policy"}
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}
