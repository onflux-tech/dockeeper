import { useMemo, useState } from "react";
import { Layers, Download, Trash2, Eraser, Search } from "lucide-react";
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
import { useListFilters } from "@/hooks/use-list-filters";
import { useRealtime } from "@/hooks/use-realtime";
import { useImageRemove, useImagePrune } from "@/hooks/use-images";
import { PullDialog } from "@/components/images/pull-dialog";
import { PaginationControls } from "@/components/ui/pagination-controls";
import { PerPageSelector } from "@/components/ui/per-page-selector";

function formatSize(bytes: number): string {
	if (bytes === 0) return "0 B";
	const mb = bytes / (1024 * 1024);
	if (mb >= 1024) return `${(mb / 1024).toFixed(1)} GB`;
	return `${mb.toFixed(1)} MB`;
}

function formatTags(tags: string[] | string | null): string {
	if (!tags) return "<none>";
	let arr: string[];
	if (typeof tags === "string") {
		try {
			const parsed = JSON.parse(tags);
			if (Array.isArray(parsed)) {
				arr = parsed;
			} else {
				return tags || "<none>";
			}
		} catch {
			return tags || "<none>";
		}
	} else if (Array.isArray(tags)) {
		arr = tags;
	} else {
		return "<none>";
	}
	if (arr.length === 0) return "<none>";
	return arr
		.map((t) => {
			// "repo:<none>" → show just the repo name (untagged)
			if (t.endsWith(":<none>")) return t.slice(0, -7);
			return t;
		})
		.join(", ");
}

export default function ImagesPage() {
	const { filters, setSearch, setPage, setPerPage, setCustom, pbOptions } = useListFilters({
		defaultSort: "-created",
		searchFields: ["tags", "image_id"],
		customFilters: { server: "server" },
	});
	const [pullOpen, setPullOpen] = useState(false);
	const [removeTarget, setRemoveTarget] = useState<{ id: string; name: string } | null>(null);

	const { data: serversResult } = useCollection("servers", { perPage: 100 });
	const { data: imagesResult, isLoading } = useCollection("images", pbOptions);

	useRealtime("images");

	const serverFilter = filters.custom.server ?? "all";
	const servers = serversResult?.items ?? [];
	const images = useMemo(() => imagesResult?.items ?? [], [imagesResult?.items]);
	const totalImages = imagesResult?.totalItems ?? 0;

	const removeMut = useImageRemove();
	const pruneMut = useImagePrune();

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
					{!isLoading && <Badge variant="secondary" className="ml-1">{totalImages}</Badge>}
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
					<Button size="sm" onClick={() => setPullOpen(true)} disabled={serverFilter === "all"}>
						<Download className="mr-1.5 h-3.5 w-3.5" />
						Pull
					</Button>
				</div>
			</div>

			{/* Filters */}
			<div className="flex items-center gap-3">
				<Select
					value={serverFilter}
					onValueChange={(value) => {
						setCustom("server", value);
					}}
				>
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
				<div className="relative w-64">
					<Search className="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
					<Input
						placeholder="Search images..."
						value={filters.search}
						onChange={(e) => {
							setSearch(e.target.value);
						}}
						className="pl-8"
					/>
				</div>
			</div>

			{/* Table */}
			{isLoading ? (
				<div className="space-y-2">
					{[1, 2, 3, 4, 5].map((i) => (
						<Skeleton key={i} className="h-10 w-full" />
					))}
				</div>
			) : images.length === 0 ? (
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
							{images.map((img) => (
								<TableRow key={img.id}>
									<TableCell className="font-mono text-xs max-w-64 truncate">
										{formatTags(img.tags)}
									</TableCell>
									<TableCell className="font-mono text-xs text-muted-foreground max-w-32 truncate">
										{img.image_id?.slice(7, 19) ?? "—"}
									</TableCell>
									<TableCell className="text-right text-xs">{formatSize(img.size)}</TableCell>
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

			<div className="flex items-center justify-between">
				<PerPageSelector value={filters.perPage} onChange={setPerPage} />
				<PaginationControls
					page={filters.page}
					totalPages={imagesResult?.totalPages ?? 0}
					totalItems={totalImages}
					onPageChange={setPage}
					label="images"
				/>
			</div>

			{/* Pull Dialog */}
			{serverFilter !== "all" && (
				<PullDialog open={pullOpen} onOpenChange={setPullOpen} serverID={serverFilter} />
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
							<span className="font-mono font-semibold">{removeTarget?.name}</span>? This action
							cannot be undone.
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
