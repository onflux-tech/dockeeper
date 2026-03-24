import { useFormContext, useFieldArray } from "react-hook-form";
import { Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import type { ContainerWizardData } from "../schema";

export function PortsStep() {
	const {
		register,
		control,
		setValue,
		watch,
		formState: { errors },
	} = useFormContext<ContainerWizardData>();

	const { fields, append, remove } = useFieldArray({ control, name: "ports" });

	return (
		<div className="flex flex-col gap-4">
			<p className="text-sm text-muted-foreground">
				Map host ports to container ports. Leave host port empty for a random assignment.
			</p>

			{fields.length > 0 && (
				<div className="grid grid-cols-[1fr_1fr_100px_40px] gap-2 text-xs font-medium text-muted-foreground">
					<span>Host Port</span>
					<span>Container Port *</span>
					<span>Protocol</span>
					<span />
				</div>
			)}

			{fields.map((field, index) => (
				<div key={field.id} className="grid grid-cols-[1fr_1fr_100px_40px] items-start gap-2">
					<Input placeholder="8080" {...register(`ports.${index}.host_port`)} />
					<div className="flex flex-col gap-1">
						<Input
							placeholder="80"
							aria-invalid={!!errors.ports?.[index]?.container_port}
							{...register(`ports.${index}.container_port`)}
						/>
						{errors.ports?.[index]?.container_port && (
							<p className="text-xs text-destructive">
								{errors.ports[index].container_port.message}
							</p>
						)}
					</div>
					<Select
						value={watch(`ports.${index}.protocol`) || "tcp"}
						onValueChange={(v) =>
							setValue(`ports.${index}.protocol`, v as "tcp" | "udp")
						}
					>
						<SelectTrigger className="w-full">
							<SelectValue />
						</SelectTrigger>
						<SelectContent>
							<SelectItem value="tcp">TCP</SelectItem>
							<SelectItem value="udp">UDP</SelectItem>
						</SelectContent>
					</Select>
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
				onClick={() => append({ host_port: "", container_port: "", protocol: "tcp" })}
			>
				<Plus className="mr-1.5 h-3.5 w-3.5" />
				Add Port
			</Button>
		</div>
	);
}
