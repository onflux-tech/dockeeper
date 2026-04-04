import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { useCollection } from "@/hooks/use-collection";
import type { ServerRecord } from "@/types/pocketbase";

interface ServerFilterSelectProps {
	value: string;
	onValueChange: (value: string) => void;
}

export function ServerFilterSelect({ value, onValueChange }: ServerFilterSelectProps) {
	const { data, isLoading } = useCollection("servers", { perPage: 200, sort: "name" });
	const servers = (data?.items ?? []) as ServerRecord[];

	return (
		<Select value={value} onValueChange={onValueChange} disabled={isLoading}>
			<SelectTrigger className="w-48">
				<SelectValue placeholder="All Servers" />
			</SelectTrigger>
			<SelectContent>
				<SelectItem value="all">All Servers</SelectItem>
				{servers
					.slice()
					.sort((a, b) => {
						const nameA = a.name || a.hostname || "";
						const nameB = b.name || b.hostname || "";
						return nameA.localeCompare(nameB);
					})
					.map((s) => (
						<SelectItem key={s.id} value={s.id}>
							{s.name || s.hostname || s.id.slice(0, 8)}
						</SelectItem>
					))}
			</SelectContent>
		</Select>
	);
}
