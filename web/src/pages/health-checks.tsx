import { useMemo, useState } from "react";
import { useSearchParams } from "react-router";
import {
	HeartPulse,
	Plus,
	Trash2,
	Play,
	Pencil,
	Loader2,
	CheckCircle2,
	XCircle,
	HelpCircle,
	ScrollText,
	ChevronLeft,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Skeleton } from "@/components/ui/skeleton";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import {
	AlertDialog,
	AlertDialogAction,
	AlertDialogCancel,
	AlertDialogContent,
	AlertDialogDescription,
	AlertDialogFooter,
	AlertDialogHeader,
	AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { useCollection } from "@/hooks/use-collection";
import {
	useHealthChecks,
	useHealthCheckLogs,
	useCreateHealthCheck,
	useUpdateHealthCheck,
	useDeleteHealthCheck,
	useTriggerHealthCheck,
	type CreateHealthCheckInput,
} from "@/hooks/use-health-checks";
import type { HealthCheckRecord } from "@/types/pocketbase";
import { relativeTime } from "@/lib/format";
import { eqFilter } from "@/lib/pocketbase-filters";

// ── Status helpers ────────────────────────────────────────

const statusConfig: Record<string, { icon: typeof CheckCircle2; color: string; label: string }> = {
	healthy: { icon: CheckCircle2, color: "text-success", label: "Healthy" },
	unhealthy: { icon: XCircle, color: "text-destructive", label: "Unhealthy" },
	unknown: { icon: HelpCircle, color: "text-muted-foreground", label: "Unknown" },
};

function StatusBadge({ status }: { status: string }) {
	const cfg = statusConfig[status] ?? statusConfig.unknown;
	const Icon = cfg.icon;
	return (
		<Badge variant="outline" className="gap-1">
			<Icon className={`h-3 w-3 ${cfg.color}`} />
			{cfg.label}
		</Badge>
	);
}

// ── Main page ─────────────────────────────────────────────

export default function HealthChecksPage() {
	const [searchParams] = useSearchParams();
	const serverFilter = searchParams.get("server");
	const { data: allChecks, isLoading } = useHealthChecks();
	const checks = useMemo(
		() => (serverFilter ? allChecks?.filter((check) => check.server === serverFilter) : allChecks),
		[allChecks, serverFilter],
	);
	const [showCreate, setShowCreate] = useState(false);
	const [logsFor, setLogsFor] = useState<HealthCheckRecord | null>(null);
	const [editCheck, setEditCheck] = useState<HealthCheckRecord | null>(null);
	const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);
	const deleteMut = useDeleteHealthCheck();
	const triggerMut = useTriggerHealthCheck();

	const healthy = checks?.filter((c) => c.status === "healthy").length ?? 0;
	const unhealthy = checks?.filter((c) => c.status === "unhealthy").length ?? 0;
	const total = checks?.length ?? 0;

	return (
		<div className="flex flex-col gap-6">
			<div className="flex items-center justify-between">
				<div className="flex items-center gap-2">
					<HeartPulse className="h-5 w-5 text-muted-foreground" />
					<h1 className="text-2xl font-semibold">Health Checks</h1>
				</div>
				<Button size="sm" onClick={() => setShowCreate(true)}>
					<Plus className="mr-1.5 h-4 w-4" />
					New Check
				</Button>
			</div>

			{/* Summary cards */}
			<div className="grid grid-cols-3 gap-4">
				<Card>
					<CardContent className="flex items-center gap-3 py-0">
						<HeartPulse className="h-5 w-5 text-muted-foreground" />
						<div>
							<p className="text-2xl font-bold">{total}</p>
							<p className="text-xs text-muted-foreground">Total Checks</p>
						</div>
					</CardContent>
				</Card>
				<Card>
					<CardContent className="flex items-center gap-3 py-0">
						<CheckCircle2 className="h-5 w-5 text-success" />
						<div>
							<p className="text-2xl font-bold">{healthy}</p>
							<p className="text-xs text-muted-foreground">Healthy</p>
						</div>
					</CardContent>
				</Card>
				<Card>
					<CardContent className="flex items-center gap-3 py-0">
						<XCircle className="h-5 w-5 text-destructive" />
						<div>
							<p className="text-2xl font-bold">{unhealthy}</p>
							<p className="text-xs text-muted-foreground">Unhealthy</p>
						</div>
					</CardContent>
				</Card>
			</div>

			{/* Checks list */}
			{isLoading ? (
				<div className="space-y-3">
					{[1, 2, 3].map((i) => (
						<Skeleton key={i} className="h-20 w-full" />
					))}
				</div>
			) : !checks || checks.length === 0 ? (
				<Card>
					<CardContent className="flex flex-col items-center justify-center py-12 text-center">
						<HeartPulse className="h-10 w-10 text-muted-foreground/50" />
						<p className="mt-2 text-sm text-muted-foreground">No health checks configured.</p>
						<Button
							variant="outline"
							size="sm"
							className="mt-4"
							onClick={() => setShowCreate(true)}
						>
							<Plus className="mr-1.5 h-4 w-4" />
							Create your first health check
						</Button>
					</CardContent>
				</Card>
			) : (
				<div className="space-y-3">
					{checks.map((check) => (
						<Card key={check.id}>
							<CardContent className="flex items-center justify-between gap-4 py-0">
								<div className="flex flex-col gap-1 min-w-0">
									<div className="flex items-center gap-2">
										<span className="font-medium truncate">{check.name}</span>
										<StatusBadge status={check.status} />
										{!check.enabled && <Badge variant="secondary">Disabled</Badge>}
										{check.target_type === "host_app" && (
											<Badge variant="outline" className="text-[10px]">
												{check.app_source}/{check.app_name}
											</Badge>
										)}
									</div>
									<div className="flex items-center gap-3 text-xs text-muted-foreground">
										<span>Port {check.port}</span>
										<span>{check.url_path || "/"}</span>
										<span>Every {check.interval_seconds}s</span>
										{check.auto_restart && (
											<Badge variant="outline" className="text-[10px]">
												Auto-restart
											</Badge>
										)}
										{check.last_check_at && <span>Last: {relativeTime(check.last_check_at)}</span>}
									</div>
								</div>
								<div className="flex items-center gap-1 shrink-0">
									<Button
										variant="ghost"
										size="icon"
										className="h-8 w-8"
										title="Trigger now"
										onClick={() => triggerMut.mutate(check.id)}
										disabled={triggerMut.isPending}
									>
										<Play className="h-4 w-4" />
									</Button>
									<Button
										variant="ghost"
										size="icon"
										className="h-8 w-8"
										title="View logs"
										onClick={() => setLogsFor(check)}
									>
										<ScrollText className="h-4 w-4" />
									</Button>
									<Button
										variant="ghost"
										size="icon"
										className="h-8 w-8"
										title="Edit"
										onClick={() => setEditCheck(check)}
									>
										<Pencil className="h-4 w-4" />
									</Button>
									<Button
										variant="ghost"
										size="icon"
										className="h-8 w-8 text-destructive hover:text-destructive"
										title="Delete"
										onClick={() => setConfirmDeleteId(check.id)}
										disabled={deleteMut.isPending}
									>
										<Trash2 className="h-4 w-4" />
									</Button>
								</div>
							</CardContent>
						</Card>
					))}
				</div>
			)}

			{/* Create dialog */}
			<HealthCheckDialog open={showCreate} onOpenChange={setShowCreate} />

			{/* Edit dialog */}
			<HealthCheckDialog
				open={!!editCheck}
				onOpenChange={(open) => !open && setEditCheck(null)}
				check={editCheck ?? undefined}
			/>

			{/* Logs dialog */}
			<LogsDialog
				check={logsFor}
				open={!!logsFor}
				onOpenChange={(open) => !open && setLogsFor(null)}
			/>

			{/* Delete confirmation */}
			<AlertDialog
				open={!!confirmDeleteId}
				onOpenChange={(open) => !open && setConfirmDeleteId(null)}
			>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>Delete Health Check</AlertDialogTitle>
						<AlertDialogDescription>
							Are you sure you want to delete this health check? This action cannot be undone.
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel>Cancel</AlertDialogCancel>
						<AlertDialogAction
							onClick={() => {
								if (confirmDeleteId) {
									deleteMut.mutate(confirmDeleteId, {
										onSuccess: () => setConfirmDeleteId(null),
									});
								}
							}}
							className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
						>
							Delete
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
		</div>
	);
}

