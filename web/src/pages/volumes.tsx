import { HardDrive } from "lucide-react";

export default function VolumesPage() {
	return (
		<div className="flex flex-col gap-4">
			<div className="flex items-center gap-2">
				<HardDrive className="h-5 w-5 text-muted-foreground" />
				<h1 className="text-2xl font-semibold">Volumes</h1>
			</div>
			<p className="text-muted-foreground">
				Manage Docker volumes across servers — coming soon.
			</p>
		</div>
	);
}
