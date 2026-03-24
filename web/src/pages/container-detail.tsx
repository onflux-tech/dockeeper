import { useState } from "react";
import { useParams, Link, useNavigate } from "react-router";
import {
	ArrowLeft,
	TerminalSquare,
	Play,
	Square,
	RotateCcw,
	Pause,
	CirclePlay,
	Trash2,
	ArrowDownToLine,
	MoreHorizontal,
} from "lucide-react";
import {
	AlertDialog,
	AlertDialogAction,
	AlertDialogCancel,
	AlertDialogContent,
	AlertDialogDescription,
	AlertDialogFooter,
	AlertDialogHeader,
	AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuSeparator,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Skeleton } from "@/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useRecord } from "@/hooks/use-record";
import { useRealtime } from "@/hooks/use-realtime";
import { useContainerMetrics } from "@/hooks/use-container-metrics";
import { useContainerActions } from "@/hooks/use-container-actions";
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
	const navigate = useNavigate();

	const { data: container, isLoading } = useRecord("containers", id);
	const { data: metricsData, isLoading: loadingMetrics } = useContainerMetrics(id);

	useRealtime("containers");
	useRealtime("container_metrics_raw");

	const actions = useContainerActions();
	const [removeOpen, setRemoveOpen] = useState(false);
	const [updateOpen, setUpdateOpen] = useState(false);

	const metrics = metricsData ?? [];
	const statusVariant = statusVariantMap[container?.status ?? ""] ?? "secondary";

	const isRunning = container?.status === "running";
	const isPaused = container?.status === "paused";
	const isStopped = container?.status === "exited" || container?.status === "created";

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
						<h1 className="text-2xl font-semibold">{container?.name || "Container"}</h1>
						{container && <Badge variant={statusVariant}>{container.status}</Badge>}

						{/* Actions */}
						{id && container && (
							<div className="ml-auto flex items-center gap-2">
								{isStopped && (
									<Button variant="outline" size="sm" onClick={() => actions.start.mutate(id)}>
										<Play className="mr-1.5 h-4 w-4" />
										Start
									</Button>
								)}
								{isRunning && (
									<Button variant="outline" size="sm" onClick={() => actions.stop.mutate(id)}>
										<Square className="mr-1.5 h-4 w-4" />
										Stop
									</Button>
								)}
								<Button variant="outline" size="sm" onClick={() => actions.restart.mutate(id)}>
									<RotateCcw className="mr-1.5 h-4 w-4" />
									Restart
								</Button>

								{/* Overflow menu */}
								<DropdownMenu>
									<DropdownMenuTrigger asChild>
										<Button variant="outline" size="icon" className="h-8 w-8">
											<MoreHorizontal className="h-4 w-4" />
										</Button>
									</DropdownMenuTrigger>
									<DropdownMenuContent align="end">
										{isRunning && (
											<DropdownMenuItem onClick={() => actions.pause.mutate(id)}>
												<Pause className="mr-2 h-4 w-4" />
												Pause
											</DropdownMenuItem>
										)}
										{isPaused && (
											<DropdownMenuItem onClick={() => actions.unpause.mutate(id)}>
												<CirclePlay className="mr-2 h-4 w-4" />
												Unpause
											</DropdownMenuItem>
										)}
										<DropdownMenuItem onClick={() => setUpdateOpen(true)}>
											<ArrowDownToLine className="mr-2 h-4 w-4" />
											Update
										</DropdownMenuItem>
										<DropdownMenuSeparator />
										<DropdownMenuItem
											onClick={() => setRemoveOpen(true)}
											className="text-destructive focus:text-destructive"
										>
											<Trash2 className="mr-2 h-4 w-4" />
											Remove
										</DropdownMenuItem>
									</DropdownMenuContent>
								</DropdownMenu>

								{isRunning && (
									<Button variant="outline" size="sm" asChild>
										<Link to={`/containers/${id}/terminal`}>
											<TerminalSquare className="mr-1.5 h-4 w-4" />
											Terminal
										</Link>
									</Button>
								)}
							</div>
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

			{/* Remove confirmation */}
			<AlertDialog open={removeOpen} onOpenChange={setRemoveOpen}>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>Remove container?</AlertDialogTitle>
						<AlertDialogDescription>
							This will permanently remove the container &quot;{container?.name}&quot;. This action
							cannot be undone.
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel>Cancel</AlertDialogCancel>
						<AlertDialogAction
							onClick={() => {
								if (id) {
									actions.remove.mutate(id, {
										onSuccess: () => navigate("/containers"),
									});
								}
								setRemoveOpen(false);
							}}
							className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
						>
							Remove
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>

			{/* Update confirmation */}
			<AlertDialog open={updateOpen} onOpenChange={setUpdateOpen}>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>Update container?</AlertDialogTitle>
						<AlertDialogDescription>
							This will pull the latest version of the image &quot;{container?.image}&quot; and
							recreate the container. The container will be briefly unavailable during the update.
							All configuration (ports, volumes, env vars, networks) will be preserved.
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel>Cancel</AlertDialogCancel>
						<AlertDialogAction
							onClick={() => {
								if (id) actions.update.mutate(id);
								setUpdateOpen(false);
							}}
						>
							Update
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
		</div>
	);
}
