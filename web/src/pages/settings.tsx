import { Settings } from "lucide-react";

export default function SettingsPage() {
	return (
		<div className="flex flex-col gap-4">
			<div className="flex items-center gap-2">
				<Settings className="h-5 w-5 text-muted-foreground" />
				<h1 className="text-2xl font-semibold">Settings</h1>
			</div>
			<p className="text-muted-foreground">
				Application settings and configuration — coming soon.
			</p>
		</div>
	);
}
