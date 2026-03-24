import { useCallback, useEffect, useMemo, useState } from "react";
import {
	Eye,
	EyeOff,
	Plus,
	Save,
	Trash2,
	AlertTriangle,
} from "lucide-react";
import { toast } from "sonner";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { getAccessToken } from "@/lib/api";
import type { ContainerRecord } from "@/types/pocketbase";

interface EnvEditorProps {
	container: ContainerRecord | undefined;
	isLoading: boolean;
}

interface EnvEntry {
	key: string;
	value: string;
}

function parseEnvVars(raw: unknown): EnvEntry[] {
	if (raw == null) return [];
	let items: string[] = [];

	if (Array.isArray(raw)) {
		items = raw.map(String);
	} else if (typeof raw === "string") {
		try {
			const parsed = JSON.parse(raw);
			if (Array.isArray(parsed)) items = parsed.map(String);
		} catch {
			return [];
		}
	} else if (typeof raw === "object") {
		items = Object.entries(raw as Record<string, unknown>).map(
			([k, v]) => `${k}=${String(v ?? "")}`,
		);
	}

	return items.map((s) => {
		const idx = s.indexOf("=");
		if (idx < 0) return { key: s, value: "" };
		return { key: s.substring(0, idx), value: s.substring(idx + 1) };
	});
}

interface DiffItem {
	key: string;
	type: "added" | "removed" | "changed" | "unchanged";
	oldValue?: string;
	newValue?: string;
}

function computeDiff(original: EnvEntry[], current: EnvEntry[]): DiffItem[] {
	const origMap = new Map(original.map((e) => [e.key, e.value]));
	const currMap = new Map(current.map((e) => [e.key, e.value]));
	const items: DiffItem[] = [];

	for (const [key, value] of currMap) {
		if (!origMap.has(key)) {
			items.push({ key, type: "added", newValue: value });
		} else if (origMap.get(key) !== value) {
			items.push({ key, type: "changed", oldValue: origMap.get(key), newValue: value });
		}
	}

	for (const [key, value] of origMap) {
		if (!currMap.has(key)) {
			items.push({ key, type: "removed", oldValue: value });
		}
	}

	return items;
}

