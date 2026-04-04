import { X } from "lucide-react";
import { cn } from "@/lib/utils";

interface TagBadgeProps {
	name: string;
	color?: string;
	onRemove?: () => void;
	className?: string;
}

export function TagBadge({ name, color = "#6b7280", onRemove, className }: TagBadgeProps) {
	return (
		<span
			className={cn(
				"inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium",
				className,
			)}
			style={{
				backgroundColor: `${color}20`,
				color,
				border: `1px solid ${color}40`,
			}}
		>
			<span className="inline-block h-2 w-2 rounded-full" style={{ backgroundColor: color }} />
			{name}
			{onRemove && (
				<button
					type="button"
					onClick={(e) => {
						e.stopPropagation();
						onRemove();
					}}
					className="ml-0.5 rounded-full p-0.5 hover:bg-black/10 dark:hover:bg-white/10"
				>
					<X className="h-2.5 w-2.5" />
				</button>
			)}
		</span>
	);
}
