import { ShieldAlert } from "lucide-react";
import { useHasCapability } from "@/hooks/use-permissions";

interface RequireCapabilityProps {
	capability: string;
	children: React.ReactNode;
}

export function RequireCapability({ capability, children }: RequireCapabilityProps) {
	const hasAccess = useHasCapability(capability);

	if (!hasAccess) {
		return (
			<div className="flex flex-1 flex-col items-center justify-center gap-4 p-8">
				<ShieldAlert className="h-16 w-16 text-destructive/60" />
				<h2 className="text-xl font-semibold">Access Denied</h2>
				<p className="max-w-md text-center text-muted-foreground">
					You don't have the required permission ({capability}) to access this page. Contact
					your administrator to request access.
				</p>
			</div>
		);
	}

	return <>{children}</>;
}
