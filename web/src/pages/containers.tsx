import { useSearchParams } from "react-router";
import { Box, Plus, Search } from "lucide-react";
import { Link } from "react-router";
import { PaginationControls } from "@/components/ui/pagination-controls";
import { PerPageSelector } from "@/components/ui/per-page-selector";
import { useCollection } from "@/hooks/use-collection";
import { useListFilters } from "@/hooks/use-list-filters";
import { useRealtime } from "@/hooks/use-realtime";
import { ContainerTable, type SortKey, type SortDir } from "@/components/server/container-table";
import { statusOptions } from "@/lib/container-constants";
import { ServerFilterSelect } from "@/components/server-filter-select";
import { TagFilterSelect } from "@/components/tag-filter-select";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { containsFilter, andFilters } from "@/lib/pocketbase-filters";

const SORT_MAP: Record<SortKey, string> = {
	name: "name",
	status: "status",
	image: "image",
	created: "container_created_at",
	uptime: "started_at",
	cpu: "current_cpu",
	ram: "current_ram",
};

export default function ContainersPage() {
	const { filters, setSearch, setPage, setPerPage, setSort, setCustom, pbOptions } =
		useListFilters({
			defaultSort: "-created",
			searchFields: ["name", "image"],
			customFilters: { status: "status", server: "server" },
		});

	// Tag filter managed separately (needs `~` contains, not `=` eq).
	const [searchParams, setSearchParams] = useSearchParams();
	const tagValue = searchParams.get("tag") ?? "all";

	function setTagFilter(value: string) {
		setSearchParams(
			(prev) => {
				const next = new URLSearchParams(prev);
				if (value === "all") {
					next.delete("tag");
				} else {
					next.set("tag", value);
				}
				next.delete("page");
				return next;
			},
			{ replace: true },
		);
	}

	const tagFilter = tagValue !== "all" ? containsFilter("tags", tagValue) : "";
	const mergedFilter = andFilters([pbOptions.filter, tagFilter]);
	const finalOptions = { ...pbOptions, filter: mergedFilter };

	const { data: containersResult, isLoading } = useCollection("containers", finalOptions);

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
						<Badge variant="secondary" className="ml-1">{totalContainers}</Badge>
					)}
				</div>
				<Button asChild size="sm">
					<Link to="/containers/create">
						<Plus className="mr-1.5 h-4 w-4" />
						Create
					</Link>
				</Button>
			</div>

			<div className="flex items-center gap-3">
				<ServerFilterSelect
					value={filters.custom.server ?? "all"}
					onValueChange={(v) => setCustom("server", v)}
				/>
				<TagFilterSelect
					resourceType="container"
					value={tagValue}
					onValueChange={setTagFilter}
				/>
				<Select
					value={filters.custom.status ?? "all"}
					onValueChange={(value) => setCustom("status", value)}
				>
					<SelectTrigger className="w-36">
						<SelectValue placeholder="Status" />
					</SelectTrigger>
					<SelectContent>
						{statusOptions.map((s) => (
							<SelectItem key={s} value={s}>
								{s === "all" ? "All statuses" : s}
							</SelectItem>
						))}
					</SelectContent>
				</Select>
				<div className="relative w-64">
					<Search className="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
					<Input
						placeholder="Search containers..."
						className="pl-8"
						value={filters.search}
						onChange={(e) => setSearch(e.target.value)}
					/>
				</div>
			</div>

			<ContainerTable
				containers={containers}
				isLoading={isLoading}
				showServer
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
