import { Layers } from "lucide-react";

export default function ImagesPage() {
	return (
		<div className="flex flex-col gap-4">
			<div className="flex items-center gap-2">
				<Layers className="h-5 w-5 text-muted-foreground" />
				<h1 className="text-2xl font-semibold">Images</h1>
			</div>
			<p className="text-muted-foreground">
				Browse Docker images across servers — coming soon.
			</p>
		</div>
	);
}
