import { Container } from "lucide-react";

export default function ContainersPage() {
	return (
		<div className="flex flex-col gap-4">
			<div className="flex items-center gap-2">
				<Container className="h-5 w-5 text-muted-foreground" />
				<h1 className="text-2xl font-semibold">Containers</h1>
			</div>
			<p className="text-muted-foreground">
				View and manage containers across all servers — coming soon.
			</p>
		</div>
	);
}
