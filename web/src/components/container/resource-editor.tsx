import { useCallback, useEffect, useState } from "react";
import { Cpu, HardDrive, Users, Save } from "lucide-react";
import { toast } from "sonner";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Slider } from "@/components/ui/slider";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { getAccessToken } from "@/lib/api";
import type { ContainerRecord } from "@/types/pocketbase";

interface ResourceEditorProps {
	container: ContainerRecord | undefined;
	isLoading: boolean;
}

interface ResourceLimits {
	cpu_limit: number;
	mem_limit: number;
	pids_limit: number;
}

function parseResourceLimits(raw: unknown): ResourceLimits {
	const defaults: ResourceLimits = { cpu_limit: 0, mem_limit: 0, pids_limit: 0 };
	if (raw == null) return defaults;

	let obj: Record<string, unknown>;
	if (typeof raw === "string") {
		try {
			obj = JSON.parse(raw) as Record<string, unknown>;
		} catch {
			return defaults;
		}
	} else if (typeof raw === "object") {
		obj = raw as Record<string, unknown>;
	} else {
		return defaults;
	}

	return {
		cpu_limit: Number(obj.cpu_limit) || 0,
		mem_limit: Number(obj.mem_limit) || 0,
		pids_limit: Number(obj.pids_limit) || 0,
	};
}

function formatBytes(bytes: number): string {
	if (bytes === 0) return "No limit";
	const mb = bytes / (1024 * 1024);
	if (mb >= 1024) return `${(mb / 1024).toFixed(1)} GB`;
	return `${Math.round(mb)} MB`;
}

function mbToBytes(mb: number): number {
	return mb * 1024 * 1024;
}

function bytesToMb(bytes: number): number {
	return Math.round(bytes / (1024 * 1024));
}

export function ResourceEditor({ container, isLoading }: ResourceEditorProps) {
	const [cpu, setCpu] = useState(0);
	const [memMb, setMemMb] = useState(0);
	const [pids, setPids] = useState(0);
	const [saving, setSaving] = useState(false);
	const [dirty, setDirty] = useState(false);

	// Sync from container record.
	useEffect(() => {
		if (!container) return;
		const limits = parseResourceLimits(container.resource_limits);
		setCpu(limits.cpu_limit);
		setMemMb(bytesToMb(limits.mem_limit));
		setPids(limits.pids_limit);
		setDirty(false);
	}, [container]);

	const handleSave = useCallback(async () => {
		if (!container) return;
		setSaving(true);
		try {
			const token = getAccessToken();
			const res = await fetch(`/api/dk/containers/${container.id}/resources`, {
				method: "PUT",
				headers: {
					"Content-Type": "application/json",
					...(token ? { Authorization: token } : {}),
				},
				body: JSON.stringify({
					cpu_limit: cpu,
					mem_limit: mbToBytes(memMb),
					pids_limit: pids,
				}),
			});

			if (!res.ok) {
				const body = await res.json().catch(() => ({ error: res.statusText }));
				throw new Error(body.error || "Failed to update resources");
			}

			toast.success("Resource limits updated (applied without restart)");
			setDirty(false);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : "Failed to update resources");
		} finally {
			setSaving(false);
		}
	}, [container, cpu, memMb, pids]);

	if (isLoading || !container) {
		return (
			<Card>
				<CardHeader>
					<Skeleton className="h-5 w-40" />
				</CardHeader>
				<CardContent className="space-y-6">
					{[1, 2, 3].map((i) => (
						<div key={i} className="space-y-2">
							<Skeleton className="h-4 w-24" />
							<Skeleton className="h-8 w-full" />
						</div>
					))}
				</CardContent>
			</Card>
		);
	}

	return (
		<Card>
			<CardHeader className="flex flex-row items-center justify-between">
				<CardTitle className="text-sm">Resource Limits</CardTitle>
				<Button size="sm" onClick={handleSave} disabled={!dirty || saving}>
					<Save className="mr-1.5 h-3.5 w-3.5" />
					{saving ? "Applying..." : "Apply"}
				</Button>
			</CardHeader>
			<CardContent className="space-y-6">
				{/* CPU Limit */}
				<div className="space-y-3">
					<div className="flex items-center justify-between">
						<label className="flex items-center gap-2 text-sm font-medium">
							<Cpu className="h-4 w-4 text-muted-foreground" />
							CPU Limit
						</label>
						<span className="text-sm text-muted-foreground">
							{cpu === 0 ? "No limit" : `${cpu} core${cpu !== 1 ? "s" : ""}`}
						</span>
					</div>
					<div className="flex items-center gap-3">
						<Slider
							value={[cpu]}
							onValueChange={([v]) => {
								setCpu(v);
								setDirty(true);
							}}
							min={0}
							max={16}
							step={0.1}
							className="flex-1"
						/>
						<Input
							type="number"
							value={cpu}
							onChange={(e) => {
								setCpu(parseFloat(e.target.value) || 0);
								setDirty(true);
							}}
							className="w-20 h-8 text-xs"
							min={0}
							max={64}
							step={0.1}
						/>
					</div>
				</div>

				{/* Memory Limit */}
				<div className="space-y-3">
					<div className="flex items-center justify-between">
						<label className="flex items-center gap-2 text-sm font-medium">
							<HardDrive className="h-4 w-4 text-muted-foreground" />
							Memory Limit
						</label>
						<span className="text-sm text-muted-foreground">{formatBytes(mbToBytes(memMb))}</span>
					</div>
					<div className="flex items-center gap-3">
						<Slider
							value={[memMb]}
							onValueChange={([v]) => {
								setMemMb(v);
								setDirty(true);
							}}
							min={0}
							max={32768}
							step={64}
							className="flex-1"
						/>
						<Input
							type="number"
							value={memMb}
							onChange={(e) => {
								setMemMb(parseInt(e.target.value) || 0);
								setDirty(true);
							}}
							className="w-20 h-8 text-xs"
							min={0}
							step={64}
							placeholder="MB"
						/>
					</div>
				</div>

				{/* PIDs Limit */}
				<div className="space-y-3">
					<div className="flex items-center justify-between">
						<label className="flex items-center gap-2 text-sm font-medium">
							<Users className="h-4 w-4 text-muted-foreground" />
							PIDs Limit
						</label>
						<span className="text-sm text-muted-foreground">{pids === 0 ? "No limit" : pids}</span>
					</div>
					<div className="flex items-center gap-3">
						<Slider
							value={[pids]}
							onValueChange={([v]) => {
								setPids(v);
								setDirty(true);
							}}
							min={0}
							max={65536}
							step={100}
							className="flex-1"
						/>
						<Input
							type="number"
							value={pids}
							onChange={(e) => {
								setPids(parseInt(e.target.value) || 0);
								setDirty(true);
							}}
							className="w-20 h-8 text-xs"
							min={0}
							max={65536}
						/>
					</div>
				</div>

				<p className="text-xs text-muted-foreground">
					Resource limits are applied immediately without restarting the container (Docker
					hot-update). Set to 0 to remove the limit.
				</p>
			</CardContent>
		</Card>
	);
}
