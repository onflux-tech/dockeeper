import { useState } from "react";
import { Network } from "lucide-react";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { useCreateNetwork } from "@/hooks/use-networks";

interface CreateNetworkDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	serverID: string;
}

export function CreateNetworkDialog({ open, onOpenChange, serverID }: CreateNetworkDialogProps) {
	const [name, setName] = useState("");
	const [driver, setDriver] = useState("bridge");
	const [subnet, setSubnet] = useState("");
	const [gateway, setGateway] = useState("");
	const [internal, setInternal] = useState(false);
	const createMut = useCreateNetwork();

	const handleCreate = () => {
		if (!name.trim()) return;
		createMut.mutate(
			{
				server_id: serverID,
				name: name.trim(),
				driver,
				...(subnet ? { subnet } : {}),
				...(gateway ? { gateway } : {}),
				internal,
			},
			{
				onSuccess: () => {
					resetForm();
					onOpenChange(false);
				},
			},
		);
	};

	const resetForm = () => {
		setName("");
		setDriver("bridge");
		setSubnet("");
		setGateway("");
		setInternal(false);
	};

	const handleClose = (nextOpen: boolean) => {
		if (createMut.isPending) return;
		if (!nextOpen) resetForm();
		onOpenChange(nextOpen);
	};

	return (
		<Dialog open={open} onOpenChange={handleClose}>
			<DialogContent>
				<DialogHeader>
					<DialogTitle className="flex items-center gap-2">
						<Network className="h-5 w-5" />
						Create Network
					</DialogTitle>
					<DialogDescription>Create a new Docker network on the selected server.</DialogDescription>
				</DialogHeader>

				<div className="space-y-4">
					<div className="space-y-1.5">
						<label className="text-xs font-medium">Name</label>
						<Input
							placeholder="my-network"
							value={name}
							onChange={(e) => setName(e.target.value)}
						/>
					</div>

					<div className="grid grid-cols-2 gap-3">
						<div className="space-y-1.5">
							<label className="text-xs font-medium">Driver</label>
							<Select value={driver} onValueChange={setDriver}>
								<SelectTrigger>
									<SelectValue />
								</SelectTrigger>
								<SelectContent>
									<SelectItem value="bridge">bridge</SelectItem>
									<SelectItem value="overlay">overlay</SelectItem>
									<SelectItem value="macvlan">macvlan</SelectItem>
									<SelectItem value="ipvlan">ipvlan</SelectItem>
									<SelectItem value="none">none</SelectItem>
								</SelectContent>
							</Select>
						</div>
						<div className="space-y-1.5">
							<label className="text-xs font-medium">Internal</label>
							<Select
								value={internal ? "yes" : "no"}
								onValueChange={(v) => setInternal(v === "yes")}
							>
								<SelectTrigger>
									<SelectValue />
								</SelectTrigger>
								<SelectContent>
									<SelectItem value="no">No</SelectItem>
									<SelectItem value="yes">Yes</SelectItem>
								</SelectContent>
							</Select>
						</div>
					</div>

					<div className="grid grid-cols-2 gap-3">
						<div className="space-y-1.5">
							<label className="text-xs font-medium">Subnet (optional)</label>
							<Input
								placeholder="172.20.0.0/16"
								value={subnet}
								onChange={(e) => setSubnet(e.target.value)}
							/>
						</div>
						<div className="space-y-1.5">
							<label className="text-xs font-medium">Gateway (optional)</label>
							<Input
								placeholder="172.20.0.1"
								value={gateway}
								onChange={(e) => setGateway(e.target.value)}
							/>
						</div>
					</div>
				</div>

				<DialogFooter>
					<Button variant="outline" onClick={() => handleClose(false)}>
						Cancel
					</Button>
					<Button onClick={handleCreate} disabled={!name.trim() || createMut.isPending}>
						{createMut.isPending ? "Creating..." : "Create Network"}
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}
