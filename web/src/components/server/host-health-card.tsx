import { Link } from "react-router";
import { HeartPulse } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { HealthCheckRecord } from "@/types/pocketbase";

interface HostHealthCardProps {
	checks: HealthCheckRecord[];
	serverID: string;
}

export function HostHealthCard({ checks, serverID }: HostHealthCardProps) {
	const passing = checks.filter((check) => check.status === "healthy").length;
	const failing = checks.filter((check) => check.status === "unhealthy").length;

	return (
		<Card>
			<CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
				<CardTitle className="flex items-center gap-2 text-sm font-medium">
					<HeartPulse className="h-4 w-4 text-muted-foreground" />
					Health checks
				</CardTitle>
				{failing > 0 ? (
					<Badge variant="destructive">{failing} failing</Badge>
				) : checks.length > 0 ? (
					<Badge variant="default">all passing</Badge>
				) : (
					<Badge variant="secondary">none</Badge>
				)}
			</CardHeader>
			<CardContent className="flex flex-col gap-1.5 text-sm">
				{checks.length === 0 ? (
					<p className="text-muted-foreground">No health checks configured for this server.</p>
				) : (
					<div className="flex items-center justify-between">
						<span className="text-muted-foreground">Checks</span>
						<Link to={`/health-checks?server=${serverID}`} className="tabular-nums hover:underline">
							{passing}/{checks.length} passing
						</Link>
					</div>
				)}
			</CardContent>
		</Card>
	);
}
