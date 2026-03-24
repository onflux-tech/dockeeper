import { useMemo, useState } from "react";
import {
	Bell,
	Plus,
	Pencil,
	Trash2,
	Radio,
	Send,
	Loader2,
	History,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import {
	Tabs,
	TabsContent,
	TabsList,
	TabsTrigger,
} from "@/components/ui/tabs";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
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
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useCollection } from "@/hooks/use-collection";
import { useRealtime } from "@/hooks/use-realtime";
import {
	useAlertRules,
	useDeleteRule,
	useToggleRule,
} from "@/hooks/use-alerts";
import {
	useNotificationChannels,
	useNotificationLogs,
	useDeleteChannel,
	useTestChannel,
} from "@/hooks/use-notifications";
import { RuleDialog } from "@/components/alerts/rule-dialog";
import { ChannelDialog } from "@/components/notifications/channel-dialog";
import type {
	AlertRuleRecord,
	NotificationChannelRecord,
	ServerRecord,
	ContainerRecord,
} from "@/types/pocketbase";

const OPERATOR_SYMBOLS: Record<string, string> = {
	gt: ">",
	lt: "<",
	gte: ">=",
	lte: "<=",
	eq: "=",
};

const STATE_VARIANT: Record<string, "default" | "destructive" | "secondary" | "outline"> = {
	ok: "default",
	firing: "destructive",
	resolved: "secondary",
};

