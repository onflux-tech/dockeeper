import { CheckCircle2, XCircle, Loader2, Ban } from "lucide-react";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useBatchProgress, buildItemStatusMap } from "@/hooks/use-batch";

interface BatchProgressDialogProps {
	jobId: string | null;
	action: string;
	open: boolean;
	onOpenChange: (open: boolean) => void;
	containerNames?: Map<string, string>;
}

export function BatchProgressDialog({
	jobId,
	action,
	open,
	onOpenChange,
	containerNames,
}: BatchProgressDialogProps) {
	const { progress, latestProgress, isComplete } = useBatchProgress(jobId);

	const statusMap = buildItemStatusMap(progress);
	const total = latestProgress?.total ?? 0;
	const succeeded = latestProgress?.succeeded ?? 0;
	const failed = latestProgress?.failed ?? 0;
	const cancelled = latestProgress?.cancelled ?? 0;
	const done = succeeded + failed + cancelled;
	const pct = total > 0 ? Math.round((done / total) * 100) : 0;

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="sm:max-w-md">
				<DialogHeader>
					<DialogTitle className="capitalize">Batch {action}</DialogTitle>
					<DialogDescription>
						{isComplete
							? `Completed: ${succeeded} succeeded, ${failed} failed${cancelled > 0 ? `, ${cancelled} cancelled` : ""}`
							: `Processing ${done}/${total}...`}
					</DialogDescription>
				</DialogHeader>

				<div className="space-y-4">
					<Progress value={pct} className="h-2" />

					<ScrollArea className="max-h-60">
						<div className="space-y-1">
							{[...statusMap.entries()].map(([containerId, item]) => (
								<div
									key={containerId}
									className="flex items-center gap-2 rounded-md px-2 py-1 text-sm"
								>
									<StatusIcon status={item.status} />
									<span className="truncate">
										{containerNames?.get(containerId) ?? containerId.slice(0, 12)}
									</span>
									{item.error && (
										<span className="ml-auto truncate text-xs text-destructive">{item.error}</span>
									)}
								</div>
							))}
						</div>
					</ScrollArea>

					{isComplete && (
						<div className="flex justify-end">
							<Button size="sm" onClick={() => onOpenChange(false)}>
								Close
							</Button>
						</div>
					)}
				</div>
			</DialogContent>
		</Dialog>
	);
}

function StatusIcon({ status }: { status: string }) {
	switch (status) {
		case "success":
			return <CheckCircle2 className="h-4 w-4 shrink-0 text-success" />;
		case "failed":
			return <XCircle className="h-4 w-4 shrink-0 text-destructive" />;
		case "cancelled":
			return <Ban className="h-4 w-4 shrink-0 text-muted-foreground" />;
		case "running":
			return <Loader2 className="h-4 w-4 shrink-0 animate-spin text-primary" />;
		default:
			return <div className="h-4 w-4 shrink-0 rounded-full border-2 border-muted-foreground/30" />;
	}
}
