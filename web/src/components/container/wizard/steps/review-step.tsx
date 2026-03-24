import { useFormContext } from "react-hook-form";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import type { ContainerWizardData } from "../schema";

function Section({ title, children }: { title: string; children: React.ReactNode }) {
	return (
		<div className="flex flex-col gap-2">
			<h4 className="text-sm font-medium text-muted-foreground">{title}</h4>
			{children}
		</div>
	);
}

function Field({ label, value }: { label: string; value: React.ReactNode }) {
	if (!value) return null;
	return (
		<div className="flex items-baseline justify-between gap-4">
			<span className="text-sm text-muted-foreground">{label}</span>
			<span className="text-sm font-medium text-right">{value}</span>
		</div>
	);
}

export function ReviewStep() {
	const { getValues } = useFormContext<ContainerWizardData>();
	const data = getValues();

	return (
		<div className="flex flex-col gap-5">
			<p className="text-sm text-muted-foreground">
				Review the configuration before creating the container.
			</p>

			{/* Image */}
			<Section title="Image">
				<Field label="Image" value={<Badge variant="secondary">{data.image}</Badge>} />
				<Field label="Auto start" value={data.auto_start ? "Yes" : "No"} />
			</Section>

			<Separator />

			{/* General */}
			<Section title="General">
				<Field label="Name" value={data.name} />
				{data.network && <Field label="Network" value={data.network} />}
				{data.restart_policy && <Field label="Restart policy" value={data.restart_policy} />}
				{data.command && (
					<Field label="Command" value={<code className="text-xs">{data.command}</code>} />
				)}
				{data.entrypoint && (
					<Field label="Entrypoint" value={<code className="text-xs">{data.entrypoint}</code>} />
				)}
			</Section>

			{/* Ports */}
			{data.ports.length > 0 && (
				<>
					<Separator />
					<Section title={`Ports (${data.ports.length})`}>
						<div className="flex flex-wrap gap-1.5">
							{data.ports.map((p, i) => (
								<Badge key={i} variant="outline" className="font-mono text-xs">
									{p.host_port || "*"}:{p.container_port}/{p.protocol || "tcp"}
								</Badge>
							))}
						</div>
					</Section>
				</>
			)}

			{/* Volumes */}
			{data.volumes.length > 0 && (
				<>
					<Separator />
					<Section title={`Volumes (${data.volumes.length})`}>
						{data.volumes.map((v, i) => (
							<div key={i} className="flex items-center gap-2 text-xs font-mono">
								<span>{v.source || "(anonymous)"}</span>
								<span className="text-muted-foreground">→</span>
								<span>{v.target}</span>
								{v.read_only && (
									<Badge variant="outline" className="text-[10px]">
										RO
									</Badge>
								)}
							</div>
						))}
					</Section>
				</>
			)}

			{/* Env */}
			{data.env.length > 0 && (
				<>
					<Separator />
					<Section title={`Environment Variables (${data.env.length})`}>
						{data.env.map((e, i) => (
							<div key={i} className="flex items-baseline gap-2 text-xs font-mono">
								<span className="font-semibold">{e.key}</span>
								<span className="text-muted-foreground">=</span>
								<span className="truncate">{e.value}</span>
							</div>
						))}
					</Section>
				</>
			)}

			{/* Labels */}
			{data.labels.length > 0 && (
				<>
					<Separator />
					<Section title={`Labels (${data.labels.length})`}>
						{data.labels.map((l, i) => (
							<div key={i} className="flex items-baseline gap-2 text-xs font-mono">
								<span className="font-semibold">{l.key}</span>
								<span className="text-muted-foreground">=</span>
								<span className="truncate">{l.value}</span>
							</div>
						))}
					</Section>
				</>
			)}

			{/* Resources */}
			{(data.cpu_limit > 0 || data.mem_limit > 0 || data.pids_limit > 0) && (
				<>
					<Separator />
					<Section title="Resources">
						{data.cpu_limit > 0 && <Field label="CPU" value={`${data.cpu_limit} core(s)`} />}
						{data.mem_limit > 0 && <Field label="Memory" value={`${data.mem_limit} MB`} />}
						{data.pids_limit > 0 && <Field label="PIDs" value={`${data.pids_limit}`} />}
					</Section>
				</>
			)}
		</div>
	);
}
