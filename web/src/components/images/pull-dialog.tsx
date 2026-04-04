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

function formatBytes(bytes: number): string {
	if (bytes === 0) return "0 B";
	const k = 1024;
	const sizes = ["B", "KB", "MB", "GB"];
	const i = Math.floor(Math.log(bytes) / Math.log(k));
	return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
}

export function PullDialog({ open, onOpenChange, serverID }: PullDialogProps) {
	const [imageRef, setImageRef] = useState("");
	const { pulling, layers, overallPercent, error, pull, cancel } = useImagePull();

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

	// Build sorted layer list for display.
	const layerList = pulling ? Array.from(layers.values()).filter((l) => l.id && l.total > 0) : [];

	return (
		<Dialog open={open} onOpenChange={handleClose}>
			<DialogContent className="sm:max-w-lg">
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
						<div className="space-y-3">
							<div className="flex items-center justify-between text-xs text-muted-foreground">
								<span>Pulling...</span>
								<span>{overallPercent}%</span>
							</div>
							<Progress value={overallPercent} />

							{/* Per-layer progress */}
							{layerList.length > 0 && (
								<div className="max-h-40 overflow-auto space-y-1.5 rounded-md bg-muted/50 p-2">
									{layerList.map((layer) => {
										const pct =
											layer.total > 0 ? Math.round((layer.current / layer.total) * 100) : 0;
										return (
											<div key={layer.id} className="space-y-0.5">
												<div className="flex items-center justify-between text-[10px] text-muted-foreground font-mono">
													<span className="truncate max-w-[120px]">{layer.id.slice(0, 12)}</span>
													<span>
														{layer.status === "Pull complete" || layer.status === "Already exists"
															? layer.status
															: `${formatBytes(layer.current)} / ${formatBytes(layer.total)}`}
													</span>
												</div>
												<div className="h-1 w-full rounded-full bg-muted">
													<div
														className="h-full rounded-full bg-primary transition-all duration-200"
														style={{ width: `${pct}%` }}
													/>
												</div>
											</div>
										);
									})}
								</div>
							)}
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
