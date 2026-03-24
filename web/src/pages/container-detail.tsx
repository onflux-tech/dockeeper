import { useParams, Link } from "react-router";
import { ArrowLeft, TerminalSquare } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useRecord } from "@/hooks/use-record";
import { useRealtime } from "@/hooks/use-realtime";
import { useContainerMetrics } from "@/hooks/use-container-metrics";
import { ContainerInfo } from "@/components/container/container-info";
import { ContainerMetrics } from "@/components/container/container-metrics";
import { LogViewer } from "@/components/container/log-viewer";
import { ResourceEditor } from "@/components/container/resource-editor";
import { EnvEditor } from "@/components/container/env-editor";

const statusVariantMap: Record<string, "default" | "secondary" | "destructive"> = {
	running: "default",
	paused: "secondary",
	exited: "destructive",
	dead: "destructive",
};

export default function ContainerDetailPage() {
	const { id } = useParams();

	const { data: container, isLoading } = useRecord("containers", id);
	const { data: metricsData, isLoading: loadingMetrics } =
		useContainerMetrics(id);

	useRealtime("containers");
	useRealtime("container_metrics_raw");

	const metrics = metricsData ?? [];
	const statusVariant = statusVariantMap[container?.status ?? ""] ?? "secondary";

	return (
		<div className="flex flex-col gap-6">
			{/* Header */}
			<div className="flex items-center gap-3">
				<Button variant="ghost" size="icon" className="h-8 w-8" asChild>
					<Link to="/containers">
						<ArrowLeft className="h-4 w-4" />
					</Link>
				</Button>
				{isLoading ? (
					<>
						<Skeleton className="h-7 w-48" />
						<Skeleton className="h-5 w-16 rounded-full" />
					</>
				) : (
					<>
						<h1 className="text-2xl font-semibold">
							{container?.name || "Container"}
						</h1>
						{container && (
							<Badge variant={statusVariant}>{container.status}</Badge>
						)}
						{id && container?.status === "running" && (
							<Button variant="outline" size="sm" className="ml-auto" asChild>
								<Link to={`/containers/${id}/terminal`}>
									<TerminalSquare className="mr-1.5 h-4 w-4" />
									Terminal
								</Link>
							</Button>
						)}
					</>
				)}
			</div>

			{/* Tabs */}
			<Tabs defaultValue="info">
				<TabsList>
					<TabsTrigger value="info">Info</TabsTrigger>
					<TabsTrigger value="metrics">Metrics</TabsTrigger>
					<TabsTrigger value="resources">Resources</TabsTrigger>
					<TabsTrigger value="env">Environment</TabsTrigger>
					<TabsTrigger value="logs">Logs</TabsTrigger>
				</TabsList>

				<TabsContent value="info" className="mt-4">
					<ContainerInfo container={container} isLoading={isLoading} />
				</TabsContent>

				<TabsContent value="metrics" className="mt-4">
					<ContainerMetrics data={metrics} isLoading={loadingMetrics} />
				</TabsContent>

				<TabsContent value="resources" className="mt-4">
					<ResourceEditor container={container} isLoading={isLoading} />
				</TabsContent>

				<TabsContent value="env" className="mt-4">
					<EnvEditor container={container} isLoading={isLoading} />
				</TabsContent>

				<TabsContent value="logs" className="mt-4">
					{id && <LogViewer containerId={id} />}
				</TabsContent>
			</Tabs>
		</div>
	);
}