export default function AlertsPage() {
	const [ruleSearch, setRuleSearch] = useState("");
	const [ruleDialogOpen, setRuleDialogOpen] = useState(false);
	const [editRule, setEditRule] = useState<AlertRuleRecord | null>(null);
	const [deleteRuleTarget, setDeleteRuleTarget] =
		useState<AlertRuleRecord | null>(null);

	const [channelDialogOpen, setChannelDialogOpen] = useState(false);
	const [editChannel, setEditChannel] =
		useState<NotificationChannelRecord | null>(null);
	const [deleteChannelTarget, setDeleteChannelTarget] =
		useState<NotificationChannelRecord | null>(null);

	// Data
	const { data: serversResult } = useCollection("servers", { perPage: 100 });
	const { data: containersResult } = useCollection("containers", {
		perPage: 500,
	});

	useRealtime("alert_rules");
	useRealtime("notification_channels");
	useRealtime("notification_logs");

	const { data: rules, isLoading: rulesLoading } = useAlertRules();
	const { data: channels, isLoading: channelsLoading } =
		useNotificationChannels();
	const { data: logs, isLoading: logsLoading } = useNotificationLogs();

	const toggleMut = useToggleRule();
	const deleteRuleMut = useDeleteRule();
	const deleteChannelMut = useDeleteChannel();
	const testChannelMut = useTestChannel();

	const servers = (serversResult?.items ?? []) as ServerRecord[];
	const containers = (containersResult?.items ?? []) as ContainerRecord[];

	const firingCount = useMemo(
		() => (rules ?? []).filter((r) => r.state === "firing").length,
		[rules],
	);

	const filteredRules = useMemo(() => {
		const list = rules ?? [];
		if (!ruleSearch) return list;
		const q = ruleSearch.toLowerCase();
		return list.filter(
			(r) =>
				r.name.toLowerCase().includes(q) ||
				r.metric.toLowerCase().includes(q),
		);
	}, [rules, ruleSearch]);

	const resolveTargetName = (targetType: string, targetId: string) => {
		if (targetType === "server") {
			const s = servers.find((sv) => sv.id === targetId);
			return s?.hostname ?? targetId.slice(0, 8);
		}
		const c = containers.find((ct) => ct.id === targetId);
		return c?.name ?? targetId.slice(0, 8);
	};

	const channelName = (id: string) => {
		const ch = (channels ?? []).find((c) => c.id === id);
		return ch?.name ?? id.slice(0, 8);
	};

	return (
		<div className="flex flex-col gap-6">
			{/* Header */}
			<div className="flex items-center gap-2">
				<Bell className="h-5 w-5 text-muted-foreground" />
				<h1 className="text-2xl font-semibold">Alerts</h1>
				{firingCount > 0 && (
					<Badge variant="destructive" className="gap-1">
						{firingCount} firing
					</Badge>
				)}
			</div>

			<Tabs defaultValue="rules">
				<TabsList>
					<TabsTrigger value="rules" className="gap-1.5">
						<Bell className="h-3.5 w-3.5" />
						Rules
					</TabsTrigger>
					<TabsTrigger value="channels" className="gap-1.5">
						<Radio className="h-3.5 w-3.5" />
						Channels
					</TabsTrigger>
					<TabsTrigger value="history" className="gap-1.5">
						<History className="h-3.5 w-3.5" />
						History
					</TabsTrigger>
				</TabsList>

				{/* ── Rules Tab ───────────────────────────── */}
				<TabsContent value="rules" className="space-y-4">
					<div className="flex items-center justify-between">
						<Input
							placeholder="Search rules..."
							value={ruleSearch}
							onChange={(e) => setRuleSearch(e.target.value)}
							className="max-w-xs"
						/>
						<Button
							size="sm"
							onClick={() => {
								setEditRule(null);
								setRuleDialogOpen(true);
							}}
						>
							<Plus className="mr-1.5 h-3.5 w-3.5" />
							New Rule
						</Button>
					</div>

					{rulesLoading ? (
						<div className="space-y-2">
							{[1, 2, 3].map((i) => (
								<Skeleton key={i} className="h-10 w-full" />
							))}
						</div>
					) : filteredRules.length === 0 ? (
						<p className="text-sm text-muted-foreground">
							No alert rules found. Click "New Rule" to create
							one.
						</p>
					) : (
						<div className="rounded-md border">
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>Name</TableHead>
										<TableHead>Condition</TableHead>
										<TableHead>Target</TableHead>
										<TableHead>State</TableHead>
										<TableHead>Enabled</TableHead>
										<TableHead className="w-12" />
									</TableRow>
								</TableHeader>
								<TableBody>
									{filteredRules.map((rule) => (
										<TableRow key={rule.id}>
											<TableCell className="font-medium">
												{rule.name}
											</TableCell>
											<TableCell className="font-mono text-xs">
												{rule.metric}{" "}
												{OPERATOR_SYMBOLS[rule.operator] ??
													rule.operator}{" "}
												{rule.threshold}
											</TableCell>
											<TableCell className="text-xs">
												<Badge
													variant="outline"
													className="mr-1"
												>
													{rule.target_type}
												</Badge>
												{resolveTargetName(
													rule.target_type,
													rule.target_id,
												)}
											</TableCell>
											<TableCell>
												<Badge
													variant={
														STATE_VARIANT[
															rule.state
														] ?? "outline"
													}
												>
													{rule.state}
												</Badge>
											</TableCell>
											<TableCell>
												<Switch
													checked={rule.enabled}
													onCheckedChange={() =>
														toggleMut.mutate(
															rule.id,
														)
													}
												/>
											</TableCell>
											<TableCell>
												<DropdownMenu>
													<DropdownMenuTrigger
														asChild
													>
														<Button
															variant="ghost"
															size="icon"
															className="h-7 w-7"
														>
															<span className="sr-only">
																Actions
															</span>
															<span className="text-xs">
																...
															</span>
														</Button>
													</DropdownMenuTrigger>
													<DropdownMenuContent align="end">
														<DropdownMenuItem
															onClick={() => {
																setEditRule(
																	rule,
																);
																setRuleDialogOpen(
																	true,
																);
															}}
														>
															<Pencil className="mr-2 h-3.5 w-3.5" />
															Edit
														</DropdownMenuItem>
														<DropdownMenuItem
															className="text-destructive"
															onClick={() =>
																setDeleteRuleTarget(
																	rule,
																)
															}
														>
															<Trash2 className="mr-2 h-3.5 w-3.5" />
															Delete
														</DropdownMenuItem>
													</DropdownMenuContent>
												</DropdownMenu>
											</TableCell>
										</TableRow>
									))}
								</TableBody>
							</Table>
						</div>
					)}
				</TabsContent>

				{/* ── Channels Tab ────────────────────────── */}
				<TabsContent value="channels" className="space-y-4">
					<div className="flex items-center justify-end">
						<Button
							size="sm"
							onClick={() => {
								setEditChannel(null);
								setChannelDialogOpen(true);
							}}
						>
							<Plus className="mr-1.5 h-3.5 w-3.5" />
							New Channel
						</Button>
					</div>

					{channelsLoading ? (
						<div className="space-y-2">
							{[1, 2, 3].map((i) => (
								<Skeleton key={i} className="h-10 w-full" />
							))}
						</div>
					) : (channels ?? []).length === 0 ? (
						<p className="text-sm text-muted-foreground">
							No notification channels configured. Click "New
							Channel" to add one.
						</p>
					) : (
						<div className="rounded-md border">
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>Name</TableHead>
										<TableHead>Type</TableHead>
										<TableHead>Status</TableHead>
										<TableHead className="w-40" />
									</TableRow>
								</TableHeader>
								<TableBody>
									{(channels ?? []).map((ch) => (
										<TableRow key={ch.id}>
											<TableCell className="font-medium">
												{ch.name}
											</TableCell>
											<TableCell>
												<Badge variant="outline">
													{ch.type}
												</Badge>
											</TableCell>
											<TableCell>
												<Badge
													variant={
														ch.enabled
															? "default"
															: "secondary"
													}
												>
													{ch.enabled
														? "Enabled"
														: "Disabled"}
												</Badge>
											</TableCell>
											<TableCell>
												<div className="flex items-center gap-1">
													<Button
														variant="ghost"
														size="sm"
														onClick={() =>
															testChannelMut.mutate(
																ch.id,
															)
														}
														disabled={
															testChannelMut.isPending
														}
													>
														{testChannelMut.isPending ? (
															<Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />
														) : (
															<Send className="mr-1.5 h-3.5 w-3.5" />
														)}
														Test
													</Button>
													<DropdownMenu>
														<DropdownMenuTrigger
															asChild
														>
															<Button
																variant="ghost"
																size="icon"
																className="h-7 w-7"
															>
																<span className="sr-only">
																	Actions
																</span>
																<span className="text-xs">
																	...
																</span>
															</Button>
														</DropdownMenuTrigger>
														<DropdownMenuContent align="end">
															<DropdownMenuItem
																onClick={() => {
																	setEditChannel(
																		ch,
																	);
																	setChannelDialogOpen(
																		true,
																	);
																}}
															>
																<Pencil className="mr-2 h-3.5 w-3.5" />
																Edit
															</DropdownMenuItem>
															<DropdownMenuItem
																className="text-destructive"
																onClick={() =>
																	setDeleteChannelTarget(
																		ch,
																	)
																}
															>
																<Trash2 className="mr-2 h-3.5 w-3.5" />
																Delete
															</DropdownMenuItem>
														</DropdownMenuContent>
													</DropdownMenu>
												</div>
											</TableCell>
										</TableRow>
									))}
								</TableBody>
							</Table>
						</div>
					)}
				</TabsContent>

				{/* ── History Tab ─────────────────────────── */}
				<TabsContent value="history" className="space-y-4">
					{logsLoading ? (
						<div className="space-y-2">
							{[1, 2, 3].map((i) => (
								<Skeleton key={i} className="h-10 w-full" />
							))}
						</div>
					) : (logs ?? []).length === 0 ? (
						<p className="text-sm text-muted-foreground">
							No notification history yet. Alerts will appear here
							once triggered.
						</p>
					) : (
						<div className="rounded-md border">
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>Sent At</TableHead>
										<TableHead>Channel</TableHead>
										<TableHead>Status</TableHead>
										<TableHead>Retries</TableHead>
										<TableHead className="max-w-xs">
											Message
										</TableHead>
										<TableHead className="max-w-xs">
											Error
										</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{(logs ?? []).map((log) => (
										<TableRow key={log.id}>
											<TableCell className="text-xs whitespace-nowrap">
												{log.sent_at
													? new Date(
															log.sent_at,
														).toLocaleString()
													: "—"}
											</TableCell>
											<TableCell className="text-xs">
												{channelName(log.channel)}
											</TableCell>
											<TableCell>
												<Badge
													variant={
														log.status === "sent"
															? "default"
															: "destructive"
													}
												>
													{log.status}
												</Badge>
											</TableCell>
											<TableCell className="text-xs text-center">
												{log.retries}
											</TableCell>
											<TableCell className="max-w-xs truncate text-xs text-muted-foreground">
												{log.message}
											</TableCell>
											<TableCell className="max-w-xs truncate text-xs text-destructive">
												{log.error || "—"}
											</TableCell>
										</TableRow>
									))}
								</TableBody>
							</Table>
						</div>
					)}
				</TabsContent>
			</Tabs>

			{/* Rule Dialog */}
			<RuleDialog
				key={editRule?.id ?? "create-rule"}
				open={ruleDialogOpen}
				onOpenChange={(open) => {
					setRuleDialogOpen(open);
					if (!open) setEditRule(null);
				}}
				servers={servers}
				containers={containers}
				channels={channels ?? []}
				editRule={editRule}
			/>

			{/* Channel Dialog */}
			<ChannelDialog
				key={editChannel?.id ?? "create-channel"}
				open={channelDialogOpen}
				onOpenChange={(open) => {
					setChannelDialogOpen(open);
					if (!open) setEditChannel(null);
				}}
				editChannel={editChannel}
			/>

			{/* Delete Rule Confirmation */}
			<AlertDialog
				open={!!deleteRuleTarget}
				onOpenChange={(open) => {
					if (!open) setDeleteRuleTarget(null);
				}}
			>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>Delete Alert Rule</AlertDialogTitle>
						<AlertDialogDescription>
							Are you sure you want to delete the alert rule{" "}
							<span className="font-semibold">
								{deleteRuleTarget?.name}
							</span>
							? This action cannot be undone.
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel>Cancel</AlertDialogCancel>
						<AlertDialogAction
							onClick={() => {
								if (deleteRuleTarget) {
									deleteRuleMut.mutate(deleteRuleTarget.id);
									setDeleteRuleTarget(null);
								}
							}}
						>
							Delete
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>

			{/* Delete Channel Confirmation */}
			<AlertDialog
				open={!!deleteChannelTarget}
				onOpenChange={(open) => {
					if (!open) setDeleteChannelTarget(null);
				}}
			>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>
							Delete Notification Channel
						</AlertDialogTitle>
						<AlertDialogDescription>
							Are you sure you want to delete the channel{" "}
							<span className="font-semibold">
								{deleteChannelTarget?.name}
							</span>
							? Rules using this channel will no longer send
							notifications to it.
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel>Cancel</AlertDialogCancel>
						<AlertDialogAction
							onClick={() => {
								if (deleteChannelTarget) {
									deleteChannelMut.mutate(
										deleteChannelTarget.id,
									);
									setDeleteChannelTarget(null);
								}
							}}
						>
							Delete
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
		</div>
	);
}
