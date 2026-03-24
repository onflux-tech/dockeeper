import { useFormContext } from "react-hook-form";
import { Box } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Checkbox } from "@/components/ui/checkbox";
import type { ContainerWizardData } from "../schema";

export function ImageStep() {
	const {
		register,
		setValue,
		watch,
		formState: { errors },
	} = useFormContext<ContainerWizardData>();

	const autoStart = watch("auto_start");

	return (
		<div className="flex flex-col gap-6">
			<div className="flex flex-col gap-1.5">
				<label htmlFor="image" className="text-sm font-medium">
					Image <span className="text-destructive">*</span>
				</label>
				<div className="relative">
					<Box className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
					<Input
						id="image"
						placeholder="e.g. nginx:latest, postgres:16-alpine"
						className="pl-9"
						aria-invalid={!!errors.image}
						{...register("image")}
					/>
				</div>
				{errors.image && <p className="text-xs text-destructive">{errors.image.message}</p>}
				<p className="text-xs text-muted-foreground">
					Docker image name with optional tag. The image will be pulled automatically if not
					present.
				</p>
			</div>

			<div className="flex items-center gap-3">
				<Checkbox
					id="auto_start"
					checked={autoStart}
					onCheckedChange={(checked) => setValue("auto_start", !!checked)}
				/>
				<div className="flex flex-col">
					<label htmlFor="auto_start" className="text-sm font-medium cursor-pointer">
						Start container after creation
					</label>
					<p className="text-xs text-muted-foreground">
						When enabled, the container will start immediately after being created.
					</p>
				</div>
			</div>
		</div>
	);
}
