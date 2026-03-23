import { Bell } from "lucide-react";

export default function AlertsPage() {
	return (
		<div className="flex flex-col gap-4">
			<div className="flex items-center gap-2">
				<Bell className="h-5 w-5 text-muted-foreground" />
				<h1 className="text-2xl font-semibold">Alerts</h1>
			</div>
			<p className="text-muted-foreground">
				Configure alert rules and notification channels — coming soon.
			</p>
		</div>
	);
}
