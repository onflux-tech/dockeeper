import { useState } from "react";
import { Globe, ShieldCheck, ShieldOff } from "lucide-react";
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
import { PaginationControls } from "@/components/ui/pagination-controls";
import { useCollection } from "@/hooks/use-collection";
import { useRealtime } from "@/hooks/use-realtime";
import { andFilters, containsAnyFilter, eqFilter } from "@/lib/pocketbase-filters";
import type { ServerRecord, TraefikRouteRecord } from "@/types/pocketbase";

export default function TraefikPage() {
	const [page, setPage] = useState(1);
	const [serverFilter, setServerFilter] = useState<string>("all");
	const [search, setSearch] = useState("");

	const serverId = serverFilter !== "all" ? serverFilter : undefined;

	const { data: serversResult } = useCollection("servers", { perPage: 100 });
	const { data: routesResult, isLoading } = useCollection("traefik_routes", {
		page,
		perPage: 25,
		sort: "name",
		filter: andFilters([
			serverId ? eqFilter("server", serverId) : "",
			containsAnyFilter(["name", "rule", "service"], search),
		]),
	});
	useRealtime("traefik_routes");

	const servers = (serversResult?.items ?? []) as ServerRecord[];
	const routes = (routesResult?.items ?? []) as TraefikRouteRecord[];

	const serverName = (sid: string) => {
		const s = servers.find((sv) => sv.id === sid);
		return s?.hostname ?? sid.slice(0, 8);
	};

	return (
		<div className="flex flex-col gap-6">
			{/* Header */}
			<div className="flex items-center gap-2">
				<Globe className="h-5 w-5 text-muted-foreground" />
				<h1 className="text-2xl font-semibold">Traefik Routes</h1>
				{!isLoading && (
					<span className="text-sm text-muted-foreground">
						({routesResult?.totalItems ?? 0} routes)
					</span>
				)}
			</div>

			{/* Filters */}
			<div className="flex items-center gap-3">
				<Select
					value={serverFilter}
					onValueChange={(value) => {
						setServerFilter(value);
						setPage(1);
					}}
				>
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
					placeholder="Search routes..."
					value={search}
					onChange={(e) => {
						setSearch(e.target.value);
						setPage(1);
					}}
					className="max-w-xs"
				/>
			</div>

			{/* Routes Table */}
			{isLoading ? (
				<div className="space-y-2">
					{[1, 2, 3, 4].map((i) => (
						<Skeleton key={i} className="h-10 w-full" />
					))}
				</div>
			) : routes.length === 0 ? (
				<p className="text-sm text-muted-foreground">
					No Traefik routes found.
					{serverFilter === "all"
						? " Routes appear automatically when Traefik is detected on an agent."
						: " No routes detected on this server."}
				</p>
			) : (
				<div className="rounded-md border">
					<Table>
						<TableHeader>
							<TableRow>
								<TableHead>Name</TableHead>
								<TableHead>Rule</TableHead>
								<TableHead>Service</TableHead>
								<TableHead>Entrypoints</TableHead>
								<TableHead>TLS</TableHead>
								<TableHead>Middlewares</TableHead>
								<TableHead>Provider</TableHead>
								<TableHead>Status</TableHead>
								{serverFilter === "all" && <TableHead>Server</TableHead>}
							</TableRow>
						</TableHeader>
						<TableBody>
							{routes.map((route) => (
								<TableRow key={route.id}>
									<TableCell className="font-medium">{route.name}</TableCell>
									<TableCell className="max-w-xs truncate font-mono text-xs">
										{route.rule}
									</TableCell>
									<TableCell className="text-xs">{route.service}</TableCell>
									<TableCell>
										<div className="flex flex-wrap gap-1">
											{(Array.isArray(route.entrypoints) ? route.entrypoints : []).map((ep) => (
												<Badge key={ep} variant="outline" className="text-[10px]">
													{ep}
												</Badge>
											))}
										</div>
									</TableCell>
									<TableCell>
										{route.tls ? (
											<ShieldCheck className="h-4 w-4 text-green-500" />
										) : (
											<ShieldOff className="h-4 w-4 text-muted-foreground" />
										)}
									</TableCell>
									<TableCell>
										<div className="flex flex-wrap gap-1">
											{(Array.isArray(route.middlewares)
												? (route.middlewares as string[])
												: []
											).map((mw) => (
												<Badge key={mw} variant="secondary" className="text-[10px]">
													{mw}
												</Badge>
											))}
											{(!Array.isArray(route.middlewares) ||
												(route.middlewares as string[]).length === 0) &&
												"—"}
										</div>
									</TableCell>
									<TableCell className="text-xs text-muted-foreground">{route.provider}</TableCell>
									<TableCell>
										<Badge variant={route.status === "enabled" ? "default" : "secondary"}>
											{route.status}
										</Badge>
									</TableCell>
									{serverFilter === "all" && (
										<TableCell className="text-xs text-muted-foreground">
											{serverName(route.server)}
										</TableCell>
									)}
								</TableRow>
							))}
						</TableBody>
					</Table>
				</div>
			)}

			<PaginationControls
				page={page}
				totalPages={routesResult?.totalPages ?? 0}
				totalItems={routesResult?.totalItems ?? 0}
				onPageChange={setPage}
				label="routes"
			/>
		</div>
	);
}
