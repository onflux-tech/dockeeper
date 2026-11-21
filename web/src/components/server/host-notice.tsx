import { AlertTriangle } from "lucide-react";
import { Link } from "react-router";
import { Card, CardContent } from "@/components/ui/card";

export function HostNotice() {
	return (
		<Card className="border-warning/30 bg-warning/5">
			<CardContent className="flex items-start gap-3 py-4">
				<AlertTriangle className="mt-0.5 h-5 w-5 shrink-0 text-warning" />
				<div className="flex flex-col gap-1">
					<span className="text-sm font-medium">Host-only server</span>
					<p className="text-sm text-muted-foreground">
						No container runtime detected. Dockeeper monitors systemd units, PM2 processes and
						listening ports directly on this host.{" "}
						<Link to="/docs" className="underline underline-offset-2 hover:text-foreground">
							Why?
						</Link>
					</p>
				</div>
			</CardContent>
		</Card>
	);
}