// ── Create/Edit Dialog ────────────────────────────────────

function HealthCheckDialog({
	open,
	onOpenChange,
	check,
}: {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	check?: HealthCheckRecord;
}) {
	const { data: servers } = useCollection("servers", { perPage: 200 });
	const { data: containers } = useCollection("containers", { perPage: 500 });
	const createMut = useCreateHealthCheck();
	const updateMut = useUpdateHealthCheck();

	const isEdit = !!check;

	const [name, setName] = useState(check?.name ?? "");
	const [server, setServer] = useState(check?.server ?? "");
	const [targetType, setTargetType] = useState<"container" | "host_app">(
		check?.target_type ?? "container",
	);
	const [container, setContainer] = useState(check?.container ?? "");
	const [appSource, setAppSource] = useState(check?.app_source ?? "systemd");
	const [appName, setAppName] = useState(check?.app_name ?? "");
	const [checkType, setCheckType] = useState<"http" | "tcp" | "process">(
		check?.check_type ?? "http",
	);
	const [host, setHost] = useState(check?.host ?? "127.0.0.1");
	const [scheme, setScheme] = useState(check?.scheme ?? "http");
	const [method, setMethod] = useState(check?.method ?? "GET");
	const [port, setPort] = useState(check?.port?.toString() ?? "80");
	const [urlPath, setUrlPath] = useState(check?.url_path ?? "/");
	const [expectedStatus, setExpectedStatus] = useState(check?.expected_status?.toString() ?? "200");
	const [expectedBody, setExpectedBody] = useState(check?.expected_body ?? "");
	const [tlsSkipVerify, setTlsSkipVerify] = useState(check?.tls_skip_verify ?? false);
	const [interval, setInterval] = useState(check?.interval_seconds?.toString() ?? "30");
	const [timeout, setTimeout] = useState(check?.timeout_seconds?.toString() ?? "5");
	const [failureThreshold, setFailureThreshold] = useState(
		check?.failure_threshold?.toString() ?? "3",
	);
	const [successThreshold, setSuccessThreshold] = useState(
		check?.success_threshold?.toString() ?? "1",
	);
	const [autoRestart, setAutoRestart] = useState(check?.auto_restart ?? false);
	const [maxRestarts, setMaxRestarts] = useState(check?.max_restarts?.toString() ?? "3");
	const [restartWindow, setRestartWindow] = useState(
		check?.restart_window_minutes?.toString() ?? "60",
	);
	const [enabled, setEnabled] = useState(check?.enabled ?? true);

	const { data: hostApps } = useCollection("host_apps", {
		perPage: 200,
		filter: eqFilter("server", server),
		enabled: targetType === "host_app" && !!server,
	});

	const isHostApp = targetType === "host_app";
	const isProcess = isHostApp && checkType === "process";

	const handleSubmit = (e: React.FormEvent) => {
		e.preventDefault();

		const common = {
			name,
			url_path: urlPath,
			port: Number(port) || 0,
			expected_status: Number(expectedStatus),
			interval_seconds: Number(interval),
			timeout_seconds: Number(timeout),
			failure_threshold: Number(failureThreshold),
			success_threshold: Number(successThreshold),
			auto_restart: isHostApp ? false : autoRestart,
			max_restarts: Number(maxRestarts),
			restart_window_minutes: Number(restartWindow),
			enabled,
		};

		const hostFields = {
			target_type: "host_app" as const,
			app_source: appSource,
			app_name: appName,
			check_type: checkType,
			host,
			scheme,
			method,
			expected_body: expectedBody,
			tls_skip_verify: tlsSkipVerify,
		};

		if (isEdit) {
			updateMut.mutate(
				{
					id: check.id,
					fields: isHostApp ? { ...common, ...hostFields } : common,
				},
				{ onSuccess: () => onOpenChange(false) },
			);
			return;
		}

		const input: CreateHealthCheckInput = isHostApp
			? { ...common, server, ...hostFields }
			: { ...common, server, container, target_type: "container", check_type: "http" };
		createMut.mutate(input, { onSuccess: () => onOpenChange(false) });
	};

	const isPending = createMut.isPending || updateMut.isPending;
	const filteredContainers = containers?.items?.filter((c) => !server || c.server === server);
	const filteredApps = hostApps?.items?.filter((app) => !appSource || app.source === appSource);

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="sm:max-w-lg max-h-[85vh] overflow-y-auto">
				<DialogHeader>
					<DialogTitle>{isEdit ? "Edit Health Check" : "New Health Check"}</DialogTitle>
				</DialogHeader>
				<form onSubmit={handleSubmit} className="space-y-4">
					<div className="space-y-2">
						<Label>Name</Label>
						<Input value={name} onChange={(e) => setName(e.target.value)} required />
					</div>

					{!isEdit && (
						<>
							<div className="grid grid-cols-2 gap-4">
								<div className="space-y-2">
									<Label>Target Type</Label>
									<Select
										value={targetType}
										onValueChange={(value) => {
											setTargetType(value as "container" | "host_app");
											setContainer("");
											setAppName("");
										}}
									>
										<SelectTrigger>
											<SelectValue />
										</SelectTrigger>
										<SelectContent>
											<SelectItem value="container">Container</SelectItem>
											<SelectItem value="host_app">Application</SelectItem>
										</SelectContent>
									</Select>
								</div>
								<div className="space-y-2">
									<Label>Server</Label>
									<Select value={server} onValueChange={setServer} required>
										<SelectTrigger>
											<SelectValue placeholder="Select server..." />
										</SelectTrigger>
										<SelectContent>
											{servers?.items?.map((s) => (
												<SelectItem key={s.id} value={s.id}>
													{s.name || s.hostname}
												</SelectItem>
											))}
										</SelectContent>
									</Select>
								</div>
							</div>

							{isHostApp ? (
								<div className="grid grid-cols-2 gap-4">
									<div className="space-y-2">
										<Label>App Source</Label>
										<Select
											value={appSource}
											onValueChange={(value) => {
												setAppSource(value as "systemd" | "pm2" | "process");
												setAppName("");
											}}
										>
											<SelectTrigger>
												<SelectValue />
											</SelectTrigger>
											<SelectContent>
												<SelectItem value="systemd">systemd</SelectItem>
												<SelectItem value="pm2">PM2</SelectItem>
												<SelectItem value="process">process</SelectItem>
											</SelectContent>
										</Select>
									</div>
									<div className="space-y-2">
										<Label>Application</Label>
										<Select value={appName} onValueChange={setAppName} required>
											<SelectTrigger>
												<SelectValue placeholder="Select application..." />
											</SelectTrigger>
											<SelectContent>
												{filteredApps?.map((app) => (
													<SelectItem key={app.id} value={app.name}>
														{app.name}
													</SelectItem>
												))}
											</SelectContent>
										</Select>
									</div>
								</div>
							) : (
								<div className="space-y-2">
									<Label>Container</Label>
									<Select value={container} onValueChange={setContainer} required>
										<SelectTrigger>
											<SelectValue placeholder="Select container..." />
										</SelectTrigger>
										<SelectContent>
											{filteredContainers?.map((c) => (
												<SelectItem key={c.id} value={c.id}>
													{c.name}
												</SelectItem>
											))}
										</SelectContent>
									</Select>
								</div>
							)}
						</>
					)}

					{isHostApp && (
						<div className="space-y-2">
							<Label>Check Type</Label>
							<Select
								value={checkType}
								onValueChange={(value) => setCheckType(value as "http" | "tcp" | "process")}
							>
								<SelectTrigger>
									<SelectValue />
								</SelectTrigger>
								<SelectContent>
									<SelectItem value="http">HTTP</SelectItem>
									<SelectItem value="tcp">TCP</SelectItem>
									<SelectItem value="process">Process</SelectItem>
								</SelectContent>
							</Select>
						</div>
					)}

					{!isProcess && (
						<div className="grid grid-cols-2 gap-4">
							<div className="space-y-2">
								<Label>Port</Label>
								<Input
									type="number"
									min={1}
									max={65535}
									value={port}
									onChange={(e) => setPort(e.target.value)}
									required
								/>
							</div>
							<div className="space-y-2">
								<Label>URL Path</Label>
								<Input
									value={urlPath}
									onChange={(e) => setUrlPath(e.target.value)}
									placeholder="/"
								/>
							</div>
						</div>
					)}

					{isHostApp && !isProcess && (
						<div className="grid grid-cols-3 gap-4">
							<div className="space-y-2">
								<Label>Host</Label>
								<Input
									value={host}
									onChange={(e) => setHost(e.target.value)}
									placeholder="127.0.0.1"
								/>
							</div>
							<div className="space-y-2">
								<Label>Scheme</Label>
								<Select
									value={scheme}
									onValueChange={(value) => setScheme(value as "http" | "https")}
								>
									<SelectTrigger>
										<SelectValue />
									</SelectTrigger>
									<SelectContent>
										<SelectItem value="http">http</SelectItem>
										<SelectItem value="https">https</SelectItem>
									</SelectContent>
								</Select>
							</div>
							<div className="space-y-2">
								<Label>Method</Label>
								<Select value={method} onValueChange={setMethod}>
									<SelectTrigger>
										<SelectValue />
									</SelectTrigger>
									<SelectContent>
										<SelectItem value="GET">GET</SelectItem>
										<SelectItem value="HEAD">HEAD</SelectItem>
										<SelectItem value="POST">POST</SelectItem>
									</SelectContent>
								</Select>
							</div>
						</div>
					)}

					{!isProcess && (
						<>
							<div className="grid grid-cols-3 gap-4">
								<div className="space-y-2">
									<Label>Expected Status</Label>
									<Input
										type="number"
										value={expectedStatus}
										onChange={(e) => setExpectedStatus(e.target.value)}
									/>
								</div>
								<div className="space-y-2">
									<Label>Interval (s)</Label>
									<Input
										type="number"
										min={5}
										value={interval}
										onChange={(e) => setInterval(e.target.value)}
									/>
								</div>
								<div className="space-y-2">
									<Label>Timeout (s)</Label>
									<Input
										type="number"
										min={1}
										value={timeout}
										onChange={(e) => setTimeout(e.target.value)}
									/>
								</div>
							</div>

							{isHostApp && (
								<>
									<div className="space-y-2">
										<Label>Expected Body Contains</Label>
										<Input
											value={expectedBody}
											onChange={(e) => setExpectedBody(e.target.value)}
											placeholder="Optional substring"
										/>
									</div>
									<div className="flex items-center justify-between rounded-md border p-3">
										<div>
											<p className="text-sm font-medium">Skip TLS verification</p>
											<p className="text-xs text-muted-foreground">
												Accept self-signed certificates for HTTPS checks
											</p>
										</div>
										<Switch checked={tlsSkipVerify} onCheckedChange={setTlsSkipVerify} />
									</div>
								</>
							)}
						</>
					)}

					<div className="grid grid-cols-2 gap-4">
						<div className="space-y-2">
							<Label>Failure Threshold</Label>
							<Input
								type="number"
								min={1}
								value={failureThreshold}
								onChange={(e) => setFailureThreshold(e.target.value)}
							/>
						</div>
						<div className="space-y-2">
							<Label>Success Threshold</Label>
							<Input
								type="number"
								min={1}
								value={successThreshold}
								onChange={(e) => setSuccessThreshold(e.target.value)}
							/>
						</div>
					</div>

					{!isHostApp && (
						<>
							<div className="flex items-center justify-between rounded-md border p-3">
								<div>
									<p className="text-sm font-medium">Auto-Restart</p>
									<p className="text-xs text-muted-foreground">
										Restart container on failure threshold
									</p>
								</div>
								<Switch checked={autoRestart} onCheckedChange={setAutoRestart} />
							</div>

							{autoRestart && (
								<div className="grid grid-cols-2 gap-4">
									<div className="space-y-2">
										<Label>Max Restarts</Label>
										<Input
											type="number"
											min={1}
											value={maxRestarts}
											onChange={(e) => setMaxRestarts(e.target.value)}
										/>
									</div>
									<div className="space-y-2">
										<Label>Window (min)</Label>
										<Input
											type="number"
											min={1}
											value={restartWindow}
											onChange={(e) => setRestartWindow(e.target.value)}
										/>
									</div>
								</div>
							)}
						</>
					)}

					<div className="flex items-center justify-between rounded-md border p-3">
						<div>
							<p className="text-sm font-medium">Enabled</p>
							<p className="text-xs text-muted-foreground">
								Start checking immediately after creation
							</p>
						</div>
						<Switch checked={enabled} onCheckedChange={setEnabled} />
					</div>

					<Button type="submit" className="w-full" disabled={isPending}>
						{isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
						{isEdit ? "Update" : "Create"}
					</Button>
				</form>
			</DialogContent>
		</Dialog>
	);
}

// ── Logs Dialog ───────────────────────────────────────────

function LogsDialog({
	check,
	open,
	onOpenChange,
}: {
	check: HealthCheckRecord | null;
	open: boolean;
	onOpenChange: (open: boolean) => void;
}) {
	const [page, setPage] = useState(1);
	const { data, isLoading } = useHealthCheckLogs(check?.id ?? "", page, 20);

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="sm:max-w-2xl max-h-[80vh] overflow-y-auto">
				<DialogHeader>
					<DialogTitle>Logs — {check?.name}</DialogTitle>
				</DialogHeader>

				{isLoading ? (
					<div className="space-y-2">
						{[1, 2, 3].map((i) => (
							<Skeleton key={i} className="h-10 w-full" />
						))}
					</div>
				) : !data?.items?.length ? (
					<p className="py-8 text-center text-sm text-muted-foreground">No logs yet.</p>
				) : (
					<div className="space-y-4">
						<div className="rounded-md border">
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>Time</TableHead>
										<TableHead>Status</TableHead>
										<TableHead>Code</TableHead>
										<TableHead>Latency</TableHead>
										<TableHead>Error</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{data.items.map((log) => (
										<TableRow key={log.id}>
											<TableCell className="text-xs text-muted-foreground whitespace-nowrap">
												{new Date(log.checked_at).toLocaleString()}
											</TableCell>
											<TableCell>
												{log.success ? (
													<CheckCircle2 className="h-4 w-4 text-success" />
												) : (
													<XCircle className="h-4 w-4 text-destructive" />
												)}
											</TableCell>
											<TableCell className="font-mono text-xs">{log.status_code || "—"}</TableCell>
											<TableCell className="font-mono text-xs">{log.response_time_ms}ms</TableCell>
											<TableCell className="text-xs text-muted-foreground truncate max-w-[200px]">
												{log.error || "—"}
											</TableCell>
										</TableRow>
									))}
								</TableBody>
							</Table>
						</div>

						{data.total_items > 20 && (
							<div className="flex items-center justify-between text-sm">
								<span className="text-muted-foreground">
									Page {page} ({data.total_items} total)
								</span>
								<div className="flex gap-2">
									<Button
										variant="outline"
										size="sm"
										disabled={page <= 1}
										onClick={() => setPage((p) => p - 1)}
									>
										<ChevronLeft className="h-4 w-4" />
									</Button>
									<Button
										variant="outline"
										size="sm"
										disabled={page * 20 >= data.total_items}
										onClick={() => setPage((p) => p + 1)}
									>
										Next
									</Button>
								</div>
							</div>
						)}
					</div>
				)}
			</DialogContent>
		</Dialog>
	);
}
