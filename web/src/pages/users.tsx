import { useState } from "react";
import { KeyRound, Plus, Shield, Trash2, Users2 } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogFooter,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
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
	useUsers,
	useCreateUser,
	useDeleteUser,
	useGroups,
	useCreateGroup,
	useDeleteGroup,
	useAPIKeys,
	useCreateAPIKey,
	useDeleteAPIKey,
	type CreatedAPIKey,
} from "@/hooks/use-users";

// ── Capabilities List ────────────────────────────────────

const ALL_CAPABILITIES = [
	"servers.view",
	"servers.manage",
	"containers.view",
	"containers.manage",
	"images.view",
	"images.manage",
	"networks.view",
	"networks.manage",
	"volumes.view",
	"volumes.manage",
	"compose.view",
	"compose.manage",
	"metrics.view",
	"alerts.view",
	"alerts.manage",
	"notifications.view",
	"notifications.manage",
	"backups.view",
	"backups.manage",
	"settings.view",
	"settings.manage",
	"users.view",
	"users.manage",
	"audit.view",
	"traefik.view",
	"traefik.manage",
];

// ── Users Tab ────────────────────────────────────────────

function UsersTab() {
	const { data: users, isLoading } = useUsers();
	const createUser = useCreateUser();
	const deleteUser = useDeleteUser();

	const [showCreate, setShowCreate] = useState(false);
	const [email, setEmail] = useState("");
	const [password, setPassword] = useState("");

	function handleCreate() {
		createUser.mutate(
			{ email, password },
			{
				onSuccess: () => {
					setShowCreate(false);
					setEmail("");
					setPassword("");
				},
			},
		);
	}

	if (isLoading) {
		return (
			<div className="space-y-3">
				{Array.from({ length: 3 }).map((_, i) => (
					<Skeleton key={i} className="h-12 w-full" />
				))}
			</div>
		);
	}

	return (
		<>
			<div className="mb-4 flex items-center justify-between">
				<p className="text-sm text-muted-foreground">{users?.length ?? 0} users</p>
				<Button size="sm" onClick={() => setShowCreate(true)}>
					<Plus className="mr-1 h-4 w-4" />
					Add User
				</Button>
			</div>

			<Table>
				<TableHeader>
					<TableRow>
						<TableHead>Email</TableHead>
						<TableHead>Created</TableHead>
						<TableHead className="w-[80px]" />
					</TableRow>
				</TableHeader>
				<TableBody>
					{users?.map((user) => (
						<TableRow key={user.id}>
							<TableCell className="font-medium">{user.email}</TableCell>
							<TableCell className="text-muted-foreground">
								{new Date(user.created).toLocaleDateString()}
							</TableCell>
							<TableCell>
								<Button
									variant="ghost"
									size="icon"
									onClick={() => deleteUser.mutate(user.id)}
								>
									<Trash2 className="h-4 w-4 text-destructive" />
								</Button>
							</TableCell>
						</TableRow>
					))}
					{users?.length === 0 && (
						<TableRow>
							<TableCell colSpan={3} className="text-center text-muted-foreground">
								No users found
							</TableCell>
						</TableRow>
					)}
				</TableBody>
			</Table>

			<Dialog open={showCreate} onOpenChange={setShowCreate}>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>Create User</DialogTitle>
						<DialogDescription>Add a new user to the system.</DialogDescription>
					</DialogHeader>
					<div className="space-y-4">
						<div>
							<Label>Email</Label>
							<Input
								type="email"
								value={email}
								onChange={(e) => setEmail(e.target.value)}
								placeholder="user@example.com"
							/>
						</div>
						<div>
							<Label>Password</Label>
							<Input
								type="password"
								value={password}
								onChange={(e) => setPassword(e.target.value)}
								placeholder="Minimum 8 characters"
							/>
						</div>
					</div>
					<DialogFooter>
						<Button variant="outline" onClick={() => setShowCreate(false)}>
							Cancel
						</Button>
						<Button
							onClick={handleCreate}
							disabled={!email || !password || createUser.isPending}
						>
							Create
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</>
	);
}

