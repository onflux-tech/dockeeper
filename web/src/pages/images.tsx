import { useMemo, useState } from "react";
import { Layers, Download, Trash2, Eraser } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
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
import { useCollection } from "@/hooks/use-collection";
import { useRealtime } from "@/hooks/use-realtime";
import { useImageRemove, useImagePrune } from "@/hooks/use-images";
import { PullDialog } from "@/components/images/pull-dialog";

function formatSize(bytes: number): string {
	if (bytes === 0) return "0 B";
	const mb = bytes / (1024 * 1024);
	if (mb >= 1024) return `${(mb / 1024).toFixed(1)} GB`;
	return `${mb.toFixed(1)} MB`;
}

function formatTags(tags: string[] | string | null): string {
	if (!tags) return "<none>";
	if (typeof tags === "string") {
		try {
			const parsed = JSON.parse(tags);
			if (Array.isArray(parsed)) return parsed.join(", ") || "<none>";
		} catch {
			return tags || "<none>";
		}
	}
	if (Array.isArray(tags)) return tags.join(", ") || "<none>";
	return "<none>";
}

export default function ImagesPage() {
	const [serverFilter, setServerFilter] = useState<string>("all");
	const [search, setSearch] = useState("");
	const [pullOpen, setPullOpen] = useState(false);
	const [removeTarget, setRemoveTarget] = useState<{ id: string; name: string } | null>(null);

	const { data: serversResult } = useCollection("servers", { perPage: 100 });
	const { data: imagesResult, isLoading } = useCollection("images", {
		perPage: 500,
		sort: "-created",
		...(serverFilter !== "all" ? { filter: `server = '${serverFilter}'` } : {}),
	});

	useRealtime("images");

	const servers = serversResult?.items ?? [];
	const images = useMemo(() => imagesResult?.items ?? [], [imagesResult?.items]);

	const removeMut = useImageRemove();
	const pruneMut = useImagePrune();

	const filtered = useMemo(() => {
		if (!search) return images;
		const q = search.toLowerCase();
		return images.filter((img) => {
			const tags = formatTags(img.tags).toLowerCase();
			const id = (img.image_id ?? "").toLowerCase();
			return tags.includes(q) || id.includes(q);
		});
	}, [images, search]);

	const handlePrune = () => {
		if (serverFilter === "all") {
			toast.error("Select a server to prune images");
			return;
		}
		pruneMut.mutate({ serverID: serverFilter, all: false });
	};

	return (
		<div className="flex flex-col gap-6">
			<div className="flex items-center justify-between">
				<div className="flex items-center gap-2">
					<Layers className="h-5 w-5 text-muted-foreground" />
					<h1 className="text-2xl font-semibold">Images</h1>
					{!isLoading && (
						<span className="text-sm text-muted-foreground">
							({filtered.length})
						</span>
					)}
				</div>
				<div className="flex items-center gap-2">
					<Button
						variant="outline"
						size="sm"
						onClick={handlePrune}
						disabled={pruneMut.isPending || serverFilter === "all"}
					>
						<Eraser className="mr-1.5 h-3.5 w-3.5" />
						Prune
					</Button>
					<Button
						size="sm"
						onClick={() => setPullOpen(true)}
						disabled={serverFilter === "all"}
					>
						<Download className="mr-1.5 h-3.5 w-3.5" />
						Pull
					</Button>
				</div>
			</div>

			{/* Filters */}
			<div className="flex items-center gap-3">
				<Select value={serverFilter} onValueChange={setServerFilter}>
					<SelectTrigger className="w-48">
						<SelectValue placeholder="All Servers" />
					</SelectTrigger>
					<SelectContent>
						<SelectItem value="all">All Servers</SelectItem>
						{servers.map((s) => (
							<SelectItem key={s.id} value={s.id}>
								{s.hostname}
							</SelectItem>
						))}
					</SelectContent>
				</Select>
				<Input
					placeholder="Search images..."
					value={search}
					onChange={(e) => setSearch(e.target.value)}
					className="max-w-xs"
				/>
			</div>

			{/* Table */}
			{isLoading ? (
				<div className="space-y-2">
					{[1, 2, 3, 4, 5].map((i) => (
						<Skeleton key={i} className="h-10 w-full" />
					))}
				</div>
			) : filtered.length === 0 ? (
				<p className="text-sm text-muted-foreground">No images found.</p>
			) : (
				<div className="rounded-md border">
					<Table>
						<TableHeader>
							<TableRow>
								<TableHead>Tags</TableHead>
								<TableHead>Image ID</TableHead>
								<TableHead className="text-right">Size</TableHead>
								<TableHead>In Use</TableHead>
								<TableHead>Created</TableHead>
								<TableHead className="w-12" />
							</TableRow>
						</TableHeader>
						<TableBody>
							{filtered.map((img) => (
								<TableRow key={img.id}>
									<TableCell className="font-mono text-xs max-w-64 truncate">
										{formatTags(img.tags)}
									</TableCell>
									<TableCell className="font-mono text-xs text-muted-foreground max-w-32 truncate">
										{img.image_id?.slice(7, 19) ?? "—"}
									</TableCell>
									<TableCell className="text-right text-xs">
										{formatSize(img.size)}
									</TableCell>
									<TableCell>
										<Badge variant={img.in_use ? "default" : "secondary"}>
											{img.in_use ? "Yes" : "No"}
										</Badge>
									</TableCell>
									<TableCell className="text-xs text-muted-foreground">
										{img.image_created_at
											? new Date(img.image_created_at).toLocaleDateString()
											: "—"}
									</TableCell>
									<TableCell>
										<Button
											variant="ghost"
											size="icon"
											className="h-8 w-8"
											onClick={() =>
												setRemoveTarget({
													id: img.id,
													name: formatTags(img.tags),
												})
											}
											disabled={img.in_use}
										>
											<Trash2 className="h-3.5 w-3.5 text-destructive" />
										</Button>
									</TableCell>
								</TableRow>
							))}
						</TableBody>
					</Table>
				</div>
			)}

			{/* Pull Dialog */}
			{serverFilter !== "all" && (
				<PullDialog
					open={pullOpen}
					onOpenChange={setPullOpen}
					serverID={serverFilter}
				/>
			)}

			{/* Remove Confirmation */}
			<AlertDialog
				open={!!removeTarget}
				onOpenChange={(open) => {
					if (!open) setRemoveTarget(null);
				}}
			>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>Remove Image</AlertDialogTitle>
						<AlertDialogDescription>
							Are you sure you want to remove{" "}
							<span className="font-mono font-semibold">
								{removeTarget?.name}
							</span>
							? This action cannot be undone.
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel>Cancel</AlertDialogCancel>
						<AlertDialogAction
							onClick={() => {
								if (removeTarget) {
									removeMut.mutate({ id: removeTarget.id, force: true });
									setRemoveTarget(null);
								}
							}}
						>
							Remove
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
		</div>
	);
}
