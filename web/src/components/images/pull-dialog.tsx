import { useState } from "react";
import { Download } from "lucide-react";
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
import { Progress } from "@/components/ui/progress";
import { useImagePull } from "@/hooks/use-images";

interface PullDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	serverID: string;
}

export function PullDialog({ open, onOpenChange, serverID }: PullDialogProps) {
	const [imageRef, setImageRef] = useState("");
	const { pulling, overallPercent, error, pull, cancel } = useImagePull();

	const handlePull = () => {
		if (!imageRef.trim()) return;
		pull(serverID, imageRef.trim());
	};

	const handleClose = (nextOpen: boolean) => {
		if (pulling) return;
		if (!nextOpen) {
			setImageRef("");
		}
		onOpenChange(nextOpen);
	};

	return (
		<Dialog open={open} onOpenChange={handleClose}>
			<DialogContent>
				<DialogHeader>
					<DialogTitle className="flex items-center gap-2">
						<Download className="h-5 w-5" />
						Pull Image
					</DialogTitle>
					<DialogDescription>
						Pull a Docker image from a registry to the selected server.
					</DialogDescription>
				</DialogHeader>

				<div className="space-y-4">
					<Input
						placeholder="e.g. nginx:latest, postgres:16-alpine"
						value={imageRef}
						onChange={(e) => setImageRef(e.target.value)}
						disabled={pulling}
						onKeyDown={(e) => {
							if (e.key === "Enter") handlePull();
						}}
					/>

					{pulling && (
						<div className="space-y-2">
							<div className="flex items-center justify-between text-xs text-muted-foreground">
								<span>Pulling...</span>
								<span>{overallPercent}%</span>
							</div>
							<Progress value={overallPercent} />
						</div>
					)}

					{error && !pulling && <p className="text-sm text-destructive">{error}</p>}
				</div>

				<DialogFooter>
					{pulling ? (
						<Button variant="outline" onClick={cancel}>
							Cancel
						</Button>
					) : (
						<>
							<Button variant="outline" onClick={() => handleClose(false)}>
								Close
							</Button>
							<Button onClick={handlePull} disabled={!imageRef.trim()}>
								<Download className="mr-1.5 h-3.5 w-3.5" />
								Pull
							</Button>
						</>
					)}
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}
