import { useState } from "react";
import { Link2, Unlink2 } from "lucide-react";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { useConnectContainer, useDisconnectContainer } from "@/hooks/use-networks";
import type { ContainerRecord } from "@/types/pocketbase";

interface ConnectDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	networkId: string;
	networkName: string;
	mode: "connect" | "disconnect";
	containers: ContainerRecord[];
}

export function ConnectDialog({
	open,
	onOpenChange,
	networkId,
	networkName,
	mode,
	containers,
}: ConnectDialogProps) {
	const [selectedContainer, setSelectedContainer] = useState("");
	const connectMut = useConnectContainer();
	const disconnectMut = useDisconnectContainer();

	const isPending = connectMut.isPending || disconnectMut.isPending;

	const handleAction = () => {
		if (!selectedContainer) return;
		const mutation = mode === "connect" ? connectMut : disconnectMut;
		mutation.mutate(
			{ networkId, containerId: selectedContainer },
			{
				onSuccess: () => {
					setSelectedContainer("");
					onOpenChange(false);
				},
			},
		);
	};

	const handleClose = (nextOpen: boolean) => {
		if (isPending) return;
		if (!nextOpen) setSelectedContainer("");
		onOpenChange(nextOpen);
	};

	const isConnect = mode === "connect";
	const Icon = isConnect ? Link2 : Unlink2;

	return (
		<Dialog open={open} onOpenChange={handleClose}>
			<DialogContent>
				<DialogHeader>
					<DialogTitle className="flex items-center gap-2">
						<Icon className="h-5 w-5" />
						{isConnect ? "Connect" : "Disconnect"} Container
					</DialogTitle>
					<DialogDescription>
						{isConnect ? "Connect" : "Disconnect"} a container{" "}
						{isConnect ? "to" : "from"} the network{" "}
						<span className="font-mono font-semibold">{networkName}</span>.
					</DialogDescription>
				</DialogHeader>

				<div className="space-y-1.5">
					<label className="text-xs font-medium">Container</label>
					<Select value={selectedContainer} onValueChange={setSelectedContainer}>
						<SelectTrigger>
							<SelectValue placeholder="Select container" />
						</SelectTrigger>
						<SelectContent>
							{containers.map((c) => (
								<SelectItem key={c.id} value={c.id}>
									{c.name}
								</SelectItem>
							))}
						</SelectContent>
					</Select>
				</div>

				<DialogFooter>
					<Button variant="outline" onClick={() => handleClose(false)}>
						Cancel
					</Button>
					<Button
						onClick={handleAction}
						disabled={!selectedContainer || isPending}
						variant={isConnect ? "default" : "destructive"}
					>
						{isPending
							? isConnect
								? "Connecting..."
								: "Disconnecting..."
							: isConnect
								? "Connect"
								: "Disconnect"}
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}
