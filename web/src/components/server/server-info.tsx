import { Globe, Cpu, HardDrive, Network, Shield, Clock, Tag } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { relativeTime } from "@/lib/format";
import type { ServerRecord } from "@/types/pocketbase";

interface ServerInfoProps {
	server: ServerRecord | undefined;
	isLoading: boolean;
}

interface InfoItemProps {
	icon: React.ReactNode;
	label: string;
	value: React.ReactNode;
}

function InfoItem({ icon, label, value }: InfoItemProps) {
	return (
		<div className="flex items-start gap-3">
			<div className="mt-0.5 text-muted-foreground">{icon}</div>
			<div className="flex flex-col">
				<span className="text-xs text-muted-foreground">{label}</span>
				<span className="text-sm font-medium">{value}</span>
			</div>
		</div>
	);
}

export function ServerInfo({ server, isLoading }: ServerInfoProps) {
	if (isLoading || !server) {
		return (
			<Card>
				<CardHeader>
					<Skeleton className="h-5 w-32" />
				</CardHeader>
				<CardContent>
					<div className="grid grid-cols-2 gap-4 lg:grid-cols-3">
						{Array.from({ length: 6 }).map((_, i) => (
							<div key={i} className="flex items-start gap-3">
								<Skeleton className="h-4 w-4 mt-0.5" />
								<div className="flex flex-col gap-1">
									<Skeleton className="h-3 w-16" />
									<Skeleton className="h-4 w-24" />
								</div>
							</div>
						))}
					</div>
				</CardContent>
			</Card>
		);
	}

	const statusVariant =
		server.status === "online"
			? "default"
			: server.status === "error"
				? "destructive"
				: "secondary";

	return (
		<Card>
			<CardHeader>
				<CardTitle className="text-sm">Server Information</CardTitle>
			</CardHeader>
			<CardContent>
				<div className="grid grid-cols-2 gap-4 lg:grid-cols-3">
					<InfoItem
						icon={<Globe className="h-4 w-4" />}
						label="IP Address"
						value={server.ip_address || "—"}
					/>
					<InfoItem
						icon={<Cpu className="h-4 w-4" />}
						label="Architecture"
						value={`${server.os} / ${server.arch}`}
					/>
					<InfoItem
						icon={<HardDrive className="h-4 w-4" />}
						label="Docker"
						value={server.docker_version || "—"}
					/>
					<InfoItem
						icon={<Tag className="h-4 w-4" />}
						label="Agent Version"
						value={server.agent_version || "—"}
					/>
					<InfoItem
						icon={<Network className="h-4 w-4" />}
						label="Orchestrator"
						value={
							<Badge variant="outline" className="mt-0.5">
								{server.orchestrator}
							</Badge>
						}
					/>
					<InfoItem
						icon={<Shield className="h-4 w-4" />}
						label="Status"
						value={
							<Badge variant={statusVariant} className="mt-0.5">
								{server.status}
							</Badge>
						}
					/>
					<InfoItem
						icon={<Clock className="h-4 w-4" />}
						label="Last Seen"
						value={relativeTime(server.last_seen)}
					/>
					<InfoItem
						icon={<Clock className="h-4 w-4" />}
						label="Enrolled"
						value={relativeTime(server.enrolled_at)}
					/>
					<InfoItem
						icon={<Shield className="h-4 w-4" />}
						label="Cert Expires"
						value={relativeTime(server.cert_expires_at)}
					/>
				</div>
			</CardContent>
		</Card>
	);
}
