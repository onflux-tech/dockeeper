import { useFormContext, useFieldArray } from "react-hook-form";
import { Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { ContainerWizardData } from "../schema";

export function EnvStep() {
	const {
		register,
		control,
		formState: { errors },
	} = useFormContext<ContainerWizardData>();

	const { fields, append, remove } = useFieldArray({ control, name: "env" });

	return (
		<div className="flex flex-col gap-4">
			<p className="text-sm text-muted-foreground">
				Define environment variables that will be available inside the container.
			</p>

			{fields.length > 0 && (
				<div className="grid grid-cols-[1fr_1fr_40px] gap-2 text-xs font-medium text-muted-foreground">
					<span>Key *</span>
					<span>Value</span>
					<span />
				</div>
			)}

			{fields.map((field, index) => (
				<div key={field.id} className="grid grid-cols-[1fr_1fr_40px] items-start gap-2">
					<div className="flex flex-col gap-1">
						<Input
							placeholder="MY_VAR"
							aria-invalid={!!errors.env?.[index]?.key}
							{...register(`env.${index}.key`)}
						/>
						{errors.env?.[index]?.key && (
							<p className="text-xs text-destructive">{errors.env[index].key.message}</p>
						)}
					</div>
					<Input placeholder="value" {...register(`env.${index}.value`)} />
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
				onClick={() => append({ key: "", value: "" })}
			>
				<Plus className="mr-1.5 h-3.5 w-3.5" />
				Add Variable
			</Button>
		</div>
	);
}
