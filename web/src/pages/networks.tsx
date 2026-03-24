import { useMemo, useState } from "react";
import { Network, Plus, Trash2, Link2, Unlink2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
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
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useCollection } from "@/hooks/use-collection";
import { useRealtime } from "@/hooks/use-realtime";
import { useRemoveNetwork } from "@/hooks/use-networks";
import { CreateNetworkDialog } from "@/components/networks/create-network-dialog";
import { ConnectDialog } from "@/components/networks/connect-dialog";
import type {
	NetworkRecord,
	ServerRecord,
	ContainerRecord,
} from "@/types/pocketbase";

export default function NetworksPage() {
	const [serverFilter, setServerFilter] = useState<string>("all");
	const [search, setSearch] = useState("");
	const [createOpen, setCreateOpen] = useState(false);
	const [removeTarget, setRemoveTarget] = useState<NetworkRecord | null>(null);
	const [connectTarget, setConnectTarget] = useState<{
		network: NetworkRecord;
		mode: "connect" | "disconnect";
	} | null>(null);

	const { data: serversResult } = useCollection("servers", { perPage: 100 });
	const { data: networksResult, isLoading } = useCollection("networks", {
		perPage: 200,
		sort: "name",
		...(serverFilter !== "all"
			? { filter: `server = '${serverFilter}'` }
			: {}),
	});
	const { data: containersResult } = useCollection("containers", {
		perPage: 500,
		...(serverFilter !== "all"
			? { filter: `server = '${serverFilter}'` }
			: {}),
	});

	useRealtime("networks");

	const servers = (serversResult?.items ?? []) as ServerRecord[];
	const networks = useMemo(
		() => (networksResult?.items ?? []) as NetworkRecord[],
		[networksResult?.items],
	);
	const containers = useMemo(
		() => (containersResult?.items ?? []) as ContainerRecord[],
		[containersResult?.items],
	);

	const removeMut = useRemoveNetwork();

	const filtered = useMemo(() => {
		if (!search) return networks;
		const q = search.toLowerCase();
		return networks.filter(
			(n) =>
				n.name.toLowerCase().includes(q) ||
				(n.network_id ?? "").toLowerCase().includes(q) ||
				(n.driver ?? "").toLowerCase().includes(q),
		);
	}, [networks, search]);

	const serverName = (serverId: string) => {
		const s = servers.find((sv) => sv.id === serverId);
		return s?.hostname ?? "—";
	};

	const serverContainers = (serverId: string) =>
		containers.filter((c) => c.server === serverId);

	return (
		<div className="flex flex-col gap-6">
			{/* Header */}
			<div className="flex items-center justify-between">
				<div className="flex items-center gap-2">
					<Network className="h-5 w-5 text-muted-foreground" />
					<h1 className="text-2xl font-semibold">Networks</h1>
					{!isLoading && (
						<span className="text-sm text-muted-foreground">
							({filtered.length})
						</span>
					)}
				</div>
				<Button
					size="sm"
					onClick={() => setCreateOpen(true)}
					disabled={serverFilter === "all"}
				>
					<Plus className="mr-1.5 h-3.5 w-3.5" />
					Create Network
				</Button>
			</div>

			{/* Filters */}
			<div className="flex items-center gap-3">
				<Select value={serverFilter} onValueChange={setServerFilter}>
					<SelectTrigger className="w-48">
						<SelectValue placeholder="All Servers" />
					</SelectTrigger>
					<SelectContent>
						<SelectItem value="all">All Servers</SelectItem>
						{servers.map((s) => (
							<SelectItem key={s.id} value={s.id}>
								{s.hostname}
							</SelectItem>
						))}
					</SelectContent>
				</Select>
				<Input
					placeholder="Search networks..."
					value={search}
					onChange={(e) => setSearch(e.target.value)}
					className="max-w-xs"
				/>
			</div>

			{/* Table */}
			{isLoading ? (
				<div className="space-y-2">
					{[1, 2, 3, 4].map((i) => (
						<Skeleton key={i} className="h-10 w-full" />
					))}
				</div>
			) : filtered.length === 0 ? (
				<p className="text-sm text-muted-foreground">No networks found.</p>
			) : (
				<div className="rounded-md border">
					<Table>
						<TableHeader>
							<TableRow>
								<TableHead>Name</TableHead>
								<TableHead>Driver</TableHead>
								<TableHead>Scope</TableHead>
								<TableHead>Subnet</TableHead>
								<TableHead>Gateway</TableHead>
								<TableHead>Internal</TableHead>
								{serverFilter === "all" && (
									<TableHead>Server</TableHead>
								)}
								<TableHead className="w-12" />
							</TableRow>
						</TableHeader>
						<TableBody>
							{filtered.map((net) => (
								<TableRow key={net.id}>
									<TableCell className="font-medium">{net.name}</TableCell>
									<TableCell>
										<Badge variant="secondary">{net.driver || "—"}</Badge>
									</TableCell>
									<TableCell className="text-xs text-muted-foreground">
										{net.scope || "—"}
									</TableCell>
									<TableCell className="font-mono text-xs">
										{net.subnet || "—"}
									</TableCell>
									<TableCell className="font-mono text-xs">
										{net.gateway || "—"}
									</TableCell>
									<TableCell>
										{net.internal ? (
											<Badge variant="outline">Yes</Badge>
										) : (
											<span className="text-xs text-muted-foreground">No</span>
										)}
									</TableCell>
									{serverFilter === "all" && (
										<TableCell className="text-xs text-muted-foreground">
											{serverName(net.server)}
										</TableCell>
									)}
									<TableCell>
										<DropdownMenu>
											<DropdownMenuTrigger asChild>
												<Button variant="ghost" size="icon" className="h-7 w-7">
													<span className="sr-only">Actions</span>
													<span className="text-xs">...</span>
												</Button>
											</DropdownMenuTrigger>
											<DropdownMenuContent align="end">
												<DropdownMenuItem
													onClick={() =>
														setConnectTarget({
															network: net,
															mode: "connect",
														})
													}
												>
													<Link2 className="mr-2 h-3.5 w-3.5" />
													Connect Container
												</DropdownMenuItem>
												<DropdownMenuItem
													onClick={() =>
														setConnectTarget({
															network: net,
															mode: "disconnect",
														})
													}
												>
													<Unlink2 className="mr-2 h-3.5 w-3.5" />
													Disconnect Container
												</DropdownMenuItem>
												<DropdownMenuItem
													className="text-destructive"
													onClick={() => setRemoveTarget(net)}
												>
													<Trash2 className="mr-2 h-3.5 w-3.5" />
													Remove
												</DropdownMenuItem>
											</DropdownMenuContent>
										</DropdownMenu>
									</TableCell>
								</TableRow>
							))}
						</TableBody>
					</Table>
				</div>
			)}

			{/* Create Dialog */}
			{serverFilter !== "all" && (
				<CreateNetworkDialog
					open={createOpen}
					onOpenChange={setCreateOpen}
					serverID={serverFilter}
				/>
			)}

			{/* Connect/Disconnect Dialog */}
			{connectTarget && (
				<ConnectDialog
					open
					onOpenChange={(open) => {
						if (!open) setConnectTarget(null);
					}}
					networkId={connectTarget.network.id}
					networkName={connectTarget.network.name}
					mode={connectTarget.mode}
					containers={serverContainers(connectTarget.network.server)}
				/>
			)}

			{/* Remove Confirmation */}
			<AlertDialog
				open={!!removeTarget}
				onOpenChange={(open) => {
					if (!open) setRemoveTarget(null);
				}}
			>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>Remove Network</AlertDialogTitle>
						<AlertDialogDescription>
							Are you sure you want to remove the network{" "}
							<span className="font-mono font-semibold">
								{removeTarget?.name}
							</span>
							? This action cannot be undone. Networks with active containers
							cannot be removed.
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel>Cancel</AlertDialogCancel>
						<AlertDialogAction
							onClick={() => {
								if (removeTarget) {
									removeMut.mutate(removeTarget.id);
									setRemoveTarget(null);
								}
							}}
						>
							Remove
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
		</div>
	);
}
