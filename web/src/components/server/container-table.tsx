import { useMemo, useState } from "react";
import {
	MoreHorizontal,
	Search,
	RotateCcw,
	Square,
	Trash2,
	ScrollText,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { relativeTime } from "@/lib/format";
import type { ContainerRecord } from "@/types/pocketbase";

interface ContainerTableProps {
	containers: ContainerRecord[];
	isLoading: boolean;
}

const statusColors: Record<string, "default" | "secondary" | "destructive"> = {
	running: "default",
	exited: "secondary",
	created: "secondary",
	restarting: "default",
	paused: "secondary",
	dead: "destructive",
	removing: "destructive",
	removed: "destructive",
};

export function ContainerTable({ containers, isLoading }: ContainerTableProps) {
	const [search, setSearch] = useState("");

	const filtered = useMemo(() => {
		if (!search) return containers;
		const q = search.toLowerCase();
		return containers.filter(
			(c) =>
				c.name.toLowerCase().includes(q) ||
				c.image.toLowerCase().includes(q),
		);
	}, [containers, search]);

	if (isLoading) {
		return (
			<div className="flex flex-col gap-3">
				<Skeleton className="h-9 w-64" />
				<div className="rounded-md border">
					<Table>
						<TableHeader>
							<TableRow>
								{["Name", "Image", "Status", "Restart", "Created", ""].map(
									(h) => (
										<TableHead key={h}>{h}</TableHead>
									),
								)}
							</TableRow>
						</TableHeader>
						<TableBody>
							{Array.from({ length: 5 }).map((_, i) => (
								<TableRow key={i}>
									{Array.from({ length: 6 }).map((_, j) => (
										<TableCell key={j}>
											<Skeleton className="h-4 w-full" />
										</TableCell>
									))}
								</TableRow>
							))}
						</TableBody>
					</Table>
				</div>
			</div>
		);
	}

	return (
		<div className="flex flex-col gap-3">
			<div className="relative w-64">
				<Search className="pointer-events-none absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
				<Input
					placeholder="Search containers..."
					value={search}
					onChange={(e) => setSearch(e.target.value)}
					className="pl-9"
				/>
			</div>

			<div className="rounded-md border">
				<Table>
					<TableHeader>
						<TableRow>
							<TableHead>Name</TableHead>
							<TableHead>Image</TableHead>
							<TableHead>Status</TableHead>
							<TableHead>Restart</TableHead>
							<TableHead>Created</TableHead>
							<TableHead className="w-10" />
						</TableRow>
					</TableHeader>
					<TableBody>
						{filtered.length === 0 ? (
							<TableRow>
								<TableCell colSpan={6} className="text-center text-muted-foreground py-8">
									{search ? "No containers match your search." : "No containers on this server."}
								</TableCell>
							</TableRow>
						) : (
							filtered.map((c) => (
								<TableRow key={c.id}>
									<TableCell className="font-medium text-sm max-w-[200px] truncate">
										{c.name}
									</TableCell>
									<TableCell className="text-sm text-muted-foreground max-w-[200px] truncate">
										{c.image}
									</TableCell>
									<TableCell>
										<Badge variant={statusColors[c.status] ?? "secondary"}>
											{c.status}
										</Badge>
									</TableCell>
									<TableCell className="text-sm text-muted-foreground">
										{c.restart_policy || "—"}
									</TableCell>
									<TableCell className="text-sm text-muted-foreground">
										{relativeTime(c.container_created_at)}
									</TableCell>
									<TableCell>
										<DropdownMenu>
											<DropdownMenuTrigger asChild>
												<Button variant="ghost" size="icon" className="h-8 w-8">
													<MoreHorizontal className="h-4 w-4" />
												</Button>
											</DropdownMenuTrigger>
											<DropdownMenuContent align="end">
												<DropdownMenuItem disabled>
													<RotateCcw className="mr-2 h-4 w-4" />
													Restart
												</DropdownMenuItem>
												<DropdownMenuItem disabled>
													<Square className="mr-2 h-4 w-4" />
													Stop
												</DropdownMenuItem>
												<DropdownMenuItem disabled>
													<ScrollText className="mr-2 h-4 w-4" />
													View Logs
												</DropdownMenuItem>
												<DropdownMenuItem disabled className="text-destructive">
													<Trash2 className="mr-2 h-4 w-4" />
													Remove
												</DropdownMenuItem>
											</DropdownMenuContent>
										</DropdownMenu>
									</TableCell>
								</TableRow>
							))
						)}
					</TableBody>
				</Table>
			</div>
		</div>
	);
}
