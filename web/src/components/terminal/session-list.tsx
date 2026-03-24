import { Plus, X } from "lucide-react";
import { Button } from "@/components/ui/button";

export interface TerminalSession {
	id: string;
	label: string;
}

interface SessionListProps {
	sessions: TerminalSession[];
	activeId: string;
	onSelect: (id: string) => void;
	onAdd: () => void;
	onClose: (id: string) => void;
}

export function SessionList({
	sessions,
	activeId,
	onSelect,
	onAdd,
	onClose,
}: SessionListProps) {
	return (
		<div className="flex items-center gap-1 px-2 py-1 border-b bg-muted/30 overflow-x-auto">
			{sessions.map((s) => (
				<button
					key={s.id}
					type="button"
					onClick={() => onSelect(s.id)}
					className={`inline-flex items-center gap-1.5 px-3 py-1 rounded-md text-xs font-medium transition-colors ${
						s.id === activeId
							? "bg-background text-foreground shadow-sm"
							: "text-muted-foreground hover:text-foreground hover:bg-background/50"
					}`}
				>
					{s.label}
					{sessions.length > 1 && (
						<span
							role="button"
							tabIndex={0}
							onClick={(e) => {
								e.stopPropagation();
								onClose(s.id);
							}}
							onKeyDown={(e) => {
								if (e.key === "Enter") {
									e.stopPropagation();
									onClose(s.id);
								}
							}}
							className="ml-0.5 rounded p-0.5 hover:bg-muted"
						>
							<X className="h-3 w-3" />
						</span>
					)}
				</button>
			))}
			<Button
				variant="ghost"
				size="icon"
				className="h-6 w-6 shrink-0"
				onClick={onAdd}
				title="New session"
			>
				<Plus className="h-3.5 w-3.5" />
			</Button>
		</div>
	);
}
