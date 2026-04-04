import { useState } from "react";
import { KeyRound, Pencil, Plus, Shield, Trash2, Users2 } from "lucide-react";

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
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";

import {
	useUsers,
	useCreateUser,
	useUpdateUser,
	useDeleteUser,
	useGroups,
	useCreateGroup,
	useDeleteGroup,
	useUpdateGroup,
	useAddMember,
	useRemoveMember,
	useAPIKeys,
	useCreateAPIKey,
	useDeleteAPIKey,
	type CreatedAPIKey,
} from "@/hooks/use-users";

// ── Capabilities List ────────────────────────────────────

// Group by domain for a nicer matrix layout.
const CAP_DOMAINS = [
	{ domain: "Servers", caps: ["servers.view", "servers.manage"] },
	{ domain: "Containers", caps: ["containers.view", "containers.manage"] },
	{ domain: "Images", caps: ["images.view", "images.manage"] },
	{ domain: "Networks", caps: ["networks.view", "networks.manage"] },
	{ domain: "Volumes", caps: ["volumes.view", "volumes.manage"] },
	{ domain: "Compose", caps: ["compose.view", "compose.manage"] },
	{ domain: "Metrics", caps: ["metrics.view"] },
	{ domain: "Alerts", caps: ["alerts.view", "alerts.manage"] },
	{ domain: "Notifications", caps: ["notifications.view", "notifications.manage"] },
	{ domain: "Backups", caps: ["backups.view", "backups.manage"] },
	{ domain: "Settings", caps: ["settings.view", "settings.manage"] },
	{ domain: "Users", caps: ["users.view", "users.manage"] },
	{ domain: "Audit", caps: ["audit.view"] },
	{ domain: "Events", caps: ["events.view"] },
	{ domain: "Traefik", caps: ["traefik.view", "traefik.manage"] },
];

// ── Users Tab ────────────────────────────────────────────

