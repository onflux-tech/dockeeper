import {
	Globe,
	Tag,
	HardDrive,
	Play,
	Clock,
	FileText,
	Network,
	Lock,
	Layers,
	Box,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { relativeTime } from "@/lib/format";
import type { ContainerRecord } from "@/types/pocketbase";

interface ContainerInfoProps {
	container: ContainerRecord | undefined;
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
			<div className="flex flex-col min-w-0">
				<span className="text-xs text-muted-foreground">{label}</span>
				<span className="text-sm font-medium break-all">{value}</span>
			</div>
		</div>
	);
}

function StatusBadge({ status }: { status: string }) {
	const variant =
		status === "running"
			? "default"
			: status === "paused"
				? "secondary"
				: "destructive";
	return <Badge variant={variant}>{status}</Badge>;
}

function parseJSON(val: unknown): Record<string, unknown> | unknown[] | null {
	if (val == null) return null;
	if (typeof val === "object") return val as Record<string, unknown>;
	if (typeof val === "string") {
		try {
			return JSON.parse(val) as Record<string, unknown>;
		} catch {
			return null;
		}
	}
	return null;
}

function PortsList({ ports }: { ports: unknown }) {
	const parsed = parseJSON(ports);
	if (!parsed || typeof parsed !== "object") return <span>—</span>;

	const entries = Object.entries(parsed as Record<string, unknown>).filter(
		([, v]) => v != null && (Array.isArray(v) ? v.length > 0 : true),
	);
	if (entries.length === 0) return <span>—</span>;

	return (
		<div className="flex flex-wrap gap-1">
			{entries.map(([port]) => (
				<Badge key={port} variant="outline" className="text-xs font-mono">
					{port}
				</Badge>
			))}
		</div>
	);
}

function LabelsList({ labels }: { labels: unknown }) {
	const parsed = parseJSON(labels) as Record<string, string> | null;
	if (!parsed || Object.keys(parsed).length === 0) return <span>—</span>;

	const entries = Object.entries(parsed).slice(0, 8);
	return (
		<div className="flex flex-col gap-0.5">
			{entries.map(([k, v]) => (
				<span key={k} className="text-xs font-mono truncate">
					<span className="text-muted-foreground">{k}=</span>
					{String(v)}
				</span>
			))}
			{Object.keys(parsed).length > 8 && (
				<span className="text-xs text-muted-foreground">
					+{Object.keys(parsed).length - 8} more
				</span>
			)}
		</div>
	);
}

function NetworksList({ networks }: { networks: unknown }) {
	const parsed = parseJSON(networks) as Record<string, unknown> | null;
	if (!parsed || Object.keys(parsed).length === 0) return <span>—</span>;

	return (
		<div className="flex flex-wrap gap-1">
			{Object.keys(parsed).map((name) => (
				<Badge key={name} variant="outline" className="text-xs">
					{name}
				</Badge>
			))}
		</div>
	);
}

function VolumesList({ volumes }: { volumes: unknown }) {
	const parsed = parseJSON(volumes);
	if (!parsed || !Array.isArray(parsed) || parsed.length === 0)
		return <span>—</span>;

	return (
		<div className="flex flex-col gap-0.5">
			{(parsed as Array<Record<string, string>>).slice(0, 5).map((m, i) => (
				<span key={i} className="text-xs font-mono truncate">
					{m.Source ?? m.Name ?? "?"} → {m.Destination ?? "?"}
				</span>
			))}
			{parsed.length > 5 && (
				<span className="text-xs text-muted-foreground">
					+{parsed.length - 5} more
				</span>
			)}
		</div>
	);
}

