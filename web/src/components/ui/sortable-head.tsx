import { ArrowUpDown } from "lucide-react";
import { TableHead } from "@/components/ui/table";

export type SortDir = "asc" | "desc";

interface SortableHeadProps<K extends string> {
	label: string;
	sortKey: K;
	currentKey: K;
	currentDir: SortDir;
	onToggle: (key: K) => void;
	className?: string;
}

export function SortableHead<K extends string>({
	label,
	sortKey,
	currentKey,
	currentDir,
	onToggle,
	className,
}: SortableHeadProps<K>) {
	const active = sortKey === currentKey;
	return (
		<TableHead className={className}>
			<button
				type="button"
				className="inline-flex items-center gap-1 text-xs font-medium"
				onClick={() => onToggle(sortKey)}
			>
				{label}
				<ArrowUpDown
					className={`h-3 w-3 ${active ? "text-foreground" : "text-muted-foreground/50"}`}
				/>
				{active && (
					<span className="text-[10px] text-muted-foreground">
						{currentDir === "asc" ? "↑" : "↓"}
					</span>
				)}
			</button>
		</TableHead>
	);
}
