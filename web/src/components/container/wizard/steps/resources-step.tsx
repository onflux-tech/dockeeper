import { useFormContext } from "react-hook-form";
import { Cpu, MemoryStick, Users } from "lucide-react";
import { Input } from "@/components/ui/input";
import type { ContainerWizardData } from "../schema";

export function ResourcesStep() {
	const {
		register,
		watch,
		formState: { errors },
	} = useFormContext<ContainerWizardData>();

	const cpuLimit = watch("cpu_limit");
	const memLimit = watch("mem_limit");
	const pidsLimit = watch("pids_limit");

	return (
		<div className="flex flex-col gap-6">
			<p className="text-sm text-muted-foreground">
				Set resource constraints for the container. Leave at 0 for unlimited.
			</p>

			{/* CPU Limit */}
			<div className="flex flex-col gap-2">
				<div className="flex items-center gap-2">
					<Cpu className="h-4 w-4 text-muted-foreground" />
					<label htmlFor="cpu_limit" className="text-sm font-medium">
						CPU Limit
					</label>
					<span className="ml-auto text-sm text-muted-foreground">
						{cpuLimit ? `${cpuLimit} core(s)` : "Unlimited"}
					</span>
				</div>
				<Input
					id="cpu_limit"
					type="number"
					step="0.1"
					min="0"
					max="128"
					placeholder="0"
					aria-invalid={!!errors.cpu_limit}
					{...register("cpu_limit", { valueAsNumber: true })}
				/>
				{errors.cpu_limit && (
					<p className="text-xs text-destructive">{errors.cpu_limit.message}</p>
				)}
				<p className="text-xs text-muted-foreground">
					Number of CPU cores (e.g. 0.5 = half a core, 2 = two cores)
				</p>
			</div>

			{/* Memory Limit */}
			<div className="flex flex-col gap-2">
				<div className="flex items-center gap-2">
					<MemoryStick className="h-4 w-4 text-muted-foreground" />
					<label htmlFor="mem_limit" className="text-sm font-medium">
						Memory Limit
					</label>
					<span className="ml-auto text-sm text-muted-foreground">
						{memLimit ? `${memLimit} MB` : "Unlimited"}
					</span>
				</div>
				<Input
					id="mem_limit"
					type="number"
					step="1"
					min="0"
					max="131072"
					placeholder="0"
					aria-invalid={!!errors.mem_limit}
					{...register("mem_limit", { valueAsNumber: true })}
				/>
				{errors.mem_limit && (
					<p className="text-xs text-destructive">{errors.mem_limit.message}</p>
				)}
				<p className="text-xs text-muted-foreground">
					Memory in megabytes (e.g. 512 = 512 MB, 2048 = 2 GB)
				</p>
			</div>

			{/* PIDs Limit */}
			<div className="flex flex-col gap-2">
				<div className="flex items-center gap-2">
					<Users className="h-4 w-4 text-muted-foreground" />
					<label htmlFor="pids_limit" className="text-sm font-medium">
						PIDs Limit
					</label>
					<span className="ml-auto text-sm text-muted-foreground">
						{pidsLimit ? `${pidsLimit} processes` : "Unlimited"}
					</span>
				</div>
				<Input
					id="pids_limit"
					type="number"
					step="1"
					min="0"
					max="32768"
					placeholder="0"
					aria-invalid={!!errors.pids_limit}
					{...register("pids_limit", { valueAsNumber: true })}
				/>
				{errors.pids_limit && (
					<p className="text-xs text-destructive">{errors.pids_limit.message}</p>
				)}
				<p className="text-xs text-muted-foreground">
					Maximum number of processes. Protects against fork bombs.
				</p>
			</div>
		</div>
	);
}
