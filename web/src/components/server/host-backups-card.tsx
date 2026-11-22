import { useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { PolicyDialog } from "@/components/backups/policy-dialog";
import { useCollection } from "@/hooks/use-collection";
import { useTriggerBackup } from "@/hooks/use-backups";
import { useHasCapability } from "@/hooks/use-permissions";
import { relativeTime } from "@/lib/format";
import type {
	BackupPolicyRecord,
	BackupRecordRecord,
	HostAppRecord,
	VolumeRecord,
} from "@/types/pocketbase";

const statusVariant: Record<string, "default" | "secondary" | "destructive"> = {
	success: "default",
	running: "secondary",
	failed: "destructive",
	pending: "secondary",
};

export function HostBackupsCard({ serverID }: { serverID: string }) {
	const canManage = useHasCapability("backups.manage");
	const [policyOpen, setPolicyOpen] = useState(false);
	const filter = `server = '${serverID}' && target_type = 'host_app'`;
	const { data: policiesResult } = useCollection("backup_policies", {
		perPage: 100,
		sort: "-created",
		filter,
	});
	const { data: recordsResult } = useCollection("backup_records", {
		perPage: 1,
		sort: "-started_at",
		filter,
	});
	const { data: volumesResult } = useCollection("volumes", {
		perPage: 500,
		filter: `server = '${serverID}'`,
	});
	const { data: hostAppsResult } = useCollection("host_apps", {
		perPage: 500,
		filter: `server = '${serverID}'`,
	});
	const triggerMut = useTriggerBackup();

	const recipes = (policiesResult?.items ?? []) as BackupPolicyRecord[];
	const last = (recordsResult?.items ?? [])[0] as BackupRecordRecord | undefined;
	const volumes = (volumesResult?.items ?? []) as VolumeRecord[];
	const hostApps = (hostAppsResult?.items ?? []) as HostAppRecord[];
	const latestPolicy = recipes[0];

	return (
		<>
			<Card className="gap-0 py-0">
				<CardHeader className="flex flex-row items-center justify-between space-y-0 p-6 pb-3">
					<CardTitle className="text-base font-semibold">Backups</CardTitle>
					{recipes.length === 0 ? (
						<Badge variant="outline" className="border-warning/40 bg-warning/10 text-warning">
							not set
						</Badge>
					) : (
						<Badge variant="secondary">
							{recipes.length} {recipes.length === 1 ? "recipe" : "recipes"}
						</Badge>
					)}
				</CardHeader>
				<CardContent className="flex flex-col gap-2.5 p-6 pt-0 text-xs">
					{last ? (
						<div className="flex items-center justify-between gap-2">
							<span className="truncate" title={last.target_label || last.app_name}>
								{last.target_label || last.app_name || "Backup"}
							</span>
							<Badge variant={statusVariant[last.status] ?? "secondary"}>{last.status}</Badge>
						</div>
					) : (
						<p className="text-muted-foreground">No host app backup yet.</p>
					)}
					{last?.started_at ? (
						<p className="text-xs text-muted-foreground">
							Last run {relativeTime(last.started_at)}
						</p>
					) : null}
					<div className="flex flex-wrap items-center gap-2 pt-1">
						<Button
							size="sm"
							variant="outline"
							onClick={() => setPolicyOpen(true)}
							disabled={!canManage}
						>
							Add recipe
						</Button>
						{latestPolicy ? (
							<Button
								size="sm"
								onClick={() => triggerMut.mutate(latestPolicy.id)}
								disabled={!canManage || triggerMut.isPending}
							>
								Back up now
							</Button>
						) : null}
					</div>
				</CardContent>
			</Card>
			<PolicyDialog
				open={policyOpen}
				onOpenChange={setPolicyOpen}
				serverID={serverID}
				volumes={volumes}
				hostApps={hostApps}
				initialTargetType="host_app"
			/>
		</>
	);
}
