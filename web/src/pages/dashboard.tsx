import { Home } from "lucide-react";

export default function DashboardPage() {
	return (
		<div className="flex flex-col gap-4">
			<div className="flex items-center gap-2">
				<Home className="h-5 w-5 text-muted-foreground" />
				<h1 className="text-2xl font-semibold">Dashboard</h1>
			</div>
			<p className="text-muted-foreground">
				Overview of your infrastructure — coming soon.
			</p>
		</div>
	);
}