function EnvVarsList({ envVars }: { envVars: unknown }) {
	const parsed = parseJSON(envVars);
	if (!parsed) return <span>—</span>;

	const entries = Array.isArray(parsed)
		? (parsed as string[])
		: Object.entries(parsed as Record<string, string>).map(
				([k, v]) => `${k}=${v}`,
			);

	if (entries.length === 0) return <span>—</span>;

	return (
		<div className="flex flex-col gap-0.5">
			{entries.slice(0, 6).map((entry, i) => {
				const eqIdx = String(entry).indexOf("=");
				const name = eqIdx > 0 ? String(entry).slice(0, eqIdx) : String(entry);
				return (
					<span key={i} className="text-xs font-mono truncate">
						<span className="text-muted-foreground">{name}=</span>
						<span>••••••••</span>
					</span>
				);
			})}
			{entries.length > 6 && (
				<span className="text-xs text-muted-foreground">
					+{entries.length - 6} more
				</span>
			)}
		</div>
	);
}

export function ContainerInfo({ container, isLoading }: ContainerInfoProps) {
	if (isLoading || !container) {
		return (
			<Card>
				<CardHeader>
					<Skeleton className="h-5 w-40" />
				</CardHeader>
				<CardContent>
					<div className="grid grid-cols-2 gap-4 lg:grid-cols-3">
						{Array.from({ length: 9 }).map((_, i) => (
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

	return (
		<div className="flex flex-col gap-4">
			<Card>
				<CardHeader>
					<CardTitle className="text-sm">Container Information</CardTitle>
				</CardHeader>
				<CardContent>
					<div className="grid grid-cols-2 gap-4 lg:grid-cols-3">
						<InfoItem
							icon={<Box className="h-4 w-4" />}
							label="Container ID"
							value={
								<span className="font-mono text-xs">
									{container.container_id}
								</span>
							}
						/>
						<InfoItem
							icon={<Tag className="h-4 w-4" />}
							label="Image"
							value={container.image}
						/>
						<InfoItem
							icon={<Play className="h-4 w-4" />}
							label="Status"
							value={<StatusBadge status={container.status} />}
						/>
						<InfoItem
							icon={<FileText className="h-4 w-4" />}
							label="Command"
							value={
								<span className="font-mono text-xs">
									{container.command || "—"}
								</span>
							}
						/>
						<InfoItem
							icon={<FileText className="h-4 w-4" />}
							label="Entrypoint"
							value={
								<span className="font-mono text-xs">
									{container.entrypoint || "—"}
								</span>
							}
						/>
						<InfoItem
							icon={<HardDrive className="h-4 w-4" />}
							label="Restart Policy"
							value={container.restart_policy || "—"}
						/>
						<InfoItem
							icon={<Clock className="h-4 w-4" />}
							label="Created"
							value={relativeTime(container.container_created_at)}
						/>
						<InfoItem
							icon={<Clock className="h-4 w-4" />}
							label="Started"
							value={relativeTime(container.started_at)}
						/>
						<InfoItem
							icon={<Globe className="h-4 w-4" />}
							label="Ports"
							value={<PortsList ports={container.ports} />}
						/>
					</div>
				</CardContent>
			</Card>

			<div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
				<Card>
					<CardHeader>
						<CardTitle className="text-sm flex items-center gap-2">
							<Network className="h-4 w-4" /> Networks
						</CardTitle>
					</CardHeader>
					<CardContent>
						<NetworksList networks={container.networks} />
					</CardContent>
				</Card>

				<Card>
					<CardHeader>
						<CardTitle className="text-sm flex items-center gap-2">
							<Layers className="h-4 w-4" /> Volumes
						</CardTitle>
					</CardHeader>
					<CardContent>
						<VolumesList volumes={container.volumes} />
					</CardContent>
				</Card>

				<Card>
					<CardHeader>
						<CardTitle className="text-sm flex items-center gap-2">
							<Lock className="h-4 w-4" /> Environment
						</CardTitle>
					</CardHeader>
					<CardContent>
						<EnvVarsList envVars={container.env_vars} />
					</CardContent>
				</Card>
			</div>

			<Card>
				<CardHeader>
					<CardTitle className="text-sm flex items-center gap-2">
						<Tag className="h-4 w-4" /> Labels
					</CardTitle>
				</CardHeader>
				<CardContent>
					<LabelsList labels={container.labels} />
				</CardContent>
			</Card>
		</div>
	);
}
