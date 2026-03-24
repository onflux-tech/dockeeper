import { useFormContext } from "react-hook-form";
import { Input } from "@/components/ui/input";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { useCollection } from "@/hooks/use-collection";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import type { ContainerWizardData } from "../schema";

const RESTART_POLICIES = [
	{ value: "", label: "None (default)" },
	{ value: "no", label: "No" },
	{ value: "always", label: "Always" },
	{ value: "unless-stopped", label: "Unless Stopped" },
	{ value: "on-failure", label: "On Failure" },
] as const;

export function GeneralStep() {
	const {
		register,
		setValue,
		watch,
		formState: { errors },
	} = useFormContext<ContainerWizardData>();

	const serverId = watch("server_id");
	const restartPolicy = watch("restart_policy");

	const { data: serversResult, isLoading: serversLoading } = useCollection("servers", {
		perPage: 200,
		sort: "name",
	});

	const { data: networksResult } = useCollection("networks", {
		perPage: 200,
		sort: "name",
		filter: serverId ? `server = "${serverId}"` : undefined,
		enabled: !!serverId,
	});

	const servers = serversResult?.items ?? [];
	const networks = networksResult?.items ?? [];

	return (
		<div className="flex flex-col gap-5">
			{/* Server */}
			<div className="flex flex-col gap-1.5">
				<label className="text-sm font-medium">
					Server <span className="text-destructive">*</span>
				</label>
				{serversLoading ? (
					<Skeleton className="h-9 w-full" />
				) : (
					<Select value={serverId} onValueChange={(v) => setValue("server_id", v)}>
						<SelectTrigger className="w-full">
							<SelectValue placeholder="Select a server" />
						</SelectTrigger>
						<SelectContent>
							{servers.map((s) => (
								<SelectItem key={s.id} value={s.id}>
									<span className="flex items-center gap-2">
										{s.name}
										<Badge
											variant={s.status === "online" ? "default" : "secondary"}
											className="text-[10px] px-1.5 py-0"
										>
											{s.status}
										</Badge>
									</span>
								</SelectItem>
							))}
						</SelectContent>
					</Select>
				)}
				{errors.server_id && <p className="text-xs text-destructive">{errors.server_id.message}</p>}
			</div>

			{/* Name */}
			<div className="flex flex-col gap-1.5">
				<label htmlFor="name" className="text-sm font-medium">
					Container Name <span className="text-destructive">*</span>
				</label>
				<Input
					id="name"
					placeholder="my-container"
					aria-invalid={!!errors.name}
					{...register("name")}
				/>
				{errors.name && <p className="text-xs text-destructive">{errors.name.message}</p>}
			</div>

			{/* Network */}
			<div className="flex flex-col gap-1.5">
				<label className="text-sm font-medium">Network</label>
				<Select
					value={watch("network") || "__default__"}
					onValueChange={(v) => setValue("network", v === "__default__" ? "" : v)}
				>
					<SelectTrigger className="w-full">
						<SelectValue placeholder="Default (bridge)" />
					</SelectTrigger>
					<SelectContent>
						<SelectItem value="__default__">Default (bridge)</SelectItem>
						{networks.map((n) => (
							<SelectItem key={n.id} value={n.name}>
								{n.name} ({n.driver})
							</SelectItem>
						))}
					</SelectContent>
				</Select>
			</div>

			{/* Restart Policy */}
			<div className="flex flex-col gap-1.5">
				<label className="text-sm font-medium">Restart Policy</label>
				<Select
					value={restartPolicy || "__none__"}
					onValueChange={(v) =>
						setValue(
							"restart_policy",
							(v === "__none__" ? "" : v) as ContainerWizardData["restart_policy"],
						)
					}
				>
					<SelectTrigger className="w-full">
						<SelectValue placeholder="None" />
					</SelectTrigger>
					<SelectContent>
						{RESTART_POLICIES.map((p) => (
							<SelectItem key={p.value || "__none__"} value={p.value || "__none__"}>
								{p.label}
							</SelectItem>
						))}
					</SelectContent>
				</Select>
			</div>

			<div className="grid grid-cols-1 gap-5 sm:grid-cols-2">
				{/* Command */}
				<div className="flex flex-col gap-1.5">
					<label htmlFor="command" className="text-sm font-medium">
						Command
					</label>
					<Input id="command" placeholder="e.g. /bin/sh -c 'echo hi'" {...register("command")} />
					<p className="text-xs text-muted-foreground">Override the default CMD</p>
				</div>

				{/* Entrypoint */}
				<div className="flex flex-col gap-1.5">
					<label htmlFor="entrypoint" className="text-sm font-medium">
						Entrypoint
					</label>
					<Input
						id="entrypoint"
						placeholder="e.g. /docker-entrypoint.sh"
						{...register("entrypoint")}
					/>
					<p className="text-xs text-muted-foreground">Override the default ENTRYPOINT</p>
				</div>
			</div>
		</div>
	);
}