export function EnvEditor({ container, isLoading }: EnvEditorProps) {
	const [entries, setEntries] = useState<EnvEntry[]>([]);
	const [original, setOriginal] = useState<EnvEntry[]>([]);
	const [revealed, setRevealed] = useState<Set<number>>(new Set());
	const [confirmOpen, setConfirmOpen] = useState(false);
	const [saving, setSaving] = useState(false);

	useEffect(() => {
		if (!container) return;
		const parsed = parseEnvVars(container.env_vars);
		setOriginal(parsed);
		setEntries(parsed.map((e) => ({ ...e })));
		setRevealed(new Set());
	}, [container]);

	const diff = useMemo(() => computeDiff(original, entries), [original, entries]);
	const hasChanges = diff.length > 0;

	const toggleReveal = useCallback((idx: number) => {
		setRevealed((prev) => {
			const next = new Set(prev);
			if (next.has(idx)) next.delete(idx);
			else next.add(idx);
			return next;
		});
	}, []);

	const updateEntry = useCallback((idx: number, field: "key" | "value", val: string) => {
		setEntries((prev) => prev.map((e, i) => (i === idx ? { ...e, [field]: val } : e)));
	}, []);

	const removeEntry = useCallback((idx: number) => {
		setEntries((prev) => prev.filter((_, i) => i !== idx));
		setRevealed((prev) => {
			const next = new Set<number>();
			for (const v of prev) {
				if (v < idx) next.add(v);
				else if (v > idx) next.add(v - 1);
			}
			return next;
		});
	}, []);

	const addEntry = useCallback(() => {
		setEntries((prev) => [...prev, { key: "", value: "" }]);
	}, []);

	const handleSave = useCallback(async () => {
		if (!container) return;
		setConfirmOpen(false);
		setSaving(true);

		try {
			const envMap: Record<string, string> = {};
			for (const e of entries) {
				if (e.key.trim()) envMap[e.key.trim()] = e.value;
			}

			const token = getAccessToken();
			const res = await fetch(`/api/dk/containers/${container.id}/env`, {
				method: "PUT",
				headers: {
					"Content-Type": "application/json",
					...(token ? { Authorization: token } : {}),
				},
				body: JSON.stringify({ env: envMap }),
			});

			if (!res.ok) {
				const body = await res.json().catch(() => ({ error: res.statusText }));
				throw new Error(body.error || "Failed to update environment");
			}

			toast.success("Environment updated — container was recreated");
		} catch (err) {
			toast.error(err instanceof Error ? err.message : "Failed to update environment");
		} finally {
			setSaving(false);
		}
	}, [container, entries]);

	if (isLoading || !container) {
		return (
			<Card>
				<CardHeader>
					<Skeleton className="h-5 w-48" />
				</CardHeader>
				<CardContent className="space-y-2">
					{[1, 2, 3].map((i) => (
						<Skeleton key={i} className="h-8 w-full" />
					))}
				</CardContent>
			</Card>
		);
	}

	return (
		<>
			<Card>
				<CardHeader className="flex flex-row items-center justify-between">
					<CardTitle className="text-sm">Environment Variables</CardTitle>
					<div className="flex items-center gap-2">
						<Button variant="outline" size="sm" onClick={addEntry}>
							<Plus className="mr-1.5 h-3.5 w-3.5" />
							Add
						</Button>
						<Button
							size="sm"
							onClick={() => setConfirmOpen(true)}
							disabled={!hasChanges || saving}
						>
							<Save className="mr-1.5 h-3.5 w-3.5" />
							{saving ? "Saving..." : "Apply"}
						</Button>
					</div>
				</CardHeader>
				<CardContent>
					{entries.length === 0 ? (
						<p className="text-sm text-muted-foreground">No environment variables set.</p>
					) : (
						<div className="space-y-2">
							<div className="grid grid-cols-[1fr_1fr_auto] gap-2 text-xs font-medium text-muted-foreground px-1">
								<span>Key</span>
								<span>Value</span>
								<span className="w-16" />
							</div>
							{entries.map((entry, idx) => (
								<div key={idx} className="grid grid-cols-[1fr_1fr_auto] gap-2 items-center">
									<Input
										value={entry.key}
										onChange={(e) => updateEntry(idx, "key", e.target.value)}
										className="h-8 text-xs font-mono"
										placeholder="KEY"
									/>
									<div className="relative">
										<Input
											type={revealed.has(idx) ? "text" : "password"}
											value={entry.value}
											onChange={(e) => updateEntry(idx, "value", e.target.value)}
											className="h-8 text-xs font-mono pr-8"
											placeholder="value"
										/>
										<button
											type="button"
											onClick={() => toggleReveal(idx)}
											className="absolute right-1.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
										>
											{revealed.has(idx) ? (
												<EyeOff className="h-3.5 w-3.5" />
											) : (
												<Eye className="h-3.5 w-3.5" />
											)}
										</button>
									</div>
									<Button
										variant="ghost"
										size="icon"
										className="h-8 w-8"
										onClick={() => removeEntry(idx)}
									>
										<Trash2 className="h-3.5 w-3.5 text-destructive" />
									</Button>
								</div>
							))}
						</div>
					)}

					<p className="mt-4 text-xs text-muted-foreground">
						Changing environment variables requires the container to be recreated.
						The container will be stopped, removed, and re-created with the same
						configuration but updated environment.
					</p>
				</CardContent>
			</Card>

			{/* Diff preview + recreate confirmation dialog */}
			<Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
				<DialogContent>
					<DialogHeader>
						<DialogTitle className="flex items-center gap-2">
							<AlertTriangle className="h-5 w-5 text-yellow-500" />
							Confirm Environment Update
						</DialogTitle>
						<DialogDescription>
							This will recreate the container with updated environment
							variables. The container will be briefly stopped.
						</DialogDescription>
					</DialogHeader>

					{diff.length > 0 && (
						<div className="space-y-1.5 max-h-60 overflow-y-auto rounded-md border p-3">
							{diff.map((item) => (
								<div key={item.key} className="flex items-center gap-2 text-xs font-mono">
									<Badge
										variant={
											item.type === "added"
												? "default"
												: item.type === "removed"
													? "destructive"
													: "secondary"
										}
										className="text-[10px] w-16 justify-center"
									>
										{item.type}
									</Badge>
									<span className="font-semibold">{item.key}</span>
									{item.type === "changed" && (
										<>
											<span className="text-destructive line-through truncate max-w-32">
												{item.oldValue}
											</span>
											<span className="text-muted-foreground">&rarr;</span>
											<span className="text-green-500 truncate max-w-32">
												{item.newValue}
											</span>
										</>
									)}
								</div>
							))}
						</div>
					)}

					<DialogFooter>
						<Button variant="outline" onClick={() => setConfirmOpen(false)}>
							Cancel
						</Button>
						<Button onClick={handleSave} disabled={saving}>
							{saving ? "Recreating..." : "Recreate Container"}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</>
	);
}
