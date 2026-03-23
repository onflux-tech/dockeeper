import { BarChart3 } from "lucide-react";

export default function MetricsPage() {
	return (
		<div className="flex flex-col gap-4">
			<div className="flex items-center gap-2">
				<BarChart3 className="h-5 w-5 text-muted-foreground" />
				<h1 className="text-2xl font-semibold">Metrics</h1>
			</div>
			<p className="text-muted-foreground">
				System and container metrics dashboards — coming soon.
			</p>
		</div>
	);
}
