import { useState } from "react";
import { Link } from "react-router";
import { Archive, Play, Plus } from "lucide-react";
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
			<Card>
				<CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
					<CardTitle className="flex items-center gap-2 text-sm font-medium">
						<Archive className="h-4 w-4 text-muted-foreground" />
						Backups
					</CardTitle>
					<Badge variant="secondary">
						{recipes.length} {recipes.length === 1 ? "recipe" : "recipes"}
					</Badge>
				</CardHeader>
				<CardContent className="flex flex-col gap-1.5 text-sm">
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
							<Plus className="mr-1 h-3.5 w-3.5" />
							Add recipe
						</Button>
						<Button
							size="sm"
							onClick={() =>
								latestPolicy ? triggerMut.mutate(latestPolicy.id) : setPolicyOpen(true)
							}
							disabled={!canManage || triggerMut.isPending}
						>
							<Play className="mr-1 h-3.5 w-3.5" />
							Back up now
						</Button>
						<Link to={`/backups?server=${serverID}`} className="text-xs hover:underline">
							View backups
						</Link>
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
