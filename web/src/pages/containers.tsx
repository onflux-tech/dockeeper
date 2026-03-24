import { useState } from "react";
import { Box, Plus } from "lucide-react";
import { Link } from "react-router";
import { PaginationControls } from "@/components/ui/pagination-controls";
import { useCollection } from "@/hooks/use-collection";
import { useRealtime } from "@/hooks/use-realtime";
import { ContainerTable } from "@/components/server/container-table";
import { Button } from "@/components/ui/button";
import { andFilters, containsAnyFilter, eqFilter } from "@/lib/pocketbase-filters";

export default function ContainersPage() {
	const [page, setPage] = useState(1);
	const [search, setSearch] = useState("");
	const [statusFilter, setStatusFilter] = useState("all");
	const { data: containersResult, isLoading } = useCollection("containers", {
		page,
		perPage: 25,
		sort: "-created",
		...(search || statusFilter !== "all"
			? {
					filter: andFilters([
						containsAnyFilter(["name", "image"], search),
						statusFilter !== "all" ? eqFilter("status", statusFilter) : "",
					]),
				}
			: {}),
	});

	useRealtime("containers");

	const containers = containersResult?.items ?? [];
	const totalContainers = containersResult?.totalItems ?? 0;

	return (
		<div className="flex flex-col gap-6">
			<div className="flex items-center justify-between">
				<div className="flex items-center gap-2">
					<Box className="h-5 w-5 text-muted-foreground" />
					<h1 className="text-2xl font-semibold">Containers</h1>
					{!isLoading && containersResult && (
						<span className="text-sm text-muted-foreground">({totalContainers})</span>
					)}
				</div>
				<Button asChild size="sm">
					<Link to="/containers/create">
						<Plus className="mr-1.5 h-4 w-4" />
						Create
					</Link>
				</Button>
			</div>

			<ContainerTable
				containers={containers}
				isLoading={isLoading}
				showServer
				search={search}
				onSearchChange={(value) => {
					setSearch(value);
					setPage(1);
				}}
				statusFilter={statusFilter}
				onStatusFilterChange={(value) => {
					setStatusFilter(value);
					setPage(1);
				}}
			/>

			<PaginationControls
				page={page}
				totalPages={containersResult?.totalPages ?? 0}
				totalItems={totalContainers}
				onPageChange={setPage}
				label="containers"
			/>
		</div>
	);
}
