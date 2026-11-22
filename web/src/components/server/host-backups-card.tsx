import { Link } from "react-router";
import { Archive } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useCollection } from "@/hooks/use-collection";
import { relativeTime } from "@/lib/format";
import type { BackupPolicyRecord, BackupRecordRecord } from "@/types/pocketbase";

const statusVariant: Record<string, "default" | "secondary" | "destructive"> = {
	success: "default",
	running: "secondary",
	failed: "destructive",
	pending: "secondary",
};

export function HostBackupsCard({ serverID }: { serverID: string }) {
	const filter = `server = '${serverID}' && target_type = 'host_app'`;
	const { data: policiesResult } = useCollection("backup_policies", {
		perPage: 100,
		filter,
	});
	const { data: recordsResult } = useCollection("backup_records", {
		perPage: 1,
		sort: "-started_at",
		filter,
	});

	const recipes = (policiesResult?.items ?? []) as BackupPolicyRecord[];
	const last = (recordsResult?.items ?? [])[0] as BackupRecordRecord | undefined;

	return (
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
					<p className="text-xs text-muted-foreground">Last run {relativeTime(last.started_at)}</p>
				) : null}
				<Link to={`/backups?server=${serverID}`} className="text-xs hover:underline">
					View backups
				</Link>
			</CardContent>
		</Card>
	);
}
