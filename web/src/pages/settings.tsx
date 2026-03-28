import { useState } from "react";
import { useSearchParams } from "react-router";
import {
	Settings,
	Server,
	User,
	Info,
	Eye,
	EyeOff,
	Loader2,
	Copy,
	Check,
	RefreshCw,
	KeyRound,
	Plus,
	Trash2,
	Moon,
	Clock,
	Download,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import {
	Table,
	TableBody,
	TableCell,
	TableHead,
	TableHeader,
	TableRow,
} from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
	useSettings,
	getSettingValue,
	useSaveSettings,
	useChangePassword,
	useHealth,
	useEnrollmentToken,
	useRegenerateEnrollmentToken,
	useUpdateCheck,
} from "@/hooks/use-settings";
import {
	useRegistryCredentials,
	useCreateRegistryCredential,
	useDeleteRegistryCredential,
} from "@/hooks/use-updates";
import {
	useRetentionSettings,
	useUpdateRetention,
	useBlackoutWindows,
	useCreateBlackoutWindow,
	useDeleteBlackoutWindow,
} from "@/hooks/use-settings";

export default function SettingsPage() {
	const [searchParams, setSearchParams] = useSearchParams();
	const activeTab = searchParams.get("tab") || "system";

	return (
		<div className="flex flex-col gap-6">
			{/* Header */}
			<div className="flex items-center gap-2">
				<Settings className="h-5 w-5 text-muted-foreground" />
				<h1 className="text-2xl font-semibold">Settings</h1>
			</div>

			<Tabs
				value={activeTab}
				onValueChange={(v) => setSearchParams({ tab: v }, { replace: true })}
				className="space-y-4"
			>
				<TabsList>
					<TabsTrigger value="system" className="gap-1.5">
						<Server className="h-4 w-4" />
						System
					</TabsTrigger>
					<TabsTrigger value="retention" className="gap-1.5">
						<Clock className="h-4 w-4" />
						Retention
					</TabsTrigger>
					<TabsTrigger value="blackout" className="gap-1.5">
						<Moon className="h-4 w-4" />
						Blackout
					</TabsTrigger>
					<TabsTrigger value="profile" className="gap-1.5">
						<User className="h-4 w-4" />
						Profile
					</TabsTrigger>
					<TabsTrigger value="about" className="gap-1.5">
						<Info className="h-4 w-4" />
						About
					</TabsTrigger>
					<TabsTrigger value="registry" className="gap-1.5">
						<KeyRound className="h-4 w-4" />
						Registry
					</TabsTrigger>
				</TabsList>

				<TabsContent value="system" className="mt-4">
					<SystemTab />
				</TabsContent>
				<TabsContent value="retention" className="mt-4">
					<RetentionTab />
				</TabsContent>
				<TabsContent value="blackout" className="mt-4">
					<BlackoutTab />
				</TabsContent>
				<TabsContent value="profile" className="mt-4">
					<ProfileTab />
				</TabsContent>
				<TabsContent value="about" className="mt-4">
					<AboutTab />
				</TabsContent>
				<TabsContent value="registry" className="mt-4">
					<RegistryTab />
				</TabsContent>
			</Tabs>
		</div>
	);
}

/* ── System Tab ─────────────────────────────────────────── */

const S3_KEYS = [
	"s3_endpoint",
	"s3_access_key",
	"s3_secret_key",
	"s3_region",
	"s3_bucket",
] as const;

