import { useCallback, useRef, useState } from "react";
import { useParams, Link } from "react-router";
import { ArrowLeft, TerminalSquare } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useRecord } from "@/hooks/use-record";
import { TerminalView } from "@/components/terminal/terminal-view";
import { SessionList, type TerminalSession } from "@/components/terminal/session-list";

const statusVariantMap: Record<string, "default" | "secondary" | "destructive"> = {
	running: "default",
	paused: "secondary",
	exited: "destructive",
	dead: "destructive",
};

export default function ContainerTerminalPage() {
	const { id } = useParams();
	const { data: container, isLoading } = useRecord("containers", id);
	const statusVariant = statusVariantMap[container?.status ?? ""] ?? "secondary";

	const nextIdRef = useRef(2);
	const [sessions, setSessions] = useState<TerminalSession[]>([{ id: "1", label: "Session 1" }]);
	const [activeId, setActiveId] = useState("1");

	const addSession = useCallback(() => {
		const num = nextIdRef.current++;
		const newSession: TerminalSession = {
			id: String(num),
			label: `Session ${num}`,
		};
		setSessions((prev) => [...prev, newSession]);
		setActiveId(newSession.id);
	}, []);

	const closeSession = useCallback(
		(sessionId: string) => {
			setSessions((prev) => {
				const next = prev.filter((s) => s.id !== sessionId);
				if (next.length === 0) return prev; // don't close last
				if (activeId === sessionId) {
					setActiveId(next[next.length - 1].id);
				}
				return next;
			});
		},
		[activeId],
	);

	return (
		<div className="flex flex-col h-[calc(100vh-4rem)]">
			{/* Header */}
			<div className="flex items-center gap-3 px-4 py-3 border-b shrink-0">
				<Button variant="ghost" size="icon" className="h-8 w-8" asChild>
					<Link to={id ? `/containers/${id}` : "/containers"}>
						<ArrowLeft className="h-4 w-4" />
					</Link>
				</Button>
				<TerminalSquare className="h-5 w-5 text-muted-foreground" />
				{isLoading ? (
					<Skeleton className="h-6 w-48" />
				) : (
					<>
						<h1 className="text-lg font-semibold">
							{container?.name || "Container"} &mdash; Terminal
						</h1>
						{container && <Badge variant={statusVariant}>{container.status}</Badge>}
					</>
				)}
			</div>

			{/* Session tabs */}
			<SessionList
				sessions={sessions}
				activeId={activeId}
				onSelect={setActiveId}
				onAdd={addSession}
				onClose={closeSession}
			/>

			{/* Terminal area */}
			<div className="flex-1 min-h-0">
				{id &&
					sessions.map((s) => (
						<TerminalView key={s.id} containerId={id} active={s.id === activeId} />
					))}
			</div>
		</div>
	);
}
