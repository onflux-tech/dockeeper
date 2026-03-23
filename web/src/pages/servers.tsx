import { Server } from "lucide-react";

export default function ServersPage() {
	return (
		<div className="flex flex-col gap-4">
			<div className="flex items-center gap-2">
				<Server className="h-5 w-5 text-muted-foreground" />
				<h1 className="text-2xl font-semibold">Servers</h1>
			</div>
			<p className="text-muted-foreground">
				Manage your monitored servers — coming soon.
			</p>
		</div>
	);
}
