import { useCallback, useMemo } from "react";
import { Check, Minus } from "lucide-react";
import { cn } from "@/lib/utils";
import { ALL_CAPS, CAP_DOMAINS } from "@/lib/capabilities";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";

interface CapabilitySelectorProps {
	selected: string[];
	onChange: (caps: string[]) => void;
	disabled?: boolean;
}

export function CapabilitySelector({ selected, onChange, disabled }: CapabilitySelectorProps) {
	const selectedSet = useMemo(() => new Set(selected), [selected]);

	const allSelected = ALL_CAPS.length > 0 && ALL_CAPS.every((c) => selectedSet.has(c));

	const toggleAll = useCallback(() => {
		onChange(allSelected ? [] : [...ALL_CAPS]);
	}, [allSelected, onChange]);

	const toggleCap = useCallback(
		(cap: string) => {
			if (selectedSet.has(cap)) {
				onChange(selected.filter((c) => c !== cap));
			} else {
				onChange([...selected, cap]);
			}
		},
		[selected, selectedSet, onChange],
	);

	const toggleDomain = useCallback(
		(caps: string[]) => {
			const allDomainSelected = caps.every((c) => selectedSet.has(c));
			if (allDomainSelected) {
				onChange(selected.filter((c) => !caps.includes(c)));
			} else {
				const newCaps = new Set(selected);
				for (const c of caps) newCaps.add(c);
				onChange([...newCaps]);
			}
		},
		[selected, selectedSet, onChange],
	);

	return (
		<div className="space-y-3">
			{/* Header with Select All / Deselect All */}
			{!disabled && (
				<div className="flex items-center justify-between">
					<span className="text-xs text-muted-foreground">
						{selected.length} of {ALL_CAPS.length} selected
					</span>
					<Button
						type="button"
						variant="ghost"
						size="sm"
						className="h-7 text-xs"
						onClick={toggleAll}
					>
						{allSelected ? "Deselect All" : "Select All"}
					</Button>
				</div>
			)}

			{/* Domain rows */}
			<div className="rounded-lg border border-border divide-y divide-border">
				{CAP_DOMAINS.map(({ domain, caps }) => {
					const allChecked = caps.every((c) => selectedSet.has(c));
					const someChecked = !allChecked && caps.some((c) => selectedSet.has(c));

					return (
						<div
							key={domain}
							className="flex items-center gap-3 px-3 py-2 hover:bg-muted/30 transition-colors"
						>
							{/* Domain checkbox (selects all caps in this domain) */}
							<button
								type="button"
								disabled={disabled}
								onClick={() => toggleDomain(caps)}
								className={cn(
									"flex size-4 shrink-0 items-center justify-center rounded-[4px] border border-input shadow-xs transition-shadow",
									disabled && "cursor-not-allowed opacity-50",
									!disabled && "cursor-pointer",
									allChecked && "border-primary bg-primary text-primary-foreground",
									someChecked && "border-primary bg-primary/20",
								)}
							>
								{allChecked && <Check className="size-3" />}
								{someChecked && <Minus className="size-3 text-primary" />}
							</button>
							<span className="w-28 text-xs font-medium shrink-0">{domain}</span>

							{/* Individual capabilities */}
							<div className="flex flex-1 flex-wrap gap-x-4 gap-y-1">
								{caps.map((cap) => {
									const action = cap.split(".")[1];
									return (
										<Label
											key={cap}
											className={cn(
												"flex items-center gap-1.5 text-xs font-normal",
												disabled ? "cursor-default" : "cursor-pointer",
											)}
										>
											<Checkbox
												checked={selectedSet.has(cap)}
												onCheckedChange={() => toggleCap(cap)}
												disabled={disabled}
											/>
											<span className="text-muted-foreground">{action}</span>
										</Label>
									);
								})}
							</div>
						</div>
					);
				})}
			</div>
		</div>
	);
}
