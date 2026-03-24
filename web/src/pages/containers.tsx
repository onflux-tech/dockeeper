import { Box, Plus } from "lucide-react";
import { Link } from "react-router";
import { useCollection } from "@/hooks/use-collection";
import { useRealtime } from "@/hooks/use-realtime";
import { ContainerTable } from "@/components/server/container-table";
import { Button } from "@/components/ui/button";

export default function ContainersPage() {
	const { data: containersResult, isLoading } = useCollection("containers", {
		perPage: 500,
		sort: "-created",
	});

	useRealtime("containers");

	const containers = containersResult?.items ?? [];

	return (
		<div className="flex flex-col gap-6">
			<div className="flex items-center justify-between">
				<div className="flex items-center gap-2">
					<Box className="h-5 w-5 text-muted-foreground" />
					<h1 className="text-2xl font-semibold">Containers</h1>
					{!isLoading && (
						<span className="text-sm text-muted-foreground">
							({containers.length})
						</span>
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
			/>
		</div>
	);
}