// ── Groups Tab ───────────────────────────────────────────

function GroupsTab() {
	const { data: groups, isLoading } = useGroups();
	const createGroup = useCreateGroup();
	const deleteGroup = useDeleteGroup();

	const [showCreate, setShowCreate] = useState(false);
	const [name, setName] = useState("");
	const [description, setDescription] = useState("");
	const [selectedCaps, setSelectedCaps] = useState<string[]>([]);

	function toggleCap(cap: string) {
		setSelectedCaps((prev) =>
			prev.includes(cap) ? prev.filter((c) => c !== cap) : [...prev, cap],
		);
	}

	function handleCreate() {
		createGroup.mutate(
			{ name, description, capabilities: selectedCaps },
			{
				onSuccess: () => {
					setShowCreate(false);
					setName("");
					setDescription("");
					setSelectedCaps([]);
				},
			},
		);
	}

	if (isLoading) {
		return (
			<div className="space-y-3">
				{Array.from({ length: 3 }).map((_, i) => (
					<Skeleton key={i} className="h-24 w-full" />
				))}
			</div>
		);
	}

	return (
		<>
			<div className="mb-4 flex items-center justify-between">
				<p className="text-sm text-muted-foreground">{groups?.length ?? 0} groups</p>
				<Button size="sm" onClick={() => setShowCreate(true)}>
					<Plus className="mr-1 h-4 w-4" />
					Add Group
				</Button>
			</div>

			<div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
				{groups?.map((group) => (
					<Card key={group.id}>
						<CardHeader className="pb-3">
							<div className="flex items-center justify-between">
								<CardTitle className="text-base">{group.name}</CardTitle>
								<div className="flex items-center gap-1">
									{group.is_preset && (
										<Badge variant="secondary" className="text-xs">
											Preset
										</Badge>
									)}
									{!group.is_preset && (
										<Button
											variant="ghost"
											size="icon"
											className="h-7 w-7"
											onClick={() => deleteGroup.mutate(group.id)}
										>
											<Trash2 className="h-3.5 w-3.5 text-destructive" />
										</Button>
									)}
								</div>
							</div>
							{group.description && (
								<CardDescription className="text-xs">
									{group.description}
								</CardDescription>
							)}
						</CardHeader>
						<CardContent>
							<div className="flex flex-wrap gap-1">
								{(group.capabilities ?? []).slice(0, 6).map((cap) => (
									<Badge key={cap} variant="outline" className="text-xs">
										{cap}
									</Badge>
								))}
								{(group.capabilities ?? []).length > 6 && (
									<Badge variant="outline" className="text-xs">
										+{group.capabilities.length - 6} more
									</Badge>
								)}
							</div>
						</CardContent>
					</Card>
				))}
			</div>

			<Dialog open={showCreate} onOpenChange={setShowCreate}>
				<DialogContent className="max-h-[80vh] overflow-y-auto sm:max-w-lg">
					<DialogHeader>
						<DialogTitle>Create Group</DialogTitle>
						<DialogDescription>
							Define a new role group with capabilities.
						</DialogDescription>
					</DialogHeader>
					<div className="space-y-4">
						<div>
							<Label>Name</Label>
							<Input
								value={name}
								onChange={(e) => setName(e.target.value)}
								placeholder="Group name"
							/>
						</div>
						<div>
							<Label>Description</Label>
							<Input
								value={description}
								onChange={(e) => setDescription(e.target.value)}
								placeholder="Optional description"
							/>
						</div>
						<div>
							<Label>Capabilities</Label>
							<div className="mt-2 grid grid-cols-2 gap-1.5">
								{ALL_CAPABILITIES.map((cap) => (
									<label
										key={cap}
										className="flex cursor-pointer items-center gap-2 rounded-md border px-2 py-1.5 text-xs hover:bg-muted"
									>
										<input
											type="checkbox"
											className="accent-primary"
											checked={selectedCaps.includes(cap)}
											onChange={() => toggleCap(cap)}
										/>
										{cap}
									</label>
								))}
							</div>
						</div>
					</div>
					<DialogFooter>
						<Button variant="outline" onClick={() => setShowCreate(false)}>
							Cancel
						</Button>
						<Button
							onClick={handleCreate}
							disabled={!name || createGroup.isPending}
						>
							Create
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</>
	);
}

