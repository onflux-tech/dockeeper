import { Archive } from "lucide-react";

export default function BackupsPage() {
	return (
		<div className="flex flex-col gap-4">
			<div className="flex items-center gap-2">
				<Archive className="h-5 w-5 text-muted-foreground" />
				<h1 className="text-2xl font-semibold">Backups</h1>
			</div>
			<p className="text-muted-foreground">
				Manage backup policies and recovery — coming soon.
			</p>
		</div>
	);
}
