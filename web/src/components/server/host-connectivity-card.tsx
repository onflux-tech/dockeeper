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
			<span className="flex items-center gap-2 text-muted-foreground">
				<span
					className={`h-1.5 w-1.5 rounded-full ${probe?.ok ? "bg-emerald-500" : "bg-destructive"}`}
				/>
				{label}
			</span>
			<span className="tabular-nums">{probe ? `${probe.ms} ms` : "—"}</span>
		</div>
	);
}

export function HostConnectivityCard({ server }: { server: ServerRecord }) {
	const details = server.connectivity_details;
	const status = server.connectivity_status || "unknown";

	return (
		<Card className="py-0">
			<CardHeader className="flex flex-row items-center justify-between space-y-0 p-6 pb-3">
				<CardTitle className="text-base font-semibold">Connectivity</CardTitle>
				<Badge variant={statusVariants[status] ?? "secondary"}>{status}</Badge>
			</CardHeader>
			<CardContent className="flex flex-col gap-2.5 p-6 pt-0 text-xs">
				<ProbeRow label="Internet" probe={details?.internet} />
				<ProbeRow label="DNS" probe={details?.dns} />
				<ProbeRow label="Gateway" probe={details?.gateway} />
				{details?.http ? (
					<div className="flex items-center justify-between border-t pt-2">
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
