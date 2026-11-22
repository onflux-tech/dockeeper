import { Activity, CheckCircle2, XCircle } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { relativeTime } from "@/lib/format";
import type { ConnectivityProbeDetail, ServerRecord } from "@/types/pocketbase";

const statusVariants: Record<string, "default" | "secondary" | "destructive"> = {
	healthy: "default",
	degraded: "destructive",
	unknown: "secondary",
};

function ProbeRow({ label, probe }: { label: string; probe?: ConnectivityProbeDetail }) {
	return (
		<div className="flex items-center justify-between gap-2">
			<span className="text-muted-foreground">{label}</span>
			<span className="flex items-center gap-1.5 tabular-nums">
				{probe?.ok ? (
					<CheckCircle2 className="h-3.5 w-3.5 text-emerald-500" />
				) : (
					<XCircle className="h-3.5 w-3.5 text-destructive" />
				)}
				{probe ? `${probe.ms} ms` : "—"}
				{probe?.method && probe.method !== "none" ? (
					<Badge variant="outline" className="ml-1 text-[10px]">
						{probe.method}
					</Badge>
				) : null}
			</span>
		</div>
	);
}

export function HostConnectivityCard({ server }: { server: ServerRecord }) {
	const details = server.connectivity_details;
	const status = server.connectivity_status || "unknown";

	return (
		<Card>
			<CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
				<CardTitle className="flex items-center gap-2 text-sm font-medium">
					<Activity className="h-4 w-4 text-muted-foreground" />
					Connectivity
				</CardTitle>
				<Badge variant={statusVariants[status] ?? "secondary"}>{status}</Badge>
			</CardHeader>
			<CardContent className="flex flex-col gap-1.5 text-sm">
				<ProbeRow label="Internet" probe={details?.internet} />
				<ProbeRow label="DNS" probe={details?.dns} />
				<ProbeRow label="Gateway" probe={details?.gateway} />
				{details?.http ? (
					<div className="flex items-center justify-between border-t pt-1.5">
						<span className="text-muted-foreground">HTTP checks</span>
						<span className="tabular-nums">
							{details.http.ok}/{details.http.total} OK · {details.http.latency_ms} ms
						</span>
					</div>
				) : null}
				{details?.error ? <p className="text-xs text-destructive">{details.error}</p> : null}
				{server.connectivity_checked_at ? (
					<p className="text-xs text-muted-foreground">
						Checked {relativeTime(server.connectivity_checked_at)}
					</p>
				) : null}
			</CardContent>
		</Card>
	);
}
