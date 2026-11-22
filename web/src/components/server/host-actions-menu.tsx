import { useState } from "react";
import { MoreHorizontal, Play, RotateCw, Square } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuLabel,
	DropdownMenuSeparator,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
	AlertDialog,
	AlertDialogAction,
	AlertDialogCancel,
	AlertDialogContent,
	AlertDialogDescription,
	AlertDialogFooter,
	AlertDialogHeader,
	AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { useHasCapability } from "@/hooks/use-permissions";
import { useHostAction, type HostAction } from "@/hooks/use-host-actions";
import type { HostAppRecord } from "@/types/pocketbase";

interface HostActionsMenuProps {
	serverID: string;
	app: HostAppRecord;
	serverOnline?: boolean;
}

const actions: { value: HostAction; label: string; icon: typeof Play; destructive?: boolean }[] = [
	{ value: "start", label: "Start", icon: Play },
	{ value: "stop", label: "Stop", icon: Square, destructive: true },
	{ value: "restart", label: "Restart", icon: RotateCw },
];

export function HostActionsMenu({ serverID, app, serverOnline = false }: HostActionsMenuProps) {
	const canManage = useHasCapability("host.apps.manage");
	const mutation = useHostAction();
	const [pending, setPending] = useState<HostAction | null>(null);

	const actionSupported = app.source === "systemd" || app.source === "pm2";
	const disabled = !canManage || !serverOnline || !actionSupported;

	const pendingAction = actions.find((action) => action.value === pending);

	const confirm = () => {
		if (!pending) {
			return;
		}
		mutation.mutate({ serverID, source: app.source, name: app.name, action: pending });
		setPending(null);
	};

	return (
		<>
			<DropdownMenu>
				<DropdownMenuTrigger asChild>
					<Button
						variant="ghost"
						size="icon"
						className="h-8 w-8"
						aria-label="Actions"
						disabled={disabled}
						title={
							!canManage
								? "Missing host.apps.manage"
								: !serverOnline
									? "Server is offline"
									: !actionSupported
										? "Actions available for systemd and PM2 apps"
										: "Actions"
						}
					>
						<MoreHorizontal className="h-4 w-4" />
					</Button>
				</DropdownMenuTrigger>
				<DropdownMenuContent align="end">
					<DropdownMenuLabel>{app.name}</DropdownMenuLabel>
					<DropdownMenuSeparator />
					{actions.map((action) => (
						<DropdownMenuItem
							key={action.value}
							className={action.destructive ? "text-destructive focus:text-destructive" : undefined}
							onClick={() => setPending(action.value)}
						>
							<action.icon className="mr-2 h-4 w-4" />
							{action.label}
						</DropdownMenuItem>
					))}
				</DropdownMenuContent>
			</DropdownMenu>

			<AlertDialog open={pending !== null} onOpenChange={(open) => !open && setPending(null)}>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>
							{pendingAction
								? `${pendingAction.label.toLowerCase()} ${app.name}?`
								: "Confirm action"}
						</AlertDialogTitle>
						<AlertDialogDescription>
							{pendingAction?.destructive
								? `This will stop ${app.name} on the host. The service may become unavailable until it is started again.`
								: `Run ${pendingAction?.label.toLowerCase()} on ${app.name} (${app.source}).`}
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel>Cancel</AlertDialogCancel>
						<AlertDialogAction
							onClick={confirm}
							className={
								pendingAction?.destructive
									? "bg-destructive text-destructive-foreground hover:bg-destructive/90"
									: undefined
							}
						>
							{pendingAction?.label}
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
		</>
	);
}
