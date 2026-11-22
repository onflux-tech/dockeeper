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
import { Textarea } from "@/components/ui/textarea";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { useCreatePolicy, useUpdatePolicy } from "@/hooks/use-backups";
import { describeCron } from "@/lib/format";
import type { BackupPolicyRecord, HostAppRecord, VolumeRecord } from "@/types/pocketbase";

interface PolicyDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	serverID: string;
	volumes: VolumeRecord[];
	hostApps: HostAppRecord[];
	editPolicy?: BackupPolicyRecord | null;
	initialTargetType?: "volume" | "host_app";
}

type AppSource = "systemd" | "pm2" | "process";

export function PolicyDialog({
	open,
	onOpenChange,
	serverID,
	volumes,
	hostApps,
	editPolicy,
	initialTargetType,
}: PolicyDialogProps) {
	const isEdit = !!editPolicy;

	const [targetType, setTargetType] = useState<"volume" | "host_app">(
		editPolicy?.target_type ?? initialTargetType ?? "volume",
	);
	const [volume, setVolume] = useState(editPolicy?.volume ?? "");
	const [appSource, setAppSource] = useState<AppSource>(
		(editPolicy?.app_source as AppSource) || "systemd",
	);
	const [appName, setAppName] = useState(editPolicy?.app_name ?? "");
	const [pathsText, setPathsText] = useState((editPolicy?.paths ?? []).join("\n"));
	const [excludeText, setExcludeText] = useState((editPolicy?.exclude ?? []).join("\n"));
	const [maxBytes, setMaxBytes] = useState(
		editPolicy?.max_bytes ? String(editPolicy.max_bytes) : "",
	);
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

	const paths = pathsText
		.split("\n")
		.map((line) => line.trim())
		.filter(Boolean);
	const excludes = excludeText
		.split("\n")
		.map((line) => line.trim())
		.filter(Boolean);
	const filteredApps = hostApps.filter((app) => app.source === appSource);

	const targetValid = targetType === "volume" ? !!volume : !!appName && paths.length > 0;
	const canSubmit = targetValid && !!schedule.trim();

	const handleSubmit = () => {
		if (!canSubmit) return;

		const targetFields = {
			target_type: targetType,
			volume: targetType === "volume" ? volume : "",
			app_source: targetType === "host_app" ? appSource : "",
			app_name: targetType === "host_app" ? appName : "",
			paths: targetType === "host_app" ? paths : [],
			exclude: targetType === "host_app" ? excludes : [],
			max_bytes: targetType === "host_app" ? Number(maxBytes) || 0 : 0,
		};

		const fields = {
			...targetFields,
			schedule: schedule.trim(),
			retention_days: parseInt(retentionDays, 10) || 30,
			s3_bucket: s3Bucket.trim(),
			s3_prefix: s3Prefix.trim(),
			enabled: enabled === "yes",
		};

		if (isEdit && editPolicy) {
			updateMut.mutate({ id: editPolicy.id, fields }, { onSuccess: () => onOpenChange(false) });
			return;
		}

		createMut.mutate({ server_id: serverID, ...fields }, { onSuccess: () => onOpenChange(false) });
	};

	const handleClose = (nextOpen: boolean) => {
		if (isPending) return;
		onOpenChange(nextOpen);
	};

	return (
		<Dialog open={open} onOpenChange={handleClose}>
			<DialogContent className="max-h-[85vh] overflow-y-auto">
				<DialogHeader>
					<DialogTitle className="flex items-center gap-2">
						<Archive className="h-5 w-5" />
						{isEdit ? "Edit Policy" : "Create Backup Policy"}
					</DialogTitle>
					<DialogDescription>
						{isEdit
							? "Update the backup policy configuration."
							: "Create a scheduled backup policy for a volume or a host application."}
					</DialogDescription>
				</DialogHeader>

				<div className="space-y-4">
					<div className="space-y-1.5">
						<label className="text-xs font-medium">Target Type</label>
						<Select
							value={targetType}
							onValueChange={(value) => setTargetType(value as "volume" | "host_app")}
							disabled={isEdit}
						>
							<SelectTrigger>
								<SelectValue />
							</SelectTrigger>
							<SelectContent>
								<SelectItem value="volume">Volume</SelectItem>
								<SelectItem value="host_app">Host app</SelectItem>
							</SelectContent>
						</Select>
					</div>

					{targetType === "volume" ? (
						<div className="space-y-1.5">
							<label className="text-xs font-medium">Volume</label>
							<Select value={volume} onValueChange={setVolume} disabled={isEdit}>
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
					) : (
						<>
							<div className="grid grid-cols-2 gap-3">
								<div className="space-y-1.5">
									<label className="text-xs font-medium">App Source</label>
									<Select
										value={appSource}
										onValueChange={(value) => {
											setAppSource(value as AppSource);
											setAppName("");
										}}
										disabled={isEdit}
									>
										<SelectTrigger>
											<SelectValue />
										</SelectTrigger>
										<SelectContent>
											<SelectItem value="systemd">systemd</SelectItem>
											<SelectItem value="pm2">PM2</SelectItem>
											<SelectItem value="process">process</SelectItem>
										</SelectContent>
									</Select>
								</div>
								<div className="space-y-1.5">
									<label className="text-xs font-medium">Application</label>
									<Select value={appName} onValueChange={setAppName} disabled={isEdit}>
										<SelectTrigger>
											<SelectValue placeholder="Select an application" />
										</SelectTrigger>
										<SelectContent>
											{filteredApps.map((app) => (
												<SelectItem key={app.id} value={app.name}>
													{app.name}
												</SelectItem>
											))}
										</SelectContent>
									</Select>
								</div>
							</div>

							<div className="space-y-1.5">
								<label className="text-xs font-medium">Paths</label>
								<Textarea
									value={pathsText}
									onChange={(event) => setPathsText(event.target.value)}
									placeholder="/etc/nginx&#10;/var/www"
									rows={3}
									className="font-mono text-xs"
								/>
								<p className="text-[10px] text-muted-foreground">
									One absolute path per line. /proc, /sys, /dev, /boot and credential files are
									rejected.
								</p>
							</div>

							<div className="grid grid-cols-2 gap-3">
								<div className="space-y-1.5">
									<label className="text-xs font-medium">Exclude patterns</label>
									<Textarea
										value={excludeText}
										onChange={(event) => setExcludeText(event.target.value)}
										placeholder="*.log"
										rows={2}
										className="font-mono text-xs"
									/>
								</div>
								<div className="space-y-1.5">
									<label className="text-xs font-medium">Max bytes</label>
									<Input
										type="number"
										min={0}
										value={maxBytes}
										onChange={(event) => setMaxBytes(event.target.value)}
										placeholder="20971520000"
									/>
								</div>
							</div>
						</>
					)}

					<div className="grid grid-cols-2 gap-3">
						<div className="space-y-1.5">
							<label className="text-xs font-medium">Schedule (cron)</label>
							<Input
								placeholder="0 2 * * *"
								value={schedule}
								onChange={(event) => setSchedule(event.target.value)}
							/>
							<p className="text-[10px] text-muted-foreground">{describeCron(schedule)}</p>
						</div>
						<div className="space-y-1.5">
							<label className="text-xs font-medium">Retention (days)</label>
							<Input
								type="number"
								min={1}
								max={365}
								value={retentionDays}
								onChange={(event) => setRetentionDays(event.target.value)}
							/>
						</div>
					</div>

					<div className="grid grid-cols-2 gap-3">
						<div className="space-y-1.5">
							<label className="text-xs font-medium">S3 Bucket</label>
							<Input
								placeholder="my-backups"
								value={s3Bucket}
								onChange={(event) => setS3Bucket(event.target.value)}
							/>
						</div>
						<div className="space-y-1.5">
							<label className="text-xs font-medium">S3 Prefix</label>
							<Input
								placeholder="dockeeper-backups"
								value={s3Prefix}
								onChange={(event) => setS3Prefix(event.target.value)}
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
					<Button onClick={handleSubmit} disabled={!canSubmit || isPending}>
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