function UsersTab() {
	const { data: users, isLoading } = useUsers();
	const { data: groups } = useGroups();
	const createUser = useCreateUser();
	const updateUser = useUpdateUser();
	const deleteUser = useDeleteUser();
	const addMember = useAddMember();
	const removeMember = useRemoveMember();

	const [showCreate, setShowCreate] = useState(false);
	const [editId, setEditId] = useState<string | null>(null);
	const [email, setEmail] = useState("");
	const [password, setPassword] = useState("");
	const [groupId, setGroupId] = useState("");
	const [originalGroupId, setOriginalGroupId] = useState("");
	const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);

	function resetForm() {
		setEmail("");
		setPassword("");
		setGroupId("");
		setOriginalGroupId("");
		setEditId(null);
		setShowCreate(false);
	}

	function openEdit(user: { id: string; email: string; groups: { id: string; name: string }[] }) {
		setEditId(user.id);
		setEmail(user.email);
		setPassword("");
		const currentGroup = user.groups?.[0]?.id ?? "none";
		setGroupId(currentGroup);
		setOriginalGroupId(currentGroup);
		setShowCreate(true);
	}

	function handleCreate() {
		createUser.mutate(
			{ email, password },
			{
				onSuccess: (data) => {
					if (groupId && groupId !== "none") {
						addMember.mutate({ groupId, userId: data.id });
					}
					resetForm();
				},
			},
		);
	}

	function handleUpdate() {
		if (!editId) return;
		updateUser.mutate(
			{
				id: editId,
				email,
				...(password ? { password } : {}),
			},
			{
				onSuccess: () => {
					// Handle group membership change.
					const newGroup = groupId === "none" ? "" : groupId;
					const oldGroup = originalGroupId === "none" ? "" : originalGroupId;
					if (newGroup !== oldGroup) {
						if (oldGroup) {
							removeMember.mutate({ groupId: oldGroup, userId: editId });
						}
						if (newGroup) {
							addMember.mutate({ groupId: newGroup, userId: editId });
						}
					}
					resetForm();
				},
			},
		);
	}

	function handleDelete(id: string) {
		deleteUser.mutate(id, { onSuccess: () => setConfirmDeleteId(null) });
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

			<div className="rounded-md border">
				<Table>
					<TableHeader>
						<TableRow>
							<TableHead>Email</TableHead>
							<TableHead>Groups</TableHead>
							<TableHead>Created</TableHead>
							<TableHead className="w-[100px]" />
						</TableRow>
					</TableHeader>
					<TableBody>
						{users?.map((user) => (
							<TableRow key={user.id}>
								<TableCell className="font-medium">{user.email}</TableCell>
								<TableCell>
									{user.groups?.length > 0 ? (
										<div className="flex flex-wrap gap-1">
											{user.groups.map((g) => (
												<Badge key={g.id} variant="secondary" className="text-xs">
													{g.name}
												</Badge>
											))}
										</div>
									) : (
										<span className="text-xs text-muted-foreground">No group</span>
									)}
								</TableCell>
								<TableCell className="text-muted-foreground">
									{new Date(user.created).toLocaleDateString()}
								</TableCell>
								<TableCell>
									<div className="flex items-center gap-1">
										<Button variant="ghost" size="icon" onClick={() => openEdit(user)}>
											<Pencil className="h-4 w-4" />
										</Button>
										<Button variant="ghost" size="icon" onClick={() => setConfirmDeleteId(user.id)}>
											<Trash2 className="h-4 w-4 text-destructive" />
										</Button>
									</div>
								</TableCell>
							</TableRow>
						))}
						{users?.length === 0 && (
							<TableRow>
								<TableCell colSpan={4} className="text-center text-muted-foreground">
									No users found
								</TableCell>
							</TableRow>
						)}
					</TableBody>
				</Table>
			</div>

			{/* Create / Edit User Dialog */}
			<Dialog
				open={showCreate}
				onOpenChange={(open) => {
					if (!open) resetForm();
				}}
			>
				<DialogContent>
					<DialogHeader>
						<DialogTitle>{editId ? "Edit User" : "Create User"}</DialogTitle>
						<DialogDescription>
							{editId ? "Update user email or password." : "Add a new user to the system."}
						</DialogDescription>
					</DialogHeader>
					<div className="space-y-4">
						<div className="space-y-2">
							<Label>Email</Label>
							<Input
								type="email"
								value={email}
								onChange={(e) => setEmail(e.target.value)}
								placeholder="user@example.com"
							/>
						</div>
						<div className="space-y-2">
							<Label>Password{editId ? " (leave blank to keep)" : ""}</Label>
							<Input
								type="password"
								value={password}
								onChange={(e) => setPassword(e.target.value)}
								placeholder={editId ? "Leave blank to keep" : "Minimum 8 characters"}
								required={!editId}
							/>
						</div>
						<div className="space-y-2">
							<Label>Group</Label>
							<Select value={groupId} onValueChange={setGroupId}>
								<SelectTrigger>
									<SelectValue placeholder="No group" />
								</SelectTrigger>
								<SelectContent>
									<SelectItem value="none">No group</SelectItem>
									{groups?.map((g) => (
										<SelectItem key={g.id} value={g.id}>
											{g.name}
										</SelectItem>
									))}
								</SelectContent>
							</Select>
						</div>
					</div>
					<DialogFooter>
						<Button variant="outline" onClick={resetForm}>
							Cancel
						</Button>
						<Button
							onClick={editId ? handleUpdate : handleCreate}
							disabled={
								!email ||
								(!editId && !password) ||
								(editId ? updateUser.isPending : createUser.isPending)
							}
						>
							{editId ? "Save" : "Create"}
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>

			{/* Delete Confirmation Dialog */}
			<AlertDialog
				open={!!confirmDeleteId}
				onOpenChange={(open) => !open && setConfirmDeleteId(null)}
			>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>Delete User</AlertDialogTitle>
						<AlertDialogDescription>
							Are you sure you want to delete this user? This action cannot be undone.
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel>Cancel</AlertDialogCancel>
						<AlertDialogAction
							onClick={() => confirmDeleteId && handleDelete(confirmDeleteId)}
							disabled={deleteUser.isPending}
							className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
						>
							Delete
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
		</>
	);
}

// ── Groups Tab ───────────────────────────────────────────

