import { useState } from "react";
import { Rocket } from "lucide-react";
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
import { Textarea } from "@/components/ui/textarea";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { useCreateStack } from "@/hooks/use-compose";
import type { ServerRecord } from "@/types/pocketbase";

interface DeployDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	servers: ServerRecord[];
}

const YAML_PLACEHOLDER = `version: "3.8"
services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"`;

export function DeployDialog({ open, onOpenChange, servers }: DeployDialogProps) {
	const [serverID, setServerID] = useState("");
	const [name, setName] = useState("");
	const [yaml, setYaml] = useState("");
	const createMut = useCreateStack();

	const handleDeploy = () => {
		if (!serverID || !name.trim() || !yaml.trim()) return;
		createMut.mutate(
			{ server_id: serverID, name: name.trim(), yaml_content: yaml },
			{
				onSuccess: () => {
					setServerID("");
					setName("");
					setYaml("");
					onOpenChange(false);
				},
			},
		);
	};

	const handleClose = (nextOpen: boolean) => {
		if (createMut.isPending) return;
		if (!nextOpen) {
			setServerID("");
			setName("");
			setYaml("");
		}
		onOpenChange(nextOpen);
	};

	const onlineServers = servers.filter((s) => s.status === "online");

	return (
		<Dialog open={open} onOpenChange={handleClose}>
			<DialogContent className="max-w-2xl">
				<DialogHeader>
					<DialogTitle className="flex items-center gap-2">
						<Rocket className="h-5 w-5" />
						Deploy Stack
					</DialogTitle>
					<DialogDescription>
						Create a new Docker Compose stack. Paste your docker-compose.yml content below.
					</DialogDescription>
				</DialogHeader>

				<div className="space-y-4">
					<div className="grid grid-cols-2 gap-3">
						<div className="space-y-1.5">
							<label className="text-xs font-medium">Server</label>
							<Select value={serverID} onValueChange={setServerID}>
								<SelectTrigger>
									<SelectValue placeholder="Select server" />
								</SelectTrigger>
								<SelectContent>
									{onlineServers.map((s) => (
										<SelectItem key={s.id} value={s.id}>
											{s.hostname}
										</SelectItem>
									))}
								</SelectContent>
							</Select>
						</div>
						<div className="space-y-1.5">
							<label className="text-xs font-medium">Stack Name</label>
							<Input
								placeholder="my-app"
								value={name}
								onChange={(e) => setName(e.target.value)}
							/>
						</div>
					</div>

					<div className="space-y-1.5">
						<label className="text-xs font-medium">Compose YAML</label>
						<Textarea
							placeholder={YAML_PLACEHOLDER}
							value={yaml}
							onChange={(e) => setYaml(e.target.value)}
							className="min-h-64 font-mono text-xs leading-relaxed"
							spellCheck={false}
						/>
					</div>
				</div>

				<DialogFooter>
					<Button variant="outline" onClick={() => handleClose(false)}>
						Cancel
					</Button>
					<Button
						onClick={handleDeploy}
						disabled={!serverID || !name.trim() || !yaml.trim() || createMut.isPending}
					>
						{createMut.isPending ? "Creating..." : "Create Stack"}
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}
