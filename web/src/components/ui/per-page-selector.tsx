import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";

interface PerPageSelectorProps {
	value: number;
	onChange: (value: number) => void;
}

const options = [25, 50, 100] as const;

export function PerPageSelector({ value, onChange }: PerPageSelectorProps) {
	return (
		<Select value={String(value)} onValueChange={(v) => onChange(Number(v))}>
			<SelectTrigger className="w-20 h-8 text-xs">
				<SelectValue />
			</SelectTrigger>
			<SelectContent>
				{options.map((opt) => (
					<SelectItem key={opt} value={String(opt)}>
						{opt}
					</SelectItem>
				))}
			</SelectContent>
		</Select>
	);
}
