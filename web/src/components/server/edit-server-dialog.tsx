import { useState } from "react";
import { Server } from "lucide-react";
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
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useUpdateServer } from "@/hooks/use-servers";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import type { ServerRecord } from "@/types/pocketbase";

interface EditServerDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	server: ServerRecord;
}

export function EditServerDialog({ open, onOpenChange, server }: EditServerDialogProps) {
	const [name, setName] = useState(server.name ?? "");
	const [description, setDescription] = useState(server.description ?? "");
	const [viewScope, setViewScope] = useState<ServerRecord["view_scope"]>(
		server.view_scope || "auto",
	);
	const [nameError, setNameError] = useState("");

	const updateMut = useUpdateServer();

	const handleSubmit = (e: React.FormEvent) => {
		e.preventDefault();

		const trimmed = name.trim();
		if (!trimmed) {
			setNameError("Name is required");
			return;
		}
		setNameError("");

		updateMut.mutate(
			{ id: server.id, name: trimmed, description, view_scope: viewScope },
			{
				onSuccess: () => onOpenChange(false),
			},
		);
	};

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="sm:max-w-md">
				<DialogHeader>
					<DialogTitle className="flex items-center gap-2">
						<Server className="h-4 w-4" />
						Edit Server
					</DialogTitle>
					<DialogDescription>Update the server name, description and view scope.</DialogDescription>
				</DialogHeader>

				<form onSubmit={handleSubmit} className="space-y-4">
					<div className="space-y-2">
						<Label htmlFor="server-name">Name *</Label>
						<Input
							id="server-name"
							value={name}
							onChange={(e) => {
								setName(e.target.value);
								if (nameError) setNameError("");
							}}
							placeholder="Server name"
						/>
						{nameError && <p className="text-xs text-destructive">{nameError}</p>}
					</div>

					<div className="space-y-2">
						<Label htmlFor="server-description">Description</Label>
						<Textarea
							id="server-description"
							value={description}
							onChange={(e) => setDescription(e.target.value)}
							placeholder="Optional description"
							rows={3}
						/>
					</div>

					<div className="space-y-2">
						<Label>View scope</Label>
						<Select
							value={viewScope}
							onValueChange={(value) => setViewScope(value as ServerRecord["view_scope"])}
						>
							<SelectTrigger>
								<SelectValue />
							</SelectTrigger>
							<SelectContent>
								<SelectItem value="auto">Auto (based on the runtime)</SelectItem>
								<SelectItem value="server">Server apps only</SelectItem>
								<SelectItem value="docker">Docker only</SelectItem>
								<SelectItem value="both">Both</SelectItem>
							</SelectContent>
						</Select>
					</div>

					<DialogFooter>
						<Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
							Cancel
						</Button>
						<Button type="submit" disabled={updateMut.isPending}>
							{updateMut.isPending ? "Saving…" : "Save"}
						</Button>
					</DialogFooter>
				</form>
			</DialogContent>
		</Dialog>
	);
}
