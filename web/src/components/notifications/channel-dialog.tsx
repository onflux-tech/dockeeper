import { useState } from "react";
import { Radio } from "lucide-react";
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
import { useCreateChannel, useUpdateChannel } from "@/hooks/use-notifications";
import type { NotificationChannelRecord } from "@/types/pocketbase";

interface ChannelDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
	editChannel?: NotificationChannelRecord | null;
}

export function ChannelDialog({ open, onOpenChange, editChannel }: ChannelDialogProps) {
	const isEdit = !!editChannel;
	const editConfig = (editChannel?.config ?? {}) as Record<string, string>;

	const [name, setName] = useState(editChannel?.name ?? "");
	const [type, setType] = useState<string>(editChannel?.type ?? "webhook");
	const [enabled, setEnabled] = useState(editChannel?.enabled !== false ? "yes" : "no");

	// WhatsApp config
	const [waEndpoint, setWaEndpoint] = useState(editConfig.endpoint ?? "");
	const [waPhone, setWaPhone] = useState(editConfig.phone ?? "");
	const [waToken, setWaToken] = useState(editConfig.token ?? "");

	// Telegram config
	const [tgBotToken, setTgBotToken] = useState(editConfig.bot_token ?? "");
	const [tgChatId, setTgChatId] = useState(editConfig.chat_id ?? "");

	// Webhook config
	const [whUrl, setWhUrl] = useState(editConfig.url ?? "");

	// Discord config
	const [dcWebhookUrl, setDcWebhookUrl] = useState(editConfig.webhook_url ?? "");

	// Slack config
	const [slWebhookUrl, setSlWebhookUrl] = useState(editConfig.webhook_url ?? "");

	// SMTP config
	const [smtpHost, setSmtpHost] = useState(editConfig.host ?? "");
	const [smtpPort, setSmtpPort] = useState(editConfig.port ?? "587");
	const [smtpUser, setSmtpUser] = useState(editConfig.username ?? "");
	const [smtpPass, setSmtpPass] = useState(editConfig.password ?? "");
	const [smtpFrom, setSmtpFrom] = useState(editConfig.from ?? "");
	const [smtpTo, setSmtpTo] = useState(editConfig.to ?? "");

	// ntfy config
	const [ntfyServer, setNtfyServer] = useState(editConfig.server ?? "https://ntfy.sh");
	const [ntfyTopic, setNtfyTopic] = useState(editConfig.topic ?? "");
	const [ntfyToken, setNtfyToken] = useState(editConfig.token ?? "");

	const createMut = useCreateChannel();
	const updateMut = useUpdateChannel();
	const isPending = createMut.isPending || updateMut.isPending;

	const buildConfig = (): Record<string, unknown> => {
		switch (type) {
			case "whatsapp":
				return {
					endpoint: waEndpoint.trim(),
					phone: waPhone.trim(),
					token: waToken.trim(),
				};
			case "telegram":
				return {
					bot_token: tgBotToken.trim(),
					chat_id: tgChatId.trim(),
				};
			case "webhook":
				return { url: whUrl.trim() };
			case "discord":
				return { webhook_url: dcWebhookUrl.trim() };
			case "slack":
				return { webhook_url: slWebhookUrl.trim() };
			case "smtp":
				return {
					host: smtpHost.trim(),
					port: smtpPort.trim(),
					username: smtpUser.trim(),
					password: smtpPass.trim(),
					from: smtpFrom.trim(),
					to: smtpTo.trim(),
				};
			case "ntfy":
				return {
					server: ntfyServer.trim(),
					topic: ntfyTopic.trim(),
					token: ntfyToken.trim(),
				};
			default:
				return {};
		}
	};

	const isValid = () => {
		if (!name.trim()) return false;
		switch (type) {
			case "whatsapp":
				return !!waEndpoint.trim() && !!waPhone.trim();
			case "telegram":
				return !!tgBotToken.trim() && !!tgChatId.trim();
			case "webhook":
				return !!whUrl.trim();
			case "discord":
				return !!dcWebhookUrl.trim();
			case "slack":
				return !!slWebhookUrl.trim();
			case "smtp":
				return !!smtpHost.trim() && !!smtpFrom.trim() && !!smtpTo.trim();
			case "ntfy":
				return !!ntfyTopic.trim();
			default:
				return false;
		}
	};

	const handleSubmit = () => {
		if (!isValid()) return;

		const fields = {
			name: name.trim(),
			type,
			config: buildConfig(),
			enabled: enabled === "yes",
		};

		if (isEdit && editChannel) {
			updateMut.mutate({ id: editChannel.id, fields }, { onSuccess: () => onOpenChange(false) });
		} else {
			createMut.mutate(fields, {
				onSuccess: () => onOpenChange(false),
			});
		}
	};

	const handleClose = (nextOpen: boolean) => {
		if (isPending) return;
		onOpenChange(nextOpen);
	};

	return (
		<Dialog open={open} onOpenChange={handleClose}>
			<DialogContent className="sm:max-w-lg">
				<DialogHeader>
					<DialogTitle className="flex items-center gap-2">
						<Radio className="h-5 w-5" />
						{isEdit ? "Edit Channel" : "Create Notification Channel"}
					</DialogTitle>
					<DialogDescription>
						{isEdit
							? "Update the notification channel settings."
							: "Set up a channel to receive alert notifications."}
					</DialogDescription>
				</DialogHeader>

				<div className="space-y-4">
					<div className="grid grid-cols-2 gap-3">
						<div className="space-y-1.5">
							<label className="text-xs font-medium">Name</label>
							<Input
								placeholder="My Channel"
								value={name}
								onChange={(e) => setName(e.target.value)}
							/>
						</div>
						<div className="space-y-1.5">
							<label className="text-xs font-medium">Type</label>
							<Select value={type} onValueChange={setType} disabled={isEdit}>
								<SelectTrigger>
									<SelectValue />
								</SelectTrigger>
								<SelectContent>
									<SelectItem value="whatsapp">WhatsApp</SelectItem>
									<SelectItem value="telegram">Telegram</SelectItem>
									<SelectItem value="webhook">Webhook</SelectItem>
									<SelectItem value="discord">Discord</SelectItem>
									<SelectItem value="slack">Slack</SelectItem>
									<SelectItem value="smtp">Email (SMTP)</SelectItem>
									<SelectItem value="ntfy">ntfy</SelectItem>
								</SelectContent>
							</Select>
						</div>
					</div>

					{type === "whatsapp" && (
						<div className="space-y-3">
							<div className="space-y-1.5">
								<label className="text-xs font-medium">WuzAPI Endpoint</label>
								<Input
									placeholder="http://localhost:8080"
									value={waEndpoint}
									onChange={(e) => setWaEndpoint(e.target.value)}
								/>
							</div>
							<div className="grid grid-cols-2 gap-3">
								<div className="space-y-1.5">
									<label className="text-xs font-medium">Phone</label>
									<Input
										placeholder="5511999999999"
										value={waPhone}
										onChange={(e) => setWaPhone(e.target.value)}
										autoComplete="off"
									/>
								</div>
								<div className="space-y-1.5">
									<label className="text-xs font-medium">Token</label>
									<Input
										type="password"
										placeholder="API token"
										value={waToken}
										onChange={(e) => setWaToken(e.target.value)}
										autoComplete="off"
									/>
								</div>
							</div>
						</div>
					)}

					{type === "telegram" && (
						<div className="grid grid-cols-2 gap-3">
							<div className="space-y-1.5">
								<label className="text-xs font-medium">Bot Token</label>
								<Input
									type="password"
									placeholder="123456:ABC-DEF..."
									value={tgBotToken}
									onChange={(e) => setTgBotToken(e.target.value)}
								/>
							</div>
							<div className="space-y-1.5">
								<label className="text-xs font-medium">Chat ID</label>
								<Input
									placeholder="-1001234567890"
									value={tgChatId}
									onChange={(e) => setTgChatId(e.target.value)}
								/>
							</div>
						</div>
					)}

					{type === "webhook" && (
						<div className="space-y-1.5">
							<label className="text-xs font-medium">Webhook URL</label>
							<Input
								placeholder="https://example.com/webhook"
								value={whUrl}
								onChange={(e) => setWhUrl(e.target.value)}
							/>
						</div>
					)}

					{type === "discord" && (
						<div className="space-y-1.5">
							<label className="text-xs font-medium">Discord Webhook URL</label>
							<Input
								placeholder="https://discord.com/api/webhooks/..."
								value={dcWebhookUrl}
								onChange={(e) => setDcWebhookUrl(e.target.value)}
							/>
							<p className="text-[10px] text-muted-foreground">
								Server Settings &rarr; Integrations &rarr; Webhooks
							</p>
						</div>
					)}

					{type === "slack" && (
						<div className="space-y-1.5">
							<label className="text-xs font-medium">Slack Incoming Webhook URL</label>
							<Input
								placeholder="https://hooks.slack.com/services/..."
								value={slWebhookUrl}
								onChange={(e) => setSlWebhookUrl(e.target.value)}
							/>
						</div>
					)}

					{type === "smtp" && (
						<div className="space-y-3">
							<div className="grid grid-cols-2 gap-3">
								<div className="space-y-1.5">
									<label className="text-xs font-medium">SMTP Host</label>
									<Input
										placeholder="smtp.gmail.com"
										value={smtpHost}
										onChange={(e) => setSmtpHost(e.target.value)}
									/>
								</div>
								<div className="space-y-1.5">
									<label className="text-xs font-medium">Port</label>
									<Input
										placeholder="587"
										value={smtpPort}
										onChange={(e) => setSmtpPort(e.target.value)}
									/>
								</div>
							</div>
							<div className="grid grid-cols-2 gap-3">
								<div className="space-y-1.5">
									<label className="text-xs font-medium">Username</label>
									<Input
										placeholder="user@example.com"
										value={smtpUser}
										onChange={(e) => setSmtpUser(e.target.value)}
									/>
								</div>
								<div className="space-y-1.5">
									<label className="text-xs font-medium">Password</label>
									<Input
										type="password"
										value={smtpPass}
										onChange={(e) => setSmtpPass(e.target.value)}
									/>
								</div>
							</div>
							<div className="grid grid-cols-2 gap-3">
								<div className="space-y-1.5">
									<label className="text-xs font-medium">From</label>
									<Input
										placeholder="alerts@example.com"
										value={smtpFrom}
										onChange={(e) => setSmtpFrom(e.target.value)}
									/>
								</div>
								<div className="space-y-1.5">
									<label className="text-xs font-medium">To (comma-separated)</label>
									<Input
										placeholder="admin@example.com"
										value={smtpTo}
										onChange={(e) => setSmtpTo(e.target.value)}
									/>
								</div>
							</div>
						</div>
					)}

					{type === "ntfy" && (
						<div className="space-y-3">
							<div className="grid grid-cols-2 gap-3">
								<div className="space-y-1.5">
									<label className="text-xs font-medium">Server URL</label>
									<Input
										placeholder="https://ntfy.sh"
										value={ntfyServer}
										onChange={(e) => setNtfyServer(e.target.value)}
									/>
								</div>
								<div className="space-y-1.5">
									<label className="text-xs font-medium">Topic</label>
									<Input
										placeholder="dockeeper-alerts"
										value={ntfyTopic}
										onChange={(e) => setNtfyTopic(e.target.value)}
									/>
								</div>
							</div>
							<div className="space-y-1.5">
								<label className="text-xs font-medium">Token (optional)</label>
								<Input
									type="password"
									placeholder="Bearer token for auth"
									value={ntfyToken}
									onChange={(e) => setNtfyToken(e.target.value)}
								/>
							</div>
						</div>
					)}

					<div className="space-y-1.5">
						<label className="text-xs font-medium">Enabled</label>
						<Select value={enabled} onValueChange={setEnabled}>
							<SelectTrigger className="w-32">
								<SelectValue />
							</SelectTrigger>
							<SelectContent>
								<SelectItem value="yes">Enabled</SelectItem>
								<SelectItem value="no">Disabled</SelectItem>
							</SelectContent>
						</Select>
					</div>
				</div>

				<DialogFooter>
					<Button variant="outline" onClick={() => handleClose(false)}>
						Cancel
					</Button>
					<Button onClick={handleSubmit} disabled={!isValid() || isPending}>
						{isPending
							? isEdit
								? "Saving..."
								: "Creating..."
							: isEdit
								? "Save Changes"
								: "Create Channel"}
					</Button>
				</DialogFooter>
			</DialogContent>
		</Dialog>
	);
}
