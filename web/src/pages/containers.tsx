import { Box, Plus } from "lucide-react";
import { Link } from "react-router";
import { PaginationControls } from "@/components/ui/pagination-controls";
import { PerPageSelector } from "@/components/ui/per-page-selector";
import { useCollection } from "@/hooks/use-collection";
import { useListFilters } from "@/hooks/use-list-filters";
import { useRealtime } from "@/hooks/use-realtime";
import { ContainerTable, type SortKey, type SortDir } from "@/components/server/container-table";
import { Button } from "@/components/ui/button";

const SORT_MAP: Record<SortKey, string> = {
	name: "name",
	status: "status",
	image: "image",
	created: "container_created_at",
};

export default function ContainersPage() {
	const { filters, setSearch, setPage, setPerPage, setSort, setCustom, pbOptions } =
		useListFilters({
			defaultSort: "-created",
			searchFields: ["name", "image"],
			customFilters: { status: "status" },
		});

	const { data: containersResult, isLoading } = useCollection("containers", pbOptions);

	useRealtime("containers");

	const containers = containersResult?.items ?? [];
	const totalContainers = containersResult?.totalItems ?? 0;

	// Derive sort key/dir from URL sort param
	const rawSort = filters.sort;
	const isDesc = rawSort.startsWith("-");
	const sortField = isDesc ? rawSort.slice(1) : rawSort;
	const sortKey = (Object.entries(SORT_MAP).find(([, v]) => v === sortField)?.[0] ?? "created") as SortKey;
	const sortDir: SortDir = isDesc ? "desc" : "asc";

	const handleSortChange = (key: SortKey) => {
		const pbField = SORT_MAP[key];
		if (sortKey === key) {
			setSort(sortDir === "asc" ? `-${pbField}` : pbField);
		} else {
			setSort(pbField);
		}
	};

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
				search={filters.search}
				onSearchChange={setSearch}
				statusFilter={filters.custom.status ?? "all"}
				onStatusFilterChange={(value) => setCustom("status", value)}
				sortKey={sortKey}
				sortDir={sortDir}
				onSortChange={handleSortChange}
			/>

			<div className="flex items-center justify-between">
				<PerPageSelector value={filters.perPage} onChange={setPerPage} />
				<PaginationControls
					page={filters.page}
					totalPages={containersResult?.totalPages ?? 0}
					totalItems={totalContainers}
					onPageChange={setPage}
					label="containers"
				/>
			</div>
		</div>
	);
}
