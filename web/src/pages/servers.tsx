import { useState } from "react";
import { useNavigate } from "react-router";
import { Server, Plus, Search, Trash2, MoreHorizontal, ExternalLink } from "lucide-react";
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
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuSeparator,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useServers, useDeleteServer } from "@/hooks/use-servers";
import { useRealtime } from "@/hooks/use-realtime";
import { relativeTime } from "@/lib/format";
import type { ServerRecord } from "@/types/pocketbase";

function statusVariant(status: string): "default" | "secondary" | "destructive" | "outline" {
	switch (status) {
		case "online":
			return "default";
		case "offline":
			return "destructive";
		case "pending":
			return "outline";
		default:
			return "secondary";
	}
}

export default function ServersPage() {
	const { data: servers, isLoading } = useServers();
	const deleteServer = useDeleteServer();
	const navigate = useNavigate();
	useRealtime("servers");

	const [search, setSearch] = useState("");
	const [deleteTarget, setDeleteTarget] = useState<ServerRecord | null>(null);

	const filtered = servers?.filter(
		(s) =>
			(s.name ?? "").toLowerCase().includes(search.toLowerCase()) ||
			(s.hostname ?? "").toLowerCase().includes(search.toLowerCase()) ||
			(s.ip_address ?? "").includes(search),
	);

	return (
		<div className="flex flex-col gap-4">
			{/* Header */}
			<div className="flex items-center justify-between">
				<div className="flex items-center gap-2">
					<Server className="h-5 w-5 text-muted-foreground" />
					<h1 className="text-2xl font-semibold">Servers</h1>
					{servers && (
						<Badge variant="secondary" className="ml-1">
							{servers.length}
						</Badge>
					)}
				</div>
				<Button onClick={() => navigate("/servers/create")}>
					<Plus className="mr-2 h-4 w-4" />
					Add Server
				</Button>
			</div>

			{/* Search */}
			<div className="relative w-72">
				<Search className="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
				<Input
					placeholder="Search servers…"
					className="pl-8"
					value={search}
					onChange={(e) => setSearch(e.target.value)}
				/>
			</div>

			{/* Table */}
			<div className="rounded-md border">
				<Table>
					<TableHeader>
						<TableRow>
							<TableHead>Name</TableHead>
							<TableHead>Hostname</TableHead>
							<TableHead>IP Address</TableHead>
							<TableHead>OS / Arch</TableHead>
							<TableHead>Agent</TableHead>
							<TableHead>Status</TableHead>
							<TableHead>Last Seen</TableHead>
							<TableHead className="w-10" />
						</TableRow>
					</TableHeader>
					<TableBody>
						{isLoading ? (
							Array.from({ length: 5 }).map((_, i) => (
								<TableRow key={i}>
									{Array.from({ length: 8 }).map((_, j) => (
										<TableCell key={j}>
											<Skeleton className="h-4 w-full" />
										</TableCell>
									))}
								</TableRow>
							))
						) : filtered?.length === 0 ? (
							<TableRow>
								<TableCell colSpan={8} className="h-32 text-center text-muted-foreground">
									{servers?.length === 0
										? 'No servers yet. Click "Add Server" to get started.'
										: "No servers match your search."}
								</TableCell>
							</TableRow>
						) : (
							filtered?.map((server) => (
								<TableRow
									key={server.id}
									className="cursor-pointer"
									onClick={() => navigate(`/servers/${server.id}`)}
								>
									<TableCell className="font-medium">
										{server.name}
										{server.description && (
											<p className="text-xs text-muted-foreground truncate max-w-[200px]">
												{server.description}
											</p>
										)}
									</TableCell>
									<TableCell className="font-mono text-xs">{server.hostname || "—"}</TableCell>
									<TableCell className="font-mono text-xs">{server.ip_address || "—"}</TableCell>
									<TableCell className="text-xs text-muted-foreground">
										{server.os && server.arch ? `${server.os}/${server.arch}` : "—"}
									</TableCell>
									<TableCell className="text-xs text-muted-foreground">
										{server.agent_version || "—"}
									</TableCell>
									<TableCell>
										<Badge variant={statusVariant(server.status)}>{server.status}</Badge>
									</TableCell>
									<TableCell className="text-xs text-muted-foreground whitespace-nowrap">
										{server.last_seen ? relativeTime(server.last_seen) : "never"}
									</TableCell>
									<TableCell>
										<DropdownMenu>
											<DropdownMenuTrigger asChild>
												<Button
													variant="ghost"
													size="sm"
													className="h-8 w-8 p-0"
													onClick={(e) => e.stopPropagation()}
												>
													<MoreHorizontal className="h-4 w-4" />
												</Button>
											</DropdownMenuTrigger>
											<DropdownMenuContent align="end">
												<DropdownMenuItem
													onClick={(e) => {
														e.stopPropagation();
														navigate(`/servers/${server.id}`);
													}}
												>
													<ExternalLink className="mr-2 h-4 w-4" />
													View Details
												</DropdownMenuItem>
												<DropdownMenuSeparator />
												<DropdownMenuItem
													className="text-destructive focus:text-destructive"
													onClick={(e) => {
														e.stopPropagation();
														setDeleteTarget(server);
													}}
												>
													<Trash2 className="mr-2 h-4 w-4" />
													Delete Server
												</DropdownMenuItem>
											</DropdownMenuContent>
										</DropdownMenu>
									</TableCell>
								</TableRow>
							))
						)}
					</TableBody>
				</Table>
			</div>

			{/* Delete confirmation dialog */}
			<AlertDialog
				open={deleteTarget !== null}
				onOpenChange={(open) => {
					if (!open) setDeleteTarget(null);
				}}
			>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>Delete server?</AlertDialogTitle>
						<AlertDialogDescription>
							This will permanently remove the server &quot;
							{deleteTarget?.name ?? deleteTarget?.hostname}&quot; and all associated data
							(containers, metrics, images, volumes, networks).
							{deleteTarget?.status === "online" &&
								" The agent will be uninstalled from the remote machine."}
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel>Cancel</AlertDialogCancel>
						<AlertDialogAction
							onClick={() => {
								if (deleteTarget) {
									deleteServer.mutate(deleteTarget.id);
								}
								setDeleteTarget(null);
							}}
							className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
						>
							Delete
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
		</div>
	);
}
