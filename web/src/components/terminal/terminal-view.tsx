import { useRef } from "react";
import { Search, ChevronUp, ChevronDown, X, Circle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { useTerminal } from "@/hooks/use-terminal";

import "@xterm/xterm/css/xterm.css";

interface TerminalViewProps {
	containerId: string;
	cmd?: string;
	active?: boolean;
}

export function TerminalView({ containerId, cmd, active = true }: TerminalViewProps) {
	const {
		containerRef,
		connected,
		searchVisible,
		searchQuery,
		setSearchQuery,
		toggleSearch,
		findNext,
		findPrevious,
	} = useTerminal(containerId, { cmd });

	const searchInputRef = useRef<HTMLInputElement>(null);

	const handleSearchKeyDown = (e: React.KeyboardEvent) => {
		if (e.key === "Enter") {
			if (e.shiftKey) findPrevious();
			else findNext();
		}
		if (e.key === "Escape") toggleSearch();
	};

	return (
		<div className="flex flex-col h-full" style={{ display: active ? "flex" : "none" }}>
			{/* Toolbar */}
			<div className="flex items-center justify-between gap-2 px-3 py-1.5 border-b bg-card">
				<Badge variant={connected ? "default" : "secondary"} className="text-xs">
					<Circle
						className={`h-2 w-2 mr-1 ${connected ? "fill-current" : "fill-muted-foreground"}`}
					/>
					{connected ? "Connected" : "Disconnected"}
				</Badge>

				<div className="flex items-center gap-1">
					{searchVisible && (
						<div className="flex items-center gap-1">
							<Input
								ref={searchInputRef}
								value={searchQuery}
								onChange={(e) => setSearchQuery(e.target.value)}
								onKeyDown={handleSearchKeyDown}
								placeholder="Search..."
								className="h-7 w-48 text-xs"
								autoFocus
							/>
							<Button
								variant="ghost"
								size="icon"
								className="h-7 w-7"
								onClick={findPrevious}
								title="Previous (Shift+Enter)"
							>
								<ChevronUp className="h-3.5 w-3.5" />
							</Button>
							<Button
								variant="ghost"
								size="icon"
								className="h-7 w-7"
								onClick={findNext}
								title="Next (Enter)"
							>
								<ChevronDown className="h-3.5 w-3.5" />
							</Button>
							<Button
								variant="ghost"
								size="icon"
								className="h-7 w-7"
								onClick={toggleSearch}
								title="Close (Esc)"
							>
								<X className="h-3.5 w-3.5" />
							</Button>
						</div>
					)}
					{!searchVisible && (
						<Button
							variant="ghost"
							size="icon"
							className="h-7 w-7"
							onClick={() => {
								toggleSearch();
								setTimeout(() => searchInputRef.current?.focus(), 50);
							}}
							title="Search (Ctrl+F)"
						>
							<Search className="h-3.5 w-3.5" />
						</Button>
					)}
				</div>
			</div>

			{/* Terminal */}
			<div ref={containerRef} className="flex-1 min-h-0 bg-[#0a0a0a] p-1" />
		</div>
	);
}