function GroupsTab() {
	const { data: groups, isLoading } = useGroups();
	const createGroup = useCreateGroup();
	const updateGroup = useUpdateGroup();
	const deleteGroup = useDeleteGroup();

	const [showCreate, setShowCreate] = useState(false);
	const [name, setName] = useState("");
	const [description, setDescription] = useState("");
	const [selectedCaps, setSelectedCaps] = useState<string[]>([]);
	const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);
	const [editingGroupId, setEditingGroupId] = useState<string | null>(null);
	const [viewingPreset, setViewingPreset] = useState(false);

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

	function openEdit(group: {
		id: string;
		name: string;
		description: string;
		capabilities: string[];
		is_preset: boolean;
	}) {
		setEditingGroupId(group.id);
		setName(group.name);
		setDescription(group.description);
		setSelectedCaps(group.capabilities ?? []);
		setViewingPreset(group.is_preset);
	}

	function handleUpdate() {
		if (!editingGroupId) return;
		updateGroup.mutate(
			{ id: editingGroupId, name, description, capabilities: selectedCaps },
			{
				onSuccess: () => {
					setEditingGroupId(null);
					setName("");
					setDescription("");
					setSelectedCaps([]);
				},
			},
		);
	}

	function handleDelete(id: string) {
		deleteGroup.mutate(id, { onSuccess: () => setConfirmDeleteId(null) });
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
					<Card
						key={group.id}
						className="cursor-pointer hover:ring-1 hover:ring-ring"
						onClick={() => openEdit(group)}
					>
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
											onClick={(e) => {
												e.stopPropagation();
												setConfirmDeleteId(group.id);
											}}
										>
											<Trash2 className="h-3.5 w-3.5 text-destructive" />
										</Button>
									)}
								</div>
							</div>
							{group.description && (
								<CardDescription className="text-xs">{group.description}</CardDescription>
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

			{/* Create / Edit / View Group Dialog */}
			<Dialog
				open={showCreate || !!editingGroupId}
				onOpenChange={(open) => {
					if (!open) {
						setShowCreate(false);
						setEditingGroupId(null);
						setViewingPreset(false);
						setName("");
						setDescription("");
						setSelectedCaps([]);
					}
				}}
			>
				<DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-lg">
					<DialogHeader>
						<DialogTitle>
							{viewingPreset ? name : editingGroupId ? "Edit Group" : "Create Group"}
						</DialogTitle>
						<DialogDescription>
							{viewingPreset
								? "This is a preset group. Its capabilities cannot be modified."
								: editingGroupId
									? "Update group details and capabilities."
									: "Define a new role group with capabilities."}
						</DialogDescription>
					</DialogHeader>
					<div className="space-y-4">
						{!viewingPreset && (
							<>
								<div className="space-y-2">
									<Label>Name</Label>
									<Input
										value={name}
										onChange={(e) => setName(e.target.value)}
										placeholder="Group name"
									/>
								</div>
								<div className="space-y-2">
									<Label>Description</Label>
									<Input
										value={description}
										onChange={(e) => setDescription(e.target.value)}
										placeholder="Optional description"
									/>
								</div>
							</>
						)}
						<div className="space-y-2">
							<Label>Capabilities</Label>
							<div className="mt-2 space-y-2">
								{CAP_DOMAINS.map(({ domain, caps }) => (
									<div key={domain} className="flex items-center gap-3 rounded-md border px-3 py-2">
										<span className="w-24 text-xs font-medium text-muted-foreground">{domain}</span>
										<div className="flex flex-1 gap-3">
											{caps.map((cap) => (
												<label
													key={cap}
													className={`flex items-center gap-1.5 text-xs ${viewingPreset ? "" : "cursor-pointer"}`}
												>
													<input
														type="checkbox"
														className="accent-primary"
														checked={selectedCaps.includes(cap)}
														onChange={() => !viewingPreset && toggleCap(cap)}
														disabled={viewingPreset}
													/>
													{cap.split(".")[1]}
												</label>
											))}
										</div>
									</div>
								))}
							</div>
						</div>
					</div>
					<DialogFooter>
						{viewingPreset ? (
							<Button
								onClick={() => {
									setEditingGroupId(null);
									setViewingPreset(false);
								}}
							>
								Close
							</Button>
						) : (
							<>
								<Button
									variant="outline"
									onClick={() => {
										setShowCreate(false);
										setEditingGroupId(null);
									}}
								>
									Cancel
								</Button>
								<Button
									onClick={editingGroupId ? handleUpdate : handleCreate}
									disabled={
										!name || (editingGroupId ? updateGroup.isPending : createGroup.isPending)
									}
								>
									{editingGroupId ? "Save" : "Create"}
								</Button>
							</>
						)}
					</DialogFooter>
				</DialogContent>
			</Dialog>

			{/* Delete Confirmation Dialog */}
			<AlertDialog
				open={!!confirmDeleteId}
				onOpenChange={(open) => !open && setConfirmDeleteId(null)}
			>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>Delete Group</AlertDialogTitle>
						<AlertDialogDescription>
							Are you sure you want to delete this group? Members will lose these capabilities.
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel>Cancel</AlertDialogCancel>
						<AlertDialogAction
							onClick={() => confirmDeleteId && handleDelete(confirmDeleteId)}
							disabled={deleteGroup.isPending}
							className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
						>
							Delete
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
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
	const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);

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

	function handleDelete(id: string) {
		deleteKey.mutate(id, { onSuccess: () => setConfirmDeleteId(null) });
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

			<div className="rounded-md border">
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
									{key.last_used_at ? new Date(key.last_used_at).toLocaleDateString() : "Never"}
								</TableCell>
								<TableCell className="text-muted-foreground text-sm">
									{new Date(key.created).toLocaleDateString()}
								</TableCell>
								<TableCell>
									<Button variant="ghost" size="icon" onClick={() => setConfirmDeleteId(key.id)}>
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
			</div>

			{/* Create API Key Dialog */}
			<Dialog
				open={showCreate && !createdKey}
				onOpenChange={(open) => {
					if (!open) setShowCreate(false);
				}}
			>
				<DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-lg">
					<DialogHeader>
						<DialogTitle>Create API Key</DialogTitle>
						<DialogDescription>Generate a new API key for programmatic access.</DialogDescription>
					</DialogHeader>
					<div className="space-y-4">
						<div className="space-y-2">
							<Label>Name</Label>
							<Input
								value={keyName}
								onChange={(e) => setKeyName(e.target.value)}
								placeholder="Key name"
							/>
						</div>
						<div className="space-y-2">
							<Label>Capabilities</Label>
							<div className="mt-2 space-y-2">
								{CAP_DOMAINS.map(({ domain, caps }) => (
									<div key={domain} className="flex items-center gap-3 rounded-md border px-3 py-2">
										<span className="w-24 text-xs font-medium text-muted-foreground">{domain}</span>
										<div className="flex flex-1 gap-3">
											{caps.map((cap) => (
												<label
													key={cap}
													className="flex cursor-pointer items-center gap-1.5 text-xs"
												>
													<input
														type="checkbox"
														className="accent-primary"
														checked={selectedCaps.includes(cap)}
														onChange={() => toggleCap(cap)}
													/>
													{cap.split(".")[1]}
												</label>
											))}
										</div>
									</div>
								))}
							</div>
						</div>
					</div>
					<DialogFooter>
						<Button variant="outline" onClick={() => setShowCreate(false)}>
							Cancel
						</Button>
						<Button onClick={handleCreate} disabled={!keyName || createKey.isPending}>
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
											import("sonner").then(({ toast }) => toast.success("Copied to clipboard"));
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

			{/* Delete API Key Confirmation Dialog */}
			<AlertDialog
				open={!!confirmDeleteId}
				onOpenChange={(open) => !open && setConfirmDeleteId(null)}
			>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>Revoke API Key</AlertDialogTitle>
						<AlertDialogDescription>
							Are you sure you want to revoke this API key? Applications using it will immediately
							lose access.
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel>Cancel</AlertDialogCancel>
						<AlertDialogAction
							onClick={() => confirmDeleteId && handleDelete(confirmDeleteId)}
							disabled={deleteKey.isPending}
							className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
						>
							Revoke
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
		</>
	);
}

// ── Main Page ────────────────────────────────────────────

export default function UsersPage() {
	return (
		<div className="flex flex-col gap-6">
			<div className="flex items-center gap-2">
				<Users2 className="h-5 w-5 text-muted-foreground" />
				<h1 className="text-2xl font-semibold">Users & Access</h1>
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

				<TabsContent value="users" className="mt-4">
					<UsersTab />
				</TabsContent>

				<TabsContent value="groups" className="mt-4">
					<GroupsTab />
				</TabsContent>

				<TabsContent value="api-keys" className="mt-4">
					<APIKeysTab />
				</TabsContent>
			</Tabs>
		</div>
	);
}
