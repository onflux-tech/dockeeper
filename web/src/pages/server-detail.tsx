import { useParams } from "react-router";
import { Server } from "lucide-react";

export default function ServerDetailPage() {
	const { id } = useParams();

	return (
		<div className="flex flex-col gap-4">
			<div className="flex items-center gap-2">
				<Server className="h-5 w-5 text-muted-foreground" />
				<h1 className="text-2xl font-semibold">Server Detail</h1>
				<span className="text-sm text-muted-foreground">({id})</span>
			</div>
			<p className="text-muted-foreground">
				Server details and metrics — coming soon.
			</p>
		</div>
	);
}
