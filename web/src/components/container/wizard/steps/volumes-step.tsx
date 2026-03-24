import { useFormContext, useFieldArray } from "react-hook-form";
import { Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Checkbox } from "@/components/ui/checkbox";
import type { ContainerWizardData } from "../schema";

export function VolumesStep() {
	const {
		register,
		control,
		setValue,
		watch,
		formState: { errors },
	} = useFormContext<ContainerWizardData>();

	const { fields, append, remove } = useFieldArray({ control, name: "volumes" });

	return (
		<div className="flex flex-col gap-4">
			<p className="text-sm text-muted-foreground">
				Bind-mount host paths or named volumes into the container.
			</p>

			{fields.length > 0 && (
				<div className="grid grid-cols-[1fr_1fr_70px_40px] gap-2 text-xs font-medium text-muted-foreground">
					<span>Source</span>
					<span>Mount Path *</span>
					<span>RO</span>
					<span />
				</div>
			)}

			{fields.map((field, index) => (
				<div key={field.id} className="grid grid-cols-[1fr_1fr_70px_40px] items-start gap-2">
					<Input placeholder="/host/data or vol_name" {...register(`volumes.${index}.source`)} />
					<div className="flex flex-col gap-1">
						<Input
							placeholder="/container/data"
							aria-invalid={!!errors.volumes?.[index]?.target}
							{...register(`volumes.${index}.target`)}
						/>
						{errors.volumes?.[index]?.target && (
							<p className="text-xs text-destructive">{errors.volumes[index].target.message}</p>
						)}
					</div>
					<div className="flex h-9 items-center justify-center">
						<Checkbox
							checked={watch(`volumes.${index}.read_only`)}
							onCheckedChange={(checked) => setValue(`volumes.${index}.read_only`, !!checked)}
						/>
					</div>
					<Button
						type="button"
						variant="ghost"
						size="icon"
						className="text-muted-foreground hover:text-destructive"
						onClick={() => remove(index)}
					>
						<Trash2 className="h-4 w-4" />
					</Button>
				</div>
			))}

			<Button
				type="button"
				variant="outline"
				size="sm"
				className="w-fit"
				onClick={() => append({ source: "", target: "", read_only: false })}
			>
				<Plus className="mr-1.5 h-3.5 w-3.5" />
				Add Volume
			</Button>
		</div>
	);
}
