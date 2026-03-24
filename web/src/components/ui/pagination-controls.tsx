import { ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";

interface PaginationControlsProps {
	page: number;
	totalPages: number;
	totalItems: number;
	onPageChange: (page: number) => void;
	label?: string;
	className?: string;
}

export function PaginationControls({
	page,
	totalPages,
	totalItems,
	onPageChange,
	label = "items",
	className,
}: PaginationControlsProps) {
	if (totalPages <= 1) {
		return null;
	}

	return (
		<div className={className ?? "flex items-center justify-between gap-3"}>
			<p className="text-sm text-muted-foreground">
				Page {page} of {totalPages} ({totalItems} {label})
			</p>
			<div className="flex items-center gap-2">
				<Button
					variant="outline"
					size="sm"
					disabled={page <= 1}
					onClick={() => onPageChange(Math.max(1, page - 1))}
				>
					<ChevronLeft className="h-4 w-4" />
					Previous
				</Button>
				<Button
					variant="outline"
					size="sm"
					disabled={page >= totalPages}
					onClick={() => onPageChange(Math.min(totalPages, page + 1))}
				>
					Next
					<ChevronRight className="h-4 w-4" />
				</Button>
			</div>
		</div>
	);
}