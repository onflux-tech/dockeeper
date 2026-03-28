import { useState } from "react";
import { Bell } from "lucide-react";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { useCreateRule, useUpdateRule } from "@/hooks/use-alerts";
import type {
	AlertRuleRecord,
	ServerRecord,
	ContainerRecord,
	NotificationChannelRecord,
} from "@/types/pocketbase";

const METRICS = [
	{ value: "cpu_percent", label: "CPU %" },
	{ value: "mem_percent", label: "Memory %" },
	{ value: "disk_used", label: "Disk Used (bytes)" },
	{ value: "net_rx", label: "Network RX (bytes/s)" },
	{ value: "net_tx", label: "Network TX (bytes/s)" },
	{ value: "load_1m", label: "Load 1m" },
	{ value: "load_5m", label: "Load 5m" },
	{ value: "mem_used", label: "Memory Used (bytes)" },
] as const;

const OPERATORS = [
	{ value: "gt", label: "> (greater than)" },
	{ value: "lt", label: "< (less than)" },
	{ value: "gte", label: ">= (greater or equal)" },
	{ value: "lte", label: "<= (less or equal)" },
	{ value: "eq", label: "= (equal)" },
] as const;

const EVENT_TYPES = [
	{ value: "container.crashed", label: "Container Crashed" },
	{ value: "container.stopped", label: "Container Stopped" },
	{ value: "container.started", label: "Container Started" },
	{ value: "container.oom_killed", label: "Container OOM Killed" },
	{ value: "server.offline", label: "Server Offline" },
	{ value: "server.online", label: "Server Online" },
] as const;

interface RuleDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	servers: ServerRecord[];
	containers: ContainerRecord[];
	channels: NotificationChannelRecord[];
	editRule?: AlertRuleRecord | null;
}

export function RuleDialog({
	open,
	onOpenChange,
	servers,
	containers,
	channels,
	editRule,
}: RuleDialogProps) {
	const isEdit = !!editRule;

	const [name, setName] = useState(editRule?.name ?? "");
	const [ruleType, setRuleType] = useState<"metric" | "event">(editRule?.rule_type ?? "metric");
	const [metric, setMetric] = useState(editRule?.metric ?? "cpu_percent");
	const [operator, setOperator] = useState<string>(editRule?.operator ?? "gt");
	const [threshold, setThreshold] = useState(editRule ? String(editRule.threshold) : "80");
	const [duration, setDuration] = useState(editRule ? String(editRule.duration) : "60");
	const [targetType, setTargetType] = useState<"server" | "container">(
		editRule?.target_type ?? "server",
	);
	const [targetId, setTargetId] = useState(editRule?.target_id ?? "");
	const [eventType, setEventType] = useState(editRule?.event_type ?? "container.crashed");
	const [scope, setScope] = useState(editRule?.scope ?? "all");
	const [activeDelay, setActiveDelay] = useState(editRule ? String(editRule.active_delay || 0) : "0");
	const [clearDelay, setClearDelay] = useState(editRule ? String(editRule.clear_delay || 0) : "0");
	const [cooldown, setCooldown] = useState(
		editRule ? String(editRule.notification_cooldown || 300) : "300",
	);
	const [suppressUpdates, setSuppressUpdates] = useState(
		editRule?.suppress_during_updates ? "yes" : "no",
	);
	const [template, setTemplate] = useState(editRule?.template ?? "");
	const [selectedChannels, setSelectedChannels] = useState<string[]>(editRule?.channels ?? []);
	const [enabled, setEnabled] = useState(editRule?.enabled !== false ? "yes" : "no");

	const createMut = useCreateRule();
	const updateMut = useUpdateRule();
	const isPending = createMut.isPending || updateMut.isPending;

	const targets = targetType === "server" ? servers : containers;

	const handleToggleChannel = (id: string) => {
		setSelectedChannels((prev) =>
			prev.includes(id) ? prev.filter((c) => c !== id) : [...prev, id],
		);
	};

	const handleSubmit = () => {
		if (!name.trim()) return;
		if (ruleType === "metric" && !targetId) return;

		const fields: Record<string, unknown> = {
			name: name.trim(),
			rule_type: ruleType,
			channels: selectedChannels,
			enabled: enabled === "yes",
			active_delay: parseInt(activeDelay, 10) || 0,
			clear_delay: parseInt(clearDelay, 10) || 0,
			notification_cooldown: parseInt(cooldown, 10) || 300,
			suppress_during_updates: suppressUpdates === "yes",
			template: template.trim(),
		};

		if (ruleType === "metric") {
			fields.metric = metric;
			fields.operator = operator;
			fields.threshold = parseFloat(threshold) || 0;
			fields.duration = parseInt(duration, 10) || 60;
			fields.target_type = targetType;
			fields.target_id = targetId;
		} else {
			fields.event_type = eventType;
			fields.scope = scope;
		}

		if (isEdit && editRule) {
			updateMut.mutate({ id: editRule.id, fields }, { onSuccess: () => onOpenChange(false) });
		} else {
			createMut.mutate(
				{
					name: fields.name as string,
					rule_type: fields.rule_type as string,
					channels: fields.channels as string[],
					enabled: fields.enabled as boolean,
					active_delay: fields.active_delay as number,
					clear_delay: fields.clear_delay as number,
					notification_cooldown: fields.notification_cooldown as number,
					suppress_during_updates: fields.suppress_during_updates as boolean,
					template: fields.template as string,
					...(ruleType === "metric"
						? {
								metric: fields.metric as string,
								operator: fields.operator as string,
								threshold: fields.threshold as number,
								duration: fields.duration as number,
								target_type: fields.target_type as string,
								target_id: fields.target_id as string,
							}
						: {
								event_type: fields.event_type as string,
								scope: fields.scope as string,
							}),
				},
				{ onSuccess: () => onOpenChange(false) },
			);
		}
	};

	const handleClose = (nextOpen: boolean) => {
		if (isPending) return;
		onOpenChange(nextOpen);
	};

	return (
		<Dialog open={open} onOpenChange={handleClose}>
			<DialogContent className="sm:max-w-lg max-h-[85vh] overflow-y-auto">
				<DialogHeader>
					<DialogTitle className="flex items-center gap-2">
						<Bell className="h-5 w-5" />
						{isEdit ? "Edit Alert Rule" : "Create Alert Rule"}
					</DialogTitle>
					<DialogDescription>
						{isEdit
							? "Update the alert rule configuration."
							: "Define a condition to trigger notifications."}
					</DialogDescription>
				</DialogHeader>

				<div className="space-y-4">
					<div className="grid grid-cols-2 gap-3">
						<div className="space-y-1.5">
							<label className="text-xs font-medium">Name</label>
							<Input
								placeholder="High CPU Alert"
								value={name}
								onChange={(e) => setName(e.target.value)}
							/>
						</div>
						<div className="space-y-1.5">
							<label className="text-xs font-medium">Rule Type</label>
							<Select
								value={ruleType}
								onValueChange={(v) => setRuleType(v as "metric" | "event")}
							>
								<SelectTrigger>
									<SelectValue />
								</SelectTrigger>
								<SelectContent>
									<SelectItem value="metric">Metric</SelectItem>
									<SelectItem value="event">Event</SelectItem>
								</SelectContent>
							</Select>
						</div>
					</div>

					{ruleType === "metric" && (
						<>
							<div className="grid grid-cols-2 gap-3">
								<div className="space-y-1.5">
									<label className="text-xs font-medium">Target Type</label>
									<Select
										value={targetType}
										onValueChange={(v) => {
											setTargetType(v as "server" | "container");
											setTargetId("");
										}}
									>
										<SelectTrigger>
											<SelectValue />
										</SelectTrigger>
										<SelectContent>
											<SelectItem value="server">Server</SelectItem>
											<SelectItem value="container">Container</SelectItem>
										</SelectContent>
									</Select>
								</div>
								<div className="space-y-1.5">
									<label className="text-xs font-medium">Target</label>
									<Select value={targetId} onValueChange={setTargetId}>
										<SelectTrigger>
											<SelectValue placeholder="Select target" />
										</SelectTrigger>
										<SelectContent>
											{targets.map((t) => (
												<SelectItem key={t.id} value={t.id}>
													{"hostname" in t
														? (t as ServerRecord).hostname
														: (t as ContainerRecord).name}
												</SelectItem>
											))}
										</SelectContent>
									</Select>
								</div>
							</div>

							<div className="grid grid-cols-3 gap-3">
								<div className="space-y-1.5">
									<label className="text-xs font-medium">Metric</label>
									<Select value={metric} onValueChange={setMetric}>
										<SelectTrigger>
											<SelectValue />
										</SelectTrigger>
										<SelectContent>
											{METRICS.map((m) => (
												<SelectItem key={m.value} value={m.value}>
													{m.label}
												</SelectItem>
											))}
										</SelectContent>
									</Select>
								</div>
								<div className="space-y-1.5">
									<label className="text-xs font-medium">Operator</label>
									<Select value={operator} onValueChange={setOperator}>
										<SelectTrigger>
											<SelectValue />
										</SelectTrigger>
										<SelectContent>
											{OPERATORS.map((op) => (
												<SelectItem key={op.value} value={op.value}>
													{op.label}
												</SelectItem>
											))}
										</SelectContent>
									</Select>
								</div>
								<div className="space-y-1.5">
									<label className="text-xs font-medium">Threshold</label>
									<Input
										type="number"
										value={threshold}
										onChange={(e) => setThreshold(e.target.value)}
									/>
								</div>
							</div>

							<div className="space-y-1.5">
								<label className="text-xs font-medium">Duration (seconds)</label>
								<Input
									type="number"
									min={10}
									className="w-32"
									value={duration}
									onChange={(e) => setDuration(e.target.value)}
								/>
								<p className="text-[10px] text-muted-foreground">
									All samples in this window must match
								</p>
							</div>
						</>
					)}

					{ruleType === "event" && (
						<div className="grid grid-cols-2 gap-3">
							<div className="space-y-1.5">
								<label className="text-xs font-medium">Event Type</label>
								<Select value={eventType} onValueChange={setEventType}>
									<SelectTrigger>
										<SelectValue />
									</SelectTrigger>
									<SelectContent>
										{EVENT_TYPES.map((et) => (
											<SelectItem key={et.value} value={et.value}>
												{et.label}
											</SelectItem>
										))}
									</SelectContent>
								</Select>
							</div>
							<div className="space-y-1.5">
								<label className="text-xs font-medium">Scope</label>
								<Input
									placeholder="all, server:<id>, container:<id>"
									value={scope}
									onChange={(e) => setScope(e.target.value)}
								/>
								<p className="text-[10px] text-muted-foreground">
									&quot;all&quot; matches every target
								</p>
							</div>
						</div>
					)}

					<div className="grid grid-cols-3 gap-3">
						<div className="space-y-1.5">
							<label className="text-xs font-medium">Active Delay (s)</label>
							<Input
								type="number"
								min={0}
								value={activeDelay}
								onChange={(e) => setActiveDelay(e.target.value)}
							/>
						</div>
						<div className="space-y-1.5">
							<label className="text-xs font-medium">Clear Delay (s)</label>
							<Input
								type="number"
								min={0}
								value={clearDelay}
								onChange={(e) => setClearDelay(e.target.value)}
							/>
						</div>
						<div className="space-y-1.5">
							<label className="text-xs font-medium">Cooldown (s)</label>
							<Input
								type="number"
								min={0}
								value={cooldown}
								onChange={(e) => setCooldown(e.target.value)}
							/>
						</div>
					</div>

					<div className="grid grid-cols-2 gap-3">
						<div className="space-y-1.5">
							<label className="text-xs font-medium">Suppress During Updates</label>
							<Select value={suppressUpdates} onValueChange={setSuppressUpdates}>
								<SelectTrigger>
									<SelectValue />
								</SelectTrigger>
								<SelectContent>
									<SelectItem value="no">No</SelectItem>
									<SelectItem value="yes">Yes</SelectItem>
								</SelectContent>
							</Select>
						</div>
						<div className="space-y-1.5">
							<label className="text-xs font-medium">Enabled</label>
							<Select value={enabled} onValueChange={setEnabled}>
								<SelectTrigger>
									<SelectValue />
								</SelectTrigger>
								<SelectContent>
									<SelectItem value="yes">Enabled</SelectItem>
									<SelectItem value="no">Disabled</SelectItem>
								</SelectContent>
							</Select>
						</div>
					</div>

					<div className="space-y-1.5">
						<label className="text-xs font-medium">Template (optional)</label>
						<Input
							placeholder="Custom notification message template"
							value={template}
							onChange={(e) => setTemplate(e.target.value)}
						/>
					</div>

					<div className="space-y-1.5">
						<label className="text-xs font-medium">Notification Channels</label>
						{channels.length === 0 ? (
							<p className="text-xs text-muted-foreground">
								No channels configured yet. Create one in the Channels tab first.
							</p>
						) : (
							<div className="flex flex-wrap gap-2">
								{channels.map((ch) => (
									<Button
										key={ch.id}
										type="button"
										variant={selectedChannels.includes(ch.id) ? "default" : "outline"}
										size="sm"
										onClick={() => handleToggleChannel(ch.id)}
									>
										{ch.name}
									</Button>
								))}
							</div>
						)}
					</div>
				</div>

				<DialogFooter>
					<Button variant="outline" onClick={() => handleClose(false)}>
						Cancel
					</Button>
					<Button
						onClick={handleSubmit}
						disabled={!name.trim() || (ruleType === "metric" && !targetId) || isPending}
					>
						{isPending
							? isEdit
								? "Saving..."
								: "Creating..."
							: isEdit
								? "Save Changes"
								: "Create Rule"}
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}