function SystemTab() {
	const { data: settings, isLoading } = useSettings();
	const save = useSaveSettings();

	const [s3, setS3] = useState<Record<string, string>>({});
	const [initialized, setInitialized] = useState(false);

	// Initialize form from settings once loaded
	if (settings && !initialized) {
		const s3Values: Record<string, string> = {};
		for (const k of S3_KEYS) {
			s3Values[k] = getSettingValue(settings, k);
		}
		setS3(s3Values);
		setInitialized(true);
	}

	if (isLoading) {
		return (
			<div className="space-y-4">
				{[1, 2, 3, 4].map((i) => (
					<Skeleton key={i} className="h-10 w-full" />
				))}
			</div>
		);
	}

	const handleSaveS3 = () => {
		save.mutate(s3);
	};

	return (
		<div className="space-y-6">
			{/* Global Enrollment Token */}
			<EnrollmentTokenSection />

			{/* S3 Backup Storage */}
			<section className="space-y-4">
				<div>
					<h3 className="text-lg font-medium">S3 Backup Storage</h3>
					<p className="text-sm text-muted-foreground">
						Configure S3-compatible storage for container backups.
					</p>
				</div>
				<div className="grid gap-4 sm:grid-cols-2">
					<div className="space-y-2">
						<Label htmlFor="s3_endpoint">Endpoint</Label>
						<Input
							id="s3_endpoint"
							autoComplete="off"
							placeholder="https://s3.amazonaws.com"
							value={s3.s3_endpoint ?? ""}
							onChange={(e) => setS3((p) => ({ ...p, s3_endpoint: e.target.value }))}
						/>
					</div>
					<div className="space-y-2">
						<Label htmlFor="s3_region">Region</Label>
						<Input
							id="s3_region"
							autoComplete="off"
							placeholder="us-east-1"
							value={s3.s3_region ?? ""}
							onChange={(e) => setS3((p) => ({ ...p, s3_region: e.target.value }))}
						/>
					</div>
					<div className="space-y-2">
						<Label htmlFor="s3_bucket">Bucket</Label>
						<Input
							id="s3_bucket"
							autoComplete="off"
							placeholder="dockeeper-backups"
							value={s3.s3_bucket ?? ""}
							onChange={(e) => setS3((p) => ({ ...p, s3_bucket: e.target.value }))}
						/>
					</div>
					<div className="space-y-2">
						<Label htmlFor="s3_access_key">Access Key</Label>
						<Input
							id="s3_access_key"
							autoComplete="off"
							placeholder="AKIAIOSFODNN7EXAMPLE"
							value={s3.s3_access_key ?? ""}
							onChange={(e) => setS3((p) => ({ ...p, s3_access_key: e.target.value }))}
						/>
					</div>
					<div className="space-y-2 sm:col-span-2">
						<Label htmlFor="s3_secret_key">Secret Key</Label>
						<Input
							id="s3_secret_key"
							autoComplete="new-password"
							type="password"
							placeholder="••••••••••••••••"
							value={s3.s3_secret_key ?? ""}
							onChange={(e) => setS3((p) => ({ ...p, s3_secret_key: e.target.value }))}
						/>
					</div>
				</div>
				<Button onClick={handleSaveS3} disabled={save.isPending}>
					{save.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
					Save S3 Settings
				</Button>
			</section>
		</div>
	);
}

/* ── Enrollment Token Section ───────────────────────────── */

function EnrollmentTokenSection() {
	const { data, isLoading } = useEnrollmentToken();
	const regenerate = useRegenerateEnrollmentToken();
	const [copied, setCopied] = useState(false);

	const handleCopy = async () => {
		if (!data?.token) return;
		await navigator.clipboard.writeText(data.token);
		setCopied(true);
		setTimeout(() => setCopied(false), 2000);
	};

	return (
		<section className="space-y-4">
			<div>
				<h3 className="text-lg font-medium">Global Enrollment Token</h3>
				<p className="text-sm text-muted-foreground">
					Used by agents to authenticate during initial enrollment via CLI. This token is also
					available in the server provisioning wizard.
				</p>
			</div>
			{isLoading ? (
				<Skeleton className="h-10 w-full" />
			) : (
				<div className="space-y-3">
					<div className="flex items-center gap-2">
						<code className="flex-1 rounded-md bg-muted px-3 py-2 font-mono text-sm break-all">
							{data?.token}
						</code>
						<Button variant="outline" size="sm" onClick={handleCopy} className="shrink-0">
							{copied ? (
								<Check className="mr-1.5 h-3.5 w-3.5" />
							) : (
								<Copy className="mr-1.5 h-3.5 w-3.5" />
							)}
							{copied ? "Copied" : "Copy"}
						</Button>
					</div>
					<Button
						variant="outline"
						size="sm"
						onClick={() => regenerate.mutate()}
						disabled={regenerate.isPending}
					>
						{regenerate.isPending ? (
							<Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />
						) : (
							<RefreshCw className="mr-1.5 h-3.5 w-3.5" />
						)}
						Regenerate Token
					</Button>
					<p className="text-xs text-muted-foreground">
						Usage:{" "}
						<code className="rounded bg-muted px-1 py-0.5">
							./dockeeper-agent --manager-url=HOST:50051 --token=TOKEN --data-dir=./data
						</code>
					</p>
				</div>
			)}
		</section>
	);
}

/* ── Profile Tab ────────────────────────────────────────── */

function ProfileTab() {
	const changePassword = useChangePassword();

	const [currentPassword, setCurrentPassword] = useState("");
	const [newPassword, setNewPassword] = useState("");
	const [confirmPassword, setConfirmPassword] = useState("");
	const [showCurrent, setShowCurrent] = useState(false);
	const [showNew, setShowNew] = useState(false);
	const [error, setError] = useState("");

	const handleSubmit = (e: React.FormEvent) => {
		e.preventDefault();
		setError("");

		if (newPassword.length < 8) {
			setError("New password must be at least 8 characters.");
			return;
		}
		if (newPassword !== confirmPassword) {
			setError("Passwords do not match.");
			return;
		}

		changePassword.mutate(
			{ current_password: currentPassword, new_password: newPassword },
			{
				onSuccess: () => {
					setCurrentPassword("");
					setNewPassword("");
					setConfirmPassword("");
				},
			},
		);
	};

	return (
		<div className="max-w-md space-y-6">
			<div>
				<h3 className="text-lg font-medium">Change Password</h3>
				<p className="text-sm text-muted-foreground">Update your administrator password.</p>
			</div>
			<form onSubmit={handleSubmit} className="space-y-4">
				<div className="space-y-2">
					<Label htmlFor="current_password">Current Password</Label>
					<div className="relative">
						<Input
							id="current_password"
							type={showCurrent ? "text" : "password"}
							value={currentPassword}
							onChange={(e) => setCurrentPassword(e.target.value)}
							required
						/>
						<button
							type="button"
							className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
							onClick={() => setShowCurrent((v) => !v)}
						>
							{showCurrent ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
						</button>
					</div>
				</div>
				<div className="space-y-2">
					<Label htmlFor="new_password">New Password</Label>
					<div className="relative">
						<Input
							id="new_password"
							type={showNew ? "text" : "password"}
							value={newPassword}
							onChange={(e) => setNewPassword(e.target.value)}
							required
							minLength={8}
						/>
						<button
							type="button"
							className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
							onClick={() => setShowNew((v) => !v)}
						>
							{showNew ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
						</button>
					</div>
				</div>
				<div className="space-y-2">
					<Label htmlFor="confirm_password">Confirm New Password</Label>
					<Input
						id="confirm_password"
						type="password"
						value={confirmPassword}
						onChange={(e) => setConfirmPassword(e.target.value)}
						required
						minLength={8}
					/>
				</div>
				{error && <p className="text-sm text-destructive">{error}</p>}
				<Button type="submit" disabled={changePassword.isPending}>
					{changePassword.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
					Change Password
				</Button>
			</form>
		</div>
	);
}

/* ── About Tab ──────────────────────────────────────────── */

function AboutTab() {
	const { data: health, isLoading } = useHealth();
	const updateCheck = useUpdateCheck();

	return (
		<div className="max-w-lg space-y-6">
			<div>
				<h3 className="text-lg font-medium">About Dockeeper</h3>
				<p className="text-sm text-muted-foreground">Server version and build information.</p>
			</div>
			{isLoading ? (
				<div className="space-y-2">
					<Skeleton className="h-5 w-48" />
					<Skeleton className="h-5 w-64" />
				</div>
			) : health ? (
				<dl className="space-y-3 text-sm">
					<div className="flex gap-2">
						<dt className="text-muted-foreground">Status:</dt>
						<dd className="font-medium">{health.status}</dd>
					</div>
					<div className="flex gap-2">
						<dt className="text-muted-foreground">Version:</dt>
						<dd className="font-mono">{health.version || "dev"}</dd>
					</div>
					<div className="flex gap-2">
						<dt className="text-muted-foreground">Commit:</dt>
						<dd className="font-mono">{health.commit ? health.commit.slice(0, 12) : "—"}</dd>
					</div>
				</dl>
			) : (
				<p className="text-sm text-muted-foreground">Unable to fetch server info.</p>
			)}

			{/* System Update Check */}
			<section className="space-y-4">
				<div>
					<h3 className="text-lg font-medium">System Updates</h3>
					<p className="text-sm text-muted-foreground">
						Check for new Dockeeper releases on GitHub.
					</p>
				</div>

				<Button
					variant="outline"
					size="sm"
					onClick={() => updateCheck.mutate()}
					disabled={updateCheck.isPending}
				>
					{updateCheck.isPending ? (
						<Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />
					) : (
						<RefreshCw className="mr-1.5 h-3.5 w-3.5" />
					)}
					Check for Updates
				</Button>

				{updateCheck.data && (
					<div className="rounded-md border p-4 space-y-3">
						{updateCheck.data.available ? (
							<>
								<div className="flex items-center gap-2">
									<Download className="h-4 w-4 text-primary" />
									<span className="font-medium text-sm">
										Update available: {updateCheck.data.latest}
									</span>
								</div>
								<div className="flex gap-4 text-xs text-muted-foreground">
									<span>Current: {updateCheck.data.current}</span>
									{updateCheck.data.published_at && (
										<span>
											Published:{" "}
											{new Date(updateCheck.data.published_at).toLocaleDateString()}
										</span>
									)}
								</div>
								{updateCheck.data.changelog && (
									<div className="rounded bg-muted p-3 text-xs whitespace-pre-wrap max-h-48 overflow-auto">
										{updateCheck.data.changelog}
									</div>
								)}
								{updateCheck.data.assets && updateCheck.data.assets.length > 0 && (
									<div className="text-xs text-muted-foreground">
										{updateCheck.data.assets.length} asset
										{updateCheck.data.assets.length > 1 ? "s" : ""} available
									</div>
								)}
							</>
						) : (
							<div className="flex items-center gap-2 text-sm text-muted-foreground">
								<Check className="h-4 w-4 text-success" />
								You're running the latest version ({updateCheck.data.current})
							</div>
						)}
					</div>
				)}

				{updateCheck.error && (
					<p className="text-sm text-destructive">{updateCheck.error.message}</p>
				)}
			</section>
		</div>
	);
}

/* ── Registry Tab ──────────────────────────────────────── */

function RegistryTab() {
	const { data: credentials, isLoading } = useRegistryCredentials();
	const createCred = useCreateRegistryCredential();
	const deleteCred = useDeleteRegistryCredential();

	const [showForm, setShowForm] = useState(false);
	const [name, setName] = useState("");
	const [host, setHost] = useState("");
	const [username, setUsername] = useState("");
	const [password, setPassword] = useState("");

	const handleCreate = (e: React.FormEvent) => {
		e.preventDefault();
		createCred.mutate(
			{ name, registry_host: host, username, password },
			{
				onSuccess: () => {
					setName("");
					setHost("");
					setUsername("");
					setPassword("");
					setShowForm(false);
				},
			},
		);
	};

	return (
		<div className="space-y-6">
			<div>
				<h3 className="text-lg font-medium">Registry Credentials</h3>
				<p className="text-sm text-muted-foreground">
					Store credentials for private Docker registries. Credentials are encrypted at rest.
				</p>
			</div>

			{isLoading ? (
				<div className="space-y-2">
					{[1, 2].map((i) => (
						<Skeleton key={i} className="h-12 w-full" />
					))}
				</div>
			) : (
				<div className="space-y-2">
					{credentials && credentials.length > 0 ? (
						<div className="rounded-md border">
							<Table>
								<TableHeader>
									<TableRow>
										<TableHead>Name</TableHead>
										<TableHead>Registry Host</TableHead>
										<TableHead>Created</TableHead>
										<TableHead className="text-right">Actions</TableHead>
									</TableRow>
								</TableHeader>
								<TableBody>
									{credentials.map((cred) => (
										<TableRow key={cred.id}>
											<TableCell className="font-medium">{cred.name}</TableCell>
											<TableCell className="font-mono text-muted-foreground">
												{cred.registry_host}
											</TableCell>
											<TableCell className="text-muted-foreground">
												{new Date(cred.created).toLocaleDateString()}
											</TableCell>
											<TableCell className="text-right">
												<Button
													variant="ghost"
													size="icon"
													className="h-8 w-8 text-destructive hover:text-destructive"
													onClick={() => deleteCred.mutate(cred.id)}
													disabled={deleteCred.isPending}
												>
													<Trash2 className="h-4 w-4" />
												</Button>
											</TableCell>
										</TableRow>
									))}
								</TableBody>
							</Table>
						</div>
					) : (
						<p className="text-sm text-muted-foreground">
							No registry credentials configured.
						</p>
					)}
				</div>
			)}

			{showForm ? (
				<form onSubmit={handleCreate} className="space-y-4 rounded-md border p-4">
					<div className="grid gap-4 sm:grid-cols-2">
						<div className="space-y-2">
							<Label htmlFor="cred_name">Label</Label>
							<Input
								id="cred_name"
								placeholder="My Docker Hub"
								value={name}
								onChange={(e) => setName(e.target.value)}
								required
							/>
						</div>
						<div className="space-y-2">
							<Label htmlFor="cred_host">Registry Host</Label>
							<Input
								id="cred_host"
								placeholder="docker.io"
								value={host}
								onChange={(e) => setHost(e.target.value)}
								required
							/>
						</div>
						<div className="space-y-2">
							<Label htmlFor="cred_user">Username</Label>
							<Input
								id="cred_user"
								placeholder="username"
								value={username}
								onChange={(e) => setUsername(e.target.value)}
								required
							/>
						</div>
						<div className="space-y-2">
							<Label htmlFor="cred_pass">Password / Token</Label>
							<Input
								id="cred_pass"
								type="password"
								placeholder="••••••••"
								value={password}
								onChange={(e) => setPassword(e.target.value)}
								required
							/>
						</div>
					</div>
					<div className="flex gap-2">
						<Button type="submit" disabled={createCred.isPending}>
							{createCred.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
							Save Credential
						</Button>
						<Button type="button" variant="outline" onClick={() => setShowForm(false)}>
							Cancel
						</Button>
					</div>
				</form>
			) : (
				<Button variant="outline" size="sm" onClick={() => setShowForm(true)}>
					<Plus className="mr-1.5 h-4 w-4" />
					Add Credential
				</Button>
			)}
		</div>
	);
}

/* ── Retention Tab ─────────────────────────────────────── */

const RETENTION_LABELS: Record<string, string> = {
	retention_raw_hours: "Raw Metrics (hours)",
	retention_hourly_days: "Hourly Aggregates (days)",
	retention_daily_days: "Daily Aggregates (days)",
	retention_events_days: "Events (days)",
	retention_audit_logs_days: "Audit Logs (days)",
	retention_notification_logs_days: "Notification Logs (days)",
};

function RetentionTab() {
	const { data, isLoading } = useRetentionSettings();
	const updateMut = useUpdateRetention();
	const [form, setForm] = useState<Record<string, string>>({});
	const [initialized, setInitialized] = useState(false);

	if (data && !initialized) {
		setForm({ ...data });
		setInitialized(true);
	}

	if (isLoading) {
		return (
			<div className="space-y-4">
				{[1, 2, 3].map((i) => (
					<Skeleton key={i} className="h-10 w-full" />
				))}
			</div>
		);
	}

	return (
		<div className="space-y-6">
			<div>
				<h3 className="text-lg font-medium">Data Retention</h3>
				<p className="text-sm text-muted-foreground">
					Configure how long each type of data is kept before automatic cleanup.
				</p>
			</div>
			<div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
				{Object.entries(RETENTION_LABELS).map(([key, label]) => (
					<div key={key} className="space-y-2">
						<Label htmlFor={key}>{label}</Label>
						<Input
							id={key}
							type="number"
							min={1}
							value={form[key] ?? ""}
							onChange={(e) => setForm((p) => ({ ...p, [key]: e.target.value }))}
						/>
					</div>
				))}
			</div>
			<Button onClick={() => updateMut.mutate(form)} disabled={updateMut.isPending}>
				{updateMut.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
				Save Retention Settings
			</Button>
		</div>
	);
}

/* ── Blackout Windows Tab ──────────────────────────────── */

function BlackoutTab() {
	const { data: windows, isLoading } = useBlackoutWindows();
	const createMut = useCreateBlackoutWindow();
	const deleteMut = useDeleteBlackoutWindow();

	const [showForm, setShowForm] = useState(false);
	const [name, setName] = useState("");
	const [server, setServer] = useState("");
	const [startTime, setStartTime] = useState("");
	const [endTime, setEndTime] = useState("");

	const handleCreate = (e: React.FormEvent) => {
		e.preventDefault();
		createMut.mutate(
			{
				name,
				server_ids: server ? [server] : [],
				start_time: startTime ? new Date(startTime).toISOString() : "",
				end_time: endTime ? new Date(endTime).toISOString() : "",
				recurring: false,
				enabled: true,
			},
			{
				onSuccess: () => {
					setName("");
					setServer("");
					setStartTime("");
					setEndTime("");
					setShowForm(false);
				},
			},
		);
	};

	return (
		<div className="space-y-6">
			<div>
				<h3 className="text-lg font-medium">Blackout Windows</h3>
				<p className="text-sm text-muted-foreground">
					Define maintenance windows during which alerts are suppressed.
				</p>
			</div>

			{isLoading ? (
				<div className="space-y-2">
					{[1, 2].map((i) => (
						<Skeleton key={i} className="h-12 w-full" />
					))}
				</div>
			) : windows && windows.length > 0 ? (
				<div className="rounded-md border">
					<Table>
						<TableHeader>
							<TableRow>
								<TableHead>Name</TableHead>
								<TableHead>Server</TableHead>
								<TableHead>Start</TableHead>
								<TableHead>End</TableHead>
								<TableHead className="text-right">Actions</TableHead>
							</TableRow>
						</TableHeader>
						<TableBody>
							{windows.map((w) => (
								<TableRow key={w.id}>
									<TableCell className="font-medium">{w.name}</TableCell>
									<TableCell className="font-mono text-xs text-muted-foreground">
										{w.server || "all"}
									</TableCell>
									<TableCell className="text-xs">
										{w.start_time ? new Date(w.start_time).toLocaleString() : "—"}
									</TableCell>
									<TableCell className="text-xs">
										{w.end_time ? new Date(w.end_time).toLocaleString() : "—"}
									</TableCell>
									<TableCell className="text-right">
										<Button
											variant="ghost"
											size="icon"
											className="h-8 w-8 text-destructive hover:text-destructive"
											onClick={() => deleteMut.mutate(w.id)}
											disabled={deleteMut.isPending}
										>
											<Trash2 className="h-4 w-4" />
										</Button>
									</TableCell>
								</TableRow>
							))}
						</TableBody>
					</Table>
				</div>
			) : (
				<p className="text-sm text-muted-foreground">No blackout windows configured.</p>
			)}

			{showForm ? (
				<form onSubmit={handleCreate} className="space-y-4 rounded-md border p-4">
					<div className="grid gap-4 sm:grid-cols-2">
						<div className="space-y-2">
							<Label htmlFor="bw_name">Name</Label>
							<Input
								id="bw_name"
								placeholder="Weekly maintenance"
								value={name}
								onChange={(e) => setName(e.target.value)}
								required
							/>
						</div>
						<div className="space-y-2">
							<Label htmlFor="bw_server">Server ID (empty = all)</Label>
							<Input
								id="bw_server"
								placeholder="Leave empty for all servers"
								value={server}
								onChange={(e) => setServer(e.target.value)}
							/>
						</div>
						<div className="space-y-2">
							<Label htmlFor="bw_start">Start Time</Label>
							<Input
								id="bw_start"
								type="datetime-local"
								value={startTime}
								onChange={(e) => setStartTime(e.target.value)}
								required
							/>
						</div>
						<div className="space-y-2">
							<Label htmlFor="bw_end">End Time</Label>
							<Input
								id="bw_end"
								type="datetime-local"
								value={endTime}
								onChange={(e) => setEndTime(e.target.value)}
								required
							/>
						</div>
					</div>
					<div className="flex gap-2">
						<Button type="submit" disabled={createMut.isPending}>
							{createMut.isPending && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
							Create Window
						</Button>
						<Button type="button" variant="outline" onClick={() => setShowForm(false)}>
							Cancel
						</Button>
					</div>
				</form>
			) : (
				<Button variant="outline" size="sm" onClick={() => setShowForm(true)}>
					<Plus className="mr-1.5 h-4 w-4" />
					Add Blackout Window
				</Button>
			)}
		</div>
	);
}
