import { useMemo, useState } from "react";
import { Rocket, Plus } from "lucide-react";
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
import { PaginationControls } from "@/components/ui/pagination-controls";
import { PerPageSelector } from "@/components/ui/per-page-selector";
import { useCollection } from "@/hooks/use-collection";
import { useListFilters } from "@/hooks/use-list-filters";
import { useRealtime } from "@/hooks/use-realtime";
import { DeployDialog } from "@/components/compose/deploy-dialog";
import { StackDetail } from "@/components/compose/stack-detail";
import type { ComposeStackRecord, ServerRecord } from "@/types/pocketbase";

const statusVariant: Record<string, "default" | "secondary" | "destructive"> = {
	running: "default",
	stopped: "secondary",
	partial: "destructive",
};

export default function ComposePage() {
	const { filters, setSearch, setPage, setPerPage, setCustom, pbOptions } = useListFilters({
		defaultSort: "-created",
		searchFields: ["name"],
		customFilters: { server: "server" },
	});
	const [deployOpen, setDeployOpen] = useState(false);
	const [selectedId, setSelectedId] = useState<string | null>(null);

	const serverFilter = filters.custom.server ?? "all";

	const { data: serversResult } = useCollection("servers", { perPage: 100 });
	const { data: stacksResult, isLoading } = useCollection("compose_stacks", pbOptions);

	useRealtime("compose_stacks");

	const servers = (serversResult?.items ?? []) as ServerRecord[];
	const stacks = useMemo(
		() => (stacksResult?.items ?? []) as ComposeStackRecord[],
		[stacksResult?.items],
	);

	const selectedStack = selectedId ? (stacks.find((s) => s.id === selectedId) ?? null) : null;

	// If the selected stack was deleted, deselect
	if (selectedId && !selectedStack) {
		// Defer to avoid setState in render
		queueMicrotask(() => setSelectedId(null));
	}

	const serverName = (serverId: string) => {
		const s = servers.find((sv) => sv.id === serverId);
		return s?.hostname ?? "—";
	};

	return (
		<div className="flex flex-col gap-6">
			{/* Header */}
			<div className="flex items-center justify-between">
				<div className="flex items-center gap-2">
					<Rocket className="h-5 w-5 text-muted-foreground" />
					<h1 className="text-2xl font-semibold">Compose Stacks</h1>
					{!isLoading && (
						<span className="text-sm text-muted-foreground">({stacksResult?.totalItems ?? 0})</span>
					)}
				</div>
				<Button size="sm" onClick={() => setDeployOpen(true)}>
					<Plus className="mr-1.5 h-3.5 w-3.5" />
					Deploy Stack
				</Button>
			</div>

			{/* Filters */}
			<div className="flex items-center gap-3">
				<Select
					value={serverFilter}
					onValueChange={(value) => {
						setCustom("server", value !== "all" ? value : "");
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
					placeholder="Search stacks..."
					value={filters.search}
					onChange={(e) => setSearch(e.target.value)}
					className="max-w-xs"
				/>
			</div>

			{/* Table */}
			{isLoading ? (
				<div className="space-y-2">
					{[1, 2, 3].map((i) => (
						<Skeleton key={i} className="h-10 w-full" />
					))}
				</div>
			) : stacks.length === 0 ? (
				<p className="text-sm text-muted-foreground">No stacks found.</p>
			) : (
				<div className="rounded-md border">
					<Table>
						<TableHeader>
							<TableRow>
								<TableHead>Name</TableHead>
								<TableHead>Server</TableHead>
								<TableHead>Status</TableHead>
								<TableHead className="text-right">Services</TableHead>
								<TableHead>Deployed</TableHead>
							</TableRow>
						</TableHeader>
						<TableBody>
							{stacks.map((stack) => (
								<TableRow
									key={stack.id}
									className={`cursor-pointer ${selectedId === stack.id ? "bg-muted/50" : ""}`}
									onClick={() => setSelectedId(selectedId === stack.id ? null : stack.id)}
								>
									<TableCell className="font-medium">{stack.name}</TableCell>
									<TableCell className="text-xs text-muted-foreground">
										{serverName(stack.server)}
									</TableCell>
									<TableCell>
										<Badge variant={statusVariant[stack.status] ?? "secondary"}>
											{stack.status}
										</Badge>
									</TableCell>
									<TableCell className="text-right text-xs">
										{stack.services_count || "—"}
									</TableCell>
									<TableCell className="text-xs text-muted-foreground">
										{stack.deployed_at ? new Date(stack.deployed_at).toLocaleDateString() : "—"}
									</TableCell>
								</TableRow>
							))}
						</TableBody>
					</Table>
				</div>
			)}

			<PaginationControls
				page={filters.page}
				totalPages={stacksResult?.totalPages ?? 0}
				totalItems={stacksResult?.totalItems ?? 0}
				onPageChange={setPage}
				label="stacks"
			/>
			<PerPageSelector value={filters.perPage} onChange={setPerPage} />

			{/* Stack Detail */}
			{selectedStack && <StackDetail stack={selectedStack} onDeleted={() => setSelectedId(null)} />}

			{/* Deploy Dialog */}
			<DeployDialog open={deployOpen} onOpenChange={setDeployOpen} servers={servers} />
		</div>
	);
}
