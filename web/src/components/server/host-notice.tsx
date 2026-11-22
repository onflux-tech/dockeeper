import { AlertTriangle } from "lucide-react";
import { Link } from "react-router";
import { Card, CardContent } from "@/components/ui/card";

export function HostNotice() {
	return (
		<Card className="border-warning/30 bg-warning/5 py-0">
			<CardContent className="flex h-12 items-center gap-2 px-6">
				<AlertTriangle className="h-4 w-4 shrink-0 text-warning" />
				<p className="min-w-0 flex-1 text-sm text-foreground">
					No container runtime detected. This server is monitored in host-only mode.
				</p>
				<Link to="/docs" className="text-xs font-medium text-primary hover:underline">
					Why?
				</Link>
			</CardContent>
		</Card>
	);
}
