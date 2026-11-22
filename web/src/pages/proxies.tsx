import { useMemo, useState } from "react";
import { Globe, Search } from "lucide-react";
import { Badge } from "@/components/ui/badge";
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
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { PaginationControls } from "@/components/ui/pagination-controls";
import { useCollection } from "@/hooks/use-collection";
import { useProxyInstances, useProxyRoutes } from "@/hooks/use-proxies";
import { relativeTime } from "@/lib/format";
import type { ProxyInstanceRecord, ServerRecord } from "@/types/pocketbase";

const PROXY_KINDS = ["traefik", "nginx", "caddy", "haproxy", "envoy"] as const;

const statusVariant: Record<string, "default" | "secondary" | "destructive"> = {
	healthy: "default",
	degraded: "destructive",
	unknown: "secondary",
};

export default function ProxiesPage() {
	const [serverFilter, setServerFilter] = useState("all");
	const [kindFilter, setKindFilter] = useState("all");
	const [search, setSearch] = useState("");
	const [instancesPage, setInstancesPage] = useState(1);
	const [routesPage, setRoutesPage] = useState(1);
	const [selected, setSelected] = useState<ProxyInstanceRecord | null>(null);

	const { data: serversResult } = useCollection("servers", { perPage: 100 });
	const instances = useProxyInstances({
		page: instancesPage,
		perPage: 25,
		server: serverFilter === "all" ? undefined : serverFilter,
		kind: kindFilter === "all" ? undefined : kindFilter,
		search,
	});
	const routes = useProxyRoutes({
		page: routesPage,
		perPage: 25,
		server: serverFilter === "all" ? undefined : serverFilter,
		kind: kindFilter === "all" ? undefined : kindFilter,
		search,
	});

	const servers = useMemo(
		() => (serversResult?.items ?? []) as ServerRecord[],
		[serversResult?.items],
	);
	const serverName = (serverID: string) => {
		const server = servers.find((item) => item.id === serverID);
		return server?.name || server?.hostname || "—";
	};

	const instanceItems = instances.data?.items ?? [];
	const routeItems = routes.data?.items ?? [];
	const totalInstances = instances.data?.totalItems ?? 0;

	return (
		<div className="flex flex-col gap-6">
			<div className="flex items-center gap-2">
				<Globe className="h-5 w-5 text-muted-foreground" />
				<h1 className="text-2xl font-semibold">Proxies</h1>
				<Badge variant="secondary" className="ml-1">
					{totalInstances}
				</Badge>
			</div>

			<div className="flex flex-wrap items-center gap-3">
				<Select
					value={serverFilter}
					onValueChange={(value) => {
						setServerFilter(value);
						setInstancesPage(1);
						setRoutesPage(1);
					}}
				>
					<SelectTrigger className="w-48">
						<SelectValue placeholder="All servers" />
					</SelectTrigger>
					<SelectContent>
						<SelectItem value="all">All servers</SelectItem>
						{servers.map((server) => (
							<SelectItem key={server.id} value={server.id}>
								{server.name || server.hostname}
							</SelectItem>
						))}
					</SelectContent>
				</Select>

				<Select
					value={kindFilter}
					onValueChange={(value) => {
						setKindFilter(value);
						setInstancesPage(1);
						setRoutesPage(1);
					}}
				>
					<SelectTrigger className="w-40">
						<SelectValue placeholder="All kinds" />
					</SelectTrigger>
					<SelectContent>
						<SelectItem value="all">All kinds</SelectItem>
						{PROXY_KINDS.map((kind) => (
							<SelectItem key={kind} value={kind}>
								{kind}
							</SelectItem>
						))}
					</SelectContent>
				</Select>

				<div className="relative w-64">
					<Search className="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
					<Input
						placeholder="Search proxies..."
						value={search}
						onChange={(event) => {
							setSearch(event.target.value);
							setInstancesPage(1);
							setRoutesPage(1);
						}}
						className="pl-8"
					/>
				</div>
			</div>

			<div>
				<h2 className="mb-3 text-lg font-medium">Instances</h2>
				{instances.isLoading ? (
					<div className="space-y-2">
						{[1, 2, 3].map((index) => (
							<Skeleton key={index} className="h-10 w-full" />
						))}
					</div>
				) : instanceItems.length === 0 ? (
					<div className="rounded-md border border-dashed p-8 text-center">
						<p className="text-sm font-medium">No proxy detected</p>
						<p className="mt-1 text-sm text-muted-foreground">
							Supported kinds: Traefik, nginx, Caddy, HAProxy and Envoy.
						</p>
					</div>
				) : (
					<>
						<div className="rounded-md border">
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>Kind</TableHead>
										<TableHead>Server</TableHead>
										<TableHead>Source / App</TableHead>
										<TableHead>Version</TableHead>
										<TableHead>Status</TableHead>
										<TableHead>Last seen</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{instanceItems.map((instance) => (
										<TableRow
											key={instance.id}
											className="cursor-pointer hover:bg-muted/30"
											onClick={() => setSelected(instance)}
										>
											<TableCell className="font-medium">{instance.kind}</TableCell>
											<TableCell className="text-xs">{serverName(instance.server)}</TableCell>
											<TableCell className="text-xs">
												<div className="flex flex-col">
													<span>{instance.app || "—"}</span>
													<span className="text-[10px] text-muted-foreground">
														{instance.source}
													</span>
												</div>
											</TableCell>
											<TableCell className="font-mono text-xs">{instance.version || "—"}</TableCell>
											<TableCell>
												<Badge variant={statusVariant[instance.status] ?? "secondary"}>
													{instance.status}
												</Badge>
											</TableCell>
											<TableCell className="text-xs text-muted-foreground">
												{relativeTime(instance.last_seen)}
											</TableCell>
										</TableRow>
									))}
								</TableBody>
							</Table>
						</div>
						<PaginationControls
							page={instancesPage}
							totalPages={instances.data?.totalPages ?? 0}
							totalItems={instances.data?.totalItems ?? 0}
							onPageChange={setInstancesPage}
							label="instances"
						/>
					</>
				)}
			</div>

			<div>
				<h2 className="mb-3 text-lg font-medium">Routes</h2>
				{routes.isLoading ? (
					<div className="space-y-2">
						{[1, 2, 3].map((index) => (
							<Skeleton key={index} className="h-10 w-full" />
						))}
					</div>
				) : routeItems.length === 0 ? (
					<p className="text-sm text-muted-foreground">No proxy routes detected.</p>
				) : (
					<>
						<div className="rounded-md border">
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>Host / Path</TableHead>
										<TableHead>Upstreams</TableHead>
										<TableHead>TLS</TableHead>
										<TableHead>Provider</TableHead>
										<TableHead>Status</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{routeItems.map((route) => (
										<TableRow key={route.id}>
											<TableCell className="text-xs">
												<div className="flex flex-col">
													<span className="font-medium">
														{route.hosts?.length ? route.hosts.join(", ") : route.name || "—"}
													</span>
													<span className="font-mono text-[10px] text-muted-foreground">
														{route.paths?.length ? route.paths.join(", ") : "/"}
													</span>
												</div>
											</TableCell>
											<TableCell className="max-w-[280px] truncate font-mono text-xs">
												{route.upstreams?.length ? route.upstreams.join(", ") : "—"}
											</TableCell>
											<TableCell>
												<Badge variant={route.tls ? "default" : "outline"}>
													{route.tls ? "TLS" : "HTTP"}
												</Badge>
											</TableCell>
											<TableCell className="text-xs">{route.provider || "—"}</TableCell>
											<TableCell>
												<Badge variant={statusVariant[route.status] ?? "secondary"}>
													{route.status}
												</Badge>
											</TableCell>
										</TableRow>
									))}
								</TableBody>
							</Table>
						</div>
						<PaginationControls
							page={routesPage}
							totalPages={routes.data?.totalPages ?? 0}
							totalItems={routes.data?.totalItems ?? 0}
							onPageChange={setRoutesPage}
							label="routes"
						/>
					</>
				)}
			</div>

			<Dialog open={selected !== null} onOpenChange={(open) => !open && setSelected(null)}>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>
							{selected?.kind} on {selected ? serverName(selected.server) : ""}
						</DialogTitle>
					</DialogHeader>
					{selected ? (
						<dl className="grid grid-cols-2 gap-3 text-sm">
							<div>
								<dt className="text-xs text-muted-foreground">Source</dt>
								<dd>{selected.source}</dd>
							</div>
							<div>
								<dt className="text-xs text-muted-foreground">App</dt>
								<dd>{selected.app || "—"}</dd>
							</div>
							<div>
								<dt className="text-xs text-muted-foreground">Version</dt>
								<dd className="font-mono text-xs">{selected.version || "—"}</dd>
							</div>
							<div>
								<dt className="text-xs text-muted-foreground">Status</dt>
								<dd>{selected.status}</dd>
							</div>
							<div className="col-span-2">
								<dt className="text-xs text-muted-foreground">Config path</dt>
								<dd className="font-mono text-xs">{selected.config_path || "—"}</dd>
							</div>
							<div className="col-span-2">
								<dt className="text-xs text-muted-foreground">Admin URL</dt>
								<dd className="font-mono text-xs">{selected.admin_url || "—"}</dd>
							</div>
						</dl>
					) : null}
				</DialogContent>
			</Dialog>
		</div>
	);
}
