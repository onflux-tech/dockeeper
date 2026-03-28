import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { useTagNames } from "@/hooks/use-tags";

interface TagFilterSelectProps {
	resourceType?: "container" | "server";
	value: string;
	onValueChange: (value: string) => void;
}

export function TagFilterSelect({
	resourceType = "container",
	value,
	onValueChange,
}: TagFilterSelectProps) {
	const { data } = useTagNames(resourceType);
	const tagNames = data?.items ?? [];

	if (tagNames.length === 0) return null;

	return (
		<Select value={value} onValueChange={onValueChange}>
			<SelectTrigger className="w-36">
				<SelectValue placeholder="Tag" />
			</SelectTrigger>
			<SelectContent>
				<SelectItem value="all">All tags</SelectItem>
				{tagNames.map((t) => (
					<SelectItem key={t.name} value={t.name}>
						<span className="flex items-center gap-1.5">
							<span
								className="inline-block h-2 w-2 rounded-full"
								style={{ backgroundColor: t.color }}
							/>
							{t.name}
							<span className="text-xs text-muted-foreground">({t.count})</span>
						</span>
					</SelectItem>
				))}
			</SelectContent>
		</Select>
	);
}