// ── API Keys Tab ─────────────────────────────────────────

function APIKeysTab() {
	const { data: keys, isLoading } = useAPIKeys();
	const createKey = useCreateAPIKey();
	const deleteKey = useDeleteAPIKey();

	const [showCreate, setShowCreate] = useState(false);
	const [keyName, setKeyName] = useState("");
	const [selectedCaps, setSelectedCaps] = useState<string[]>([]);
	const [createdKey, setCreatedKey] = useState<CreatedAPIKey | null>(null);

	function toggleCap(cap: string) {
		setSelectedCaps((prev) =>
			prev.includes(cap) ? prev.filter((c) => c !== cap) : [...prev, cap],
		);
	}

	function handleCreate() {
		createKey.mutate(
			{ name: keyName, capabilities: selectedCaps },
			{
				onSuccess: (data) => {
					setCreatedKey(data);
					setKeyName("");
					setSelectedCaps([]);
				},
			},
		);
	}

	if (isLoading) {
		return (
			<div className="space-y-3">
				{Array.from({ length: 3 }).map((_, i) => (
					<Skeleton key={i} className="h-12 w-full" />
				))}
			</div>
		);
	}

	return (
		<>
			<div className="mb-4 flex items-center justify-between">
				<p className="text-sm text-muted-foreground">{keys?.length ?? 0} API keys</p>
				<Button size="sm" onClick={() => setShowCreate(true)}>
					<Plus className="mr-1 h-4 w-4" />
					Create API Key
				</Button>
			</div>

			<Table>
				<TableHeader>
					<TableRow>
						<TableHead>Name</TableHead>
						<TableHead>Prefix</TableHead>
						<TableHead>Status</TableHead>
						<TableHead>Last Used</TableHead>
						<TableHead>Created</TableHead>
						<TableHead className="w-[80px]" />
					</TableRow>
				</TableHeader>
				<TableBody>
					{keys?.map((key) => (
						<TableRow key={key.id}>
							<TableCell className="font-medium">{key.name}</TableCell>
							<TableCell>
								<code className="text-xs">{key.key_prefix}...</code>
							</TableCell>
							<TableCell>
								<Badge variant={key.enabled ? "default" : "secondary"}>
									{key.enabled ? "Active" : "Disabled"}
								</Badge>
							</TableCell>
							<TableCell className="text-muted-foreground text-sm">
								{key.last_used_at
									? new Date(key.last_used_at).toLocaleDateString()
									: "Never"}
							</TableCell>
							<TableCell className="text-muted-foreground text-sm">
								{new Date(key.created).toLocaleDateString()}
							</TableCell>
							<TableCell>
								<Button
									variant="ghost"
									size="icon"
									onClick={() => deleteKey.mutate(key.id)}
								>
									<Trash2 className="h-4 w-4 text-destructive" />
								</Button>
							</TableCell>
						</TableRow>
					))}
					{keys?.length === 0 && (
						<TableRow>
							<TableCell colSpan={6} className="text-center text-muted-foreground">
								No API keys found
							</TableCell>
						</TableRow>
					)}
				</TableBody>
			</Table>

			{/* Create API Key Dialog */}
			<Dialog
				open={showCreate && !createdKey}
				onOpenChange={(open) => {
					if (!open) setShowCreate(false);
				}}
			>
				<DialogContent className="max-h-[80vh] overflow-y-auto sm:max-w-lg">
					<DialogHeader>
						<DialogTitle>Create API Key</DialogTitle>
						<DialogDescription>
							Generate a new API key for programmatic access.
						</DialogDescription>
					</DialogHeader>
					<div className="space-y-4">
						<div>
							<Label>Name</Label>
							<Input
								value={keyName}
								onChange={(e) => setKeyName(e.target.value)}
								placeholder="Key name"
							/>
						</div>
						<div>
							<Label>Capabilities</Label>
							<div className="mt-2 grid grid-cols-2 gap-1.5">
								{ALL_CAPABILITIES.map((cap) => (
									<label
										key={cap}
										className="flex cursor-pointer items-center gap-2 rounded-md border px-2 py-1.5 text-xs hover:bg-muted"
									>
										<input
											type="checkbox"
											className="accent-primary"
											checked={selectedCaps.includes(cap)}
											onChange={() => toggleCap(cap)}
										/>
										{cap}
									</label>
								))}
							</div>
						</div>
					</div>
					<DialogFooter>
						<Button variant="outline" onClick={() => setShowCreate(false)}>
							Cancel
						</Button>
						<Button
							onClick={handleCreate}
							disabled={!keyName || createKey.isPending}
						>
							Generate
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>

			{/* Show Created Key Dialog */}
			<Dialog
				open={!!createdKey}
				onOpenChange={(open) => {
					if (!open) {
						setCreatedKey(null);
						setShowCreate(false);
					}
				}}
			>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>API Key Created</DialogTitle>
						<DialogDescription>
							Copy your API key now. It will not be shown again.
						</DialogDescription>
					</DialogHeader>
					{createdKey && (
						<div className="space-y-3">
							<div>
								<Label className="text-xs text-muted-foreground">Name</Label>
								<p className="font-medium">{createdKey.name}</p>
							</div>
							<div>
								<Label className="text-xs text-muted-foreground">API Key</Label>
								<div className="flex items-center gap-2">
									<code className="flex-1 rounded-md border bg-muted px-3 py-2 text-sm break-all">
										{createdKey.key}
									</code>
									<Button
										size="sm"
										variant="outline"
										onClick={() => {
											void navigator.clipboard.writeText(createdKey.key);
											import("sonner").then(({ toast }) =>
												toast.success("Copied to clipboard"),
											);
										}}
									>
										Copy
									</Button>
								</div>
							</div>
						</div>
					)}
					<DialogFooter>
						<Button
							onClick={() => {
								setCreatedKey(null);
								setShowCreate(false);
							}}
						>
							Done
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</>
	);
}

// ── Main Page ────────────────────────────────────────────

export default function UsersPage() {
	return (
		<div className="space-y-6">
			<div>
				<h1 className="text-2xl font-bold tracking-tight">Users & Access</h1>
				<p className="text-muted-foreground">
					Manage users, role groups, and API keys.
				</p>
			</div>

			<Tabs defaultValue="users">
				<TabsList>
					<TabsTrigger value="users" className="gap-1.5">
						<Users2 className="h-4 w-4" />
						Users
					</TabsTrigger>
					<TabsTrigger value="groups" className="gap-1.5">
						<Shield className="h-4 w-4" />
						Groups
					</TabsTrigger>
					<TabsTrigger value="api-keys" className="gap-1.5">
						<KeyRound className="h-4 w-4" />
						API Keys
					</TabsTrigger>
				</TabsList>

				<TabsContent value="users" className="mt-6">
					<UsersTab />
				</TabsContent>

				<TabsContent value="groups" className="mt-6">
					<GroupsTab />
				</TabsContent>

				<TabsContent value="api-keys" className="mt-6">
					<APIKeysTab />
				</TabsContent>
			</Tabs>
		</div>
	);
}
