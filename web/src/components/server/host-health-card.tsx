import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { HealthCheckRecord } from "@/types/pocketbase";

interface HostHealthCardProps {
	checks: HealthCheckRecord[];
}

export function HostHealthCard({ checks }: HostHealthCardProps) {
	const failing = checks.filter((check) => check.status === "unhealthy").length;

	return (
		<Card className="gap-0 py-0">
			<CardHeader className="flex flex-row items-center justify-between space-y-0 p-6 pb-3">
				<CardTitle className="text-base font-semibold">Health checks</CardTitle>
				{failing > 0 ? (
					<span className="text-xs font-medium text-destructive">{failing} failing</span>
				) : checks.length > 0 ? (
					<span className="text-xs font-medium text-emerald-500">{checks.length} passing</span>
				) : (
					<span className="text-xs font-medium text-muted-foreground">none</span>
				)}
			</CardHeader>
			<CardContent className="p-6 pt-0 text-xs">
				{checks.length === 0 ? (
					<p className="text-muted-foreground">No health checks configured.</p>
				) : (
					<div className="flex flex-col gap-1.5">
						{checks.slice(0, 3).map((check) => (
							<div key={check.id} className="flex items-center justify-between gap-2">
								<span className="truncate text-muted-foreground">
									{check.name}
									{check.host ? ` · ${check.host}` : ""}
								</span>
								<span className="tabular-nums text-muted-foreground">
									{check.status === "healthy"
										? check.check_type === "http"
											? "200 OK"
											: "TCP OK"
										: check.status}
								</span>
							</div>
						))}
					</div>
				)}
			</CardContent>
		</Card>
	);
}
