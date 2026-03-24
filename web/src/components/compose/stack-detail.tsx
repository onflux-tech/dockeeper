import { useState, useRef, useCallback, useEffect } from "react";
import { Play, Square, Download, Trash2, Save, FileCode, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
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
import {
	useUpdateStack,
	useDeleteStack,
	streamComposeOp,
	type ComposeOutputEvent,
} from "@/hooks/use-compose";
import type { ComposeStackRecord } from "@/types/pocketbase";

interface StackDetailProps {
	stack: ComposeStackRecord;
	onDeleted: () => void;
}

const statusVariant: Record<string, "default" | "secondary" | "destructive"> = {
	running: "default",
	stopped: "secondary",
	partial: "destructive",
};

export function StackDetail({ stack, onDeleted }: StackDetailProps) {
	const [yaml, setYaml] = useState(stack.yaml_content);
	const [logs, setLogs] = useState<string[]>([]);
	const [opRunning, setOpRunning] = useState(false);
	const [deleteOpen, setDeleteOpen] = useState(false);
	const abortRef = useRef<AbortController | null>(null);
	const logsEndRef = useRef<HTMLDivElement>(null);

	const updateMut = useUpdateStack();
	const deleteMut = useDeleteStack();

	const dirty = yaml !== stack.yaml_content;

	// Sync yaml when stack changes externally
	useEffect(() => {
		setYaml(stack.yaml_content);
	}, [stack.yaml_content]);

	// Auto-scroll logs
	useEffect(() => {
		logsEndRef.current?.scrollIntoView({ behavior: "smooth" });
	}, [logs]);

	const handleSave = () => {
		if (!dirty) return;
		updateMut.mutate({ id: stack.id, yaml_content: yaml });
	};

	const runOp = useCallback(
		async (op: "up" | "down" | "pull") => {
			setOpRunning(true);
			setLogs([`>>> compose ${op}`]);
			const ctrl = new AbortController();
			abortRef.current = ctrl;

			try {
				await streamComposeOp(
					stack.id,
					op,
					(ev: ComposeOutputEvent) => {
						if (ev.line) {
							setLogs((prev) => [...prev, ev.line!]);
						}
						if (ev.error) {
							setLogs((prev) => [...prev, `ERROR: ${ev.error}`]);
						}
						if (ev.done) {
							const code = ev.exit_code ?? 0;
							setLogs((prev) => [
								...prev,
								code === 0
									? `✓ compose ${op} completed`
									: `✗ compose ${op} exited with code ${code}`,
							]);
						}
					},
					ctrl.signal,
				);
			} catch (err) {
				if ((err as Error).name !== "AbortError") {
					toast.error(`Compose ${op} failed: ${(err as Error).message}`);
					setLogs((prev) => [...prev, `FATAL: ${(err as Error).message}`]);
				}
			} finally {
				setOpRunning(false);
				abortRef.current = null;
			}
		},
		[stack.id],
	);

	const handleDelete = () => {
		deleteMut.mutate(stack.id, {
			onSuccess: () => onDeleted(),
		});
	};

	return (
		<div className="flex flex-col gap-4">
			{/* Header */}
			<div className="flex items-center justify-between">
				<div className="flex items-center gap-3">
					<h2 className="text-lg font-semibold">{stack.name}</h2>
					<Badge variant={statusVariant[stack.status] ?? "secondary"}>{stack.status}</Badge>
					{stack.services_count > 0 && (
						<span className="text-xs text-muted-foreground">
							{stack.services_count} service{stack.services_count !== 1 && "s"}
						</span>
					)}
				</div>

				<div className="flex items-center gap-2">
					<Button variant="outline" size="sm" onClick={() => runOp("pull")} disabled={opRunning}>
						<Download className="mr-1.5 h-3.5 w-3.5" />
						Pull
					</Button>
					<Button variant="outline" size="sm" onClick={() => runOp("up")} disabled={opRunning}>
						<Play className="mr-1.5 h-3.5 w-3.5" />
						Up
					</Button>
					<Button variant="outline" size="sm" onClick={() => runOp("down")} disabled={opRunning}>
						<Square className="mr-1.5 h-3.5 w-3.5" />
						Down
					</Button>
					<Button
						variant="destructive"
						size="sm"
						onClick={() => setDeleteOpen(true)}
						disabled={opRunning || deleteMut.isPending}
					>
						<Trash2 className="mr-1.5 h-3.5 w-3.5" />
						Delete
					</Button>
				</div>
			</div>

			{/* YAML Editor */}
			<div className="space-y-1.5">
				<div className="flex items-center justify-between">
					<label className="flex items-center gap-1.5 text-xs font-medium">
						<FileCode className="h-3.5 w-3.5" />
						docker-compose.yml
					</label>
					<Button
						variant="outline"
						size="sm"
						onClick={handleSave}
						disabled={!dirty || updateMut.isPending}
					>
						{updateMut.isPending ? (
							<Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />
						) : (
							<Save className="mr-1.5 h-3.5 w-3.5" />
						)}
						Save
					</Button>
				</div>
				<Textarea
					value={yaml}
					onChange={(e) => setYaml(e.target.value)}
					className="min-h-64 font-mono text-xs leading-relaxed"
					spellCheck={false}
				/>
			</div>

			{/* Operation Logs */}
			{logs.length > 0 && (
				<div className="space-y-1.5">
					<label className="text-xs font-medium">Output</label>
					<div className="max-h-48 overflow-y-auto rounded-md border bg-muted/30 p-3 font-mono text-xs leading-relaxed">
						{logs.map((line, i) => (
							<div key={i}>{line}</div>
						))}
						<div ref={logsEndRef} />
					</div>
				</div>
			)}

			{/* Delete Confirmation */}
			<AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>Delete Stack</AlertDialogTitle>
						<AlertDialogDescription>
							This will run <span className="font-mono">compose down</span> and permanently delete
							the stack <span className="font-mono font-semibold">{stack.name}</span>. This action
							cannot be undone.
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel>Cancel</AlertDialogCancel>
						<AlertDialogAction onClick={handleDelete}>Delete</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
		</div>
	);
}
