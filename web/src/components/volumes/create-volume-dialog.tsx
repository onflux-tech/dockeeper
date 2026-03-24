import { useState } from "react";
import { HardDrive } from "lucide-react";
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
import { useCreateVolume } from "@/hooks/use-volumes";

interface CreateVolumeDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	serverID: string;
}

export function CreateVolumeDialog({ open, onOpenChange, serverID }: CreateVolumeDialogProps) {
	const [name, setName] = useState("");
	const [driver, setDriver] = useState("local");
	const createMut = useCreateVolume();

	const handleCreate = () => {
		if (!name.trim()) return;
		createMut.mutate(
			{ server_id: serverID, name: name.trim(), driver },
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
		setDriver("local");
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
						<HardDrive className="h-5 w-5" />
						Create Volume
					</DialogTitle>
					<DialogDescription>Create a new Docker volume on the selected server.</DialogDescription>
				</DialogHeader>

				<div className="space-y-4">
					<div className="space-y-1.5">
						<label className="text-xs font-medium">Name</label>
						<Input placeholder="my-volume" value={name} onChange={(e) => setName(e.target.value)} />
					</div>

					<div className="space-y-1.5">
						<label className="text-xs font-medium">Driver</label>
						<Select value={driver} onValueChange={setDriver}>
							<SelectTrigger>
								<SelectValue />
							</SelectTrigger>
							<SelectContent>
								<SelectItem value="local">local</SelectItem>
								<SelectItem value="nfs">nfs</SelectItem>
								<SelectItem value="tmpfs">tmpfs</SelectItem>
							</SelectContent>
						</Select>
					</div>
				</div>

				<DialogFooter>
					<Button variant="outline" onClick={() => handleClose(false)}>
						Cancel
					</Button>
					<Button onClick={handleCreate} disabled={!name.trim() || createMut.isPending}>
						{createMut.isPending ? "Creating..." : "Create Volume"}
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}
