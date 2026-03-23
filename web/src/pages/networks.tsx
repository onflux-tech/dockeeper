import { Network } from "lucide-react";

export default function NetworksPage() {
	return (
		<div className="flex flex-col gap-4">
			<div className="flex items-center gap-2">
				<Network className="h-5 w-5 text-muted-foreground" />
				<h1 className="text-2xl font-semibold">Networks</h1>
			</div>
			<p className="text-muted-foreground">
				View Docker networks across servers — coming soon.
			</p>
		</div>
	);
}
