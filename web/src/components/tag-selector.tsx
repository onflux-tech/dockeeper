import { useState } from "react";
import { ChevronsUpDown, Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { useTagNames, useApplyTag, useRemoveTag, useTags } from "@/hooks/use-tags";
import { TagBadge } from "./tag-badge";
import type { Tag } from "@/hooks/use-tags";

const TAG_COLORS = [
	"#ef4444",
	"#f97316",
	"#eab308",
	"#22c55e",
	"#06b6d4",
	"#3b82f6",
	"#8b5cf6",
	"#ec4899",
	"#6b7280",
	"#14b8a6",
];

interface TagSelectorProps {
	resourceType: "container" | "server";
	resourceId: string;
}

export function TagSelector({ resourceType, resourceId }: TagSelectorProps) {
	const [open, setOpen] = useState(false);
	const [search, setSearch] = useState("");
	const [newColor, setNewColor] = useState(TAG_COLORS[0]);

	const { data: tagNamesData } = useTagNames(resourceType);
	const { data: tagsData } = useTags(resourceType);
	const applyTag = useApplyTag();
	const removeTag = useRemoveTag();

	const tagNames = tagNamesData?.items ?? [];
	const allTags = tagsData?.items ?? [];

	// Tags applied to this resource.
	const appliedTags = allTags.filter((t) => t.resource_id === resourceId);
	const appliedNames = new Set(appliedTags.map((t) => t.name));

	// Filter suggestions.
	const filtered = tagNames.filter(
		(t) => t.name.toLowerCase().includes(search.toLowerCase()) && !appliedNames.has(t.name),
	);

	const canCreate =
		search.trim() !== "" &&
		!tagNames.some((t) => t.name.toLowerCase() === search.trim().toLowerCase());

	function handleApply(name: string, color: string) {
		applyTag.mutate({
			name,
			color,
			resource_type: resourceType,
			resource_id: resourceId,
		});
		setSearch("");
	}

	function handleRemove(tag: Tag) {
		removeTag.mutate(tag.id);
	}

	return (
		<div className="flex flex-wrap items-center gap-1.5">
			{appliedTags.map((tag) => (
				<TagBadge
					key={tag.id}
					name={tag.name}
					color={tag.color}
					onRemove={() => handleRemove(tag)}
				/>
			))}

			<Popover open={open} onOpenChange={setOpen}>
				<PopoverTrigger asChild>
					<Button
						variant="ghost"
						size="sm"
						className="h-6 gap-1 px-2 text-xs text-muted-foreground"
					>
						<Plus className="h-3 w-3" />
						Tag
						<ChevronsUpDown className="h-3 w-3" />
					</Button>
				</PopoverTrigger>
				<PopoverContent className="w-64 p-2" align="start">
					<Input
						placeholder="Search or create tag..."
						value={search}
						onChange={(e) => setSearch(e.target.value)}
						className="mb-2 h-8 text-sm"
					/>

					<div className="max-h-40 overflow-y-auto">
						{filtered.map((t) => (
							<button
								key={t.name}
								type="button"
								className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm hover:bg-accent"
								onClick={() => handleApply(t.name, t.color)}
							>
								<span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: t.color }} />
								{t.name}
								<span className="ml-auto text-xs text-muted-foreground">{t.count}</span>
							</button>
						))}

						{canCreate && (
							<button
								type="button"
								className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm hover:bg-accent"
								onClick={() => handleApply(search.trim(), newColor)}
							>
								<Plus className="h-3 w-3" />
								Create &quot;{search.trim()}&quot;
							</button>
						)}

						{filtered.length === 0 && !canCreate && (
							<p className="py-2 text-center text-xs text-muted-foreground">No tags found.</p>
						)}
					</div>

					{/* Color picker for new tags */}
					{canCreate && (
						<div className="mt-2 flex flex-wrap gap-1 border-t pt-2">
							{TAG_COLORS.map((c) => (
								<button
									key={c}
									type="button"
									className={cn(
										"h-5 w-5 rounded-full border-2 transition-transform",
										newColor === c ? "scale-110 border-foreground" : "border-transparent",
									)}
									style={{ backgroundColor: c }}
									onClick={() => setNewColor(c)}
								/>
							))}
						</div>
					)}
				</PopoverContent>
			</Popover>
		</div>
	);
}
