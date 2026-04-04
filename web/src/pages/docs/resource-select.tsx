import { useCallback, useRef, useState } from "react";
import { Loader2 } from "lucide-react";
import { useAuthStore } from "@/stores/auth-store";
import { Input } from "@/components/ui/input";
import {
	Select,
	SelectContent,
	SelectGroup,
	SelectItem,
	SelectLabel,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { usePlaygroundAuth } from "./use-playground-auth";
import { detectResourceType } from "./playground-utils";

// ── Types ──────────────────────────────────────────────────────────────────

interface ResourceOption {
	id: string;
	label: string;
	group?: string;
	status?: "online" | "offline" | "running" | "stopped";
}

interface ResourceSelectProps {
	endpointPath: string;
	paramName: string;
	value: string;
	onChange: (value: string) => void;
	serverId?: string;
	className?: string;
}

// ── Cache ──────────────────────────────────────────────────────────────────

const cache = new Map<string, { data: ResourceOption[]; ts: number }>();
const CACHE_TTL = 60_000;

function getCached(key: string): ResourceOption[] | null {
	const entry = cache.get(key);
	if (!entry) return null;
	if (Date.now() - entry.ts > CACHE_TTL) {
		cache.delete(key);
		return null;
	}
	return entry.data;
}

const DK_PATH_MAP: Record<string, string> = {
	servers: "/api/dk/servers",
	images: "/api/dk/images",
	compose: "/api/dk/compose",
	alerts: "/api/dk/alerts",
	"backup-policies": "/api/dk/backups/policies",
	"backup-records": "/api/dk/backups/records",
	"health-checks": "/api/dk/health-checks",
	"notification-channels": "/api/dk/notifications/channels",
	users: "/api/dk/users",
	groups: "/api/dk/groups",
	"api-keys": "/api/dk/api-keys",
	tags: "/api/dk/tags",
};

const PB_COLLECTION_MAP: Record<string, string> = {
	containers: "containers",
	networks: "networks",
	volumes: "volumes",
};

async function fetchResources(
	resourceType: string,
	playgroundAuthHeaders: Record<string, string>,
	pbToken: string | null,
	serverId?: string,
): Promise<ResourceOption[]> {
	const base = window.location.origin;
	let url: URL;
	let headers: Record<string, string>;

	if (DK_PATH_MAP[resourceType]) {
		url = new URL(DK_PATH_MAP[resourceType], base);
		headers = { ...playgroundAuthHeaders };
	} else if (PB_COLLECTION_MAP[resourceType]) {
		if (!pbToken) return [];
		url = new URL(`/api/collections/${PB_COLLECTION_MAP[resourceType]}/records`, base);
		headers = { Authorization: `Bearer ${pbToken}` };
	} else {
		return [];
	}

	url.searchParams.set("perPage", "100");
	if (serverId) {
		const filterField = PB_COLLECTION_MAP[resourceType] ? "server" : "server_id";
		url.searchParams.set("filter", `${filterField}="${serverId}"`);
	}

	const res = await fetch(url.toString(), { headers });
	if (!res.ok) return [];

	const json = await res.json();
	const items = (json.items ?? json ?? []) as Record<string, unknown>[];

	return items.slice(0, 100).map((item) => {
		const id = (item.id as string) ?? "";
		let label = "";
		let group: string | undefined;
		let status: ResourceOption["status"];

		switch (resourceType) {
			case "servers":
				label = (item.hostname as string) ?? (item.name as string) ?? id;
				status = item.status === "online" ? "online" : "offline";
				break;
			case "containers":
				label = (item.name as string) ?? id;
				group = (item.server_hostname as string) ?? (item.server as string);
				status = item.status === "running" ? "running" : "stopped";
				break;
			case "images":
				label = (item.repo_tag as string) ?? (item.name as string) ?? id;
				group = (item.server_hostname as string) ?? (item.server as string);
				break;
			case "compose":
				label = (item.name as string) ?? id;
				group = (item.server_hostname as string) ?? (item.server as string);
				break;
			case "networks":
				label = (item.name as string) ?? id;
				group = (item.server_hostname as string) ?? (item.server as string);
				break;
			case "volumes":
				label = (item.name as string) ?? id;
				group = (item.server_hostname as string) ?? (item.server as string);
				break;
			case "alerts":
				label = (item.name as string) ?? id;
				break;
			case "users":
				label = (item.email as string) ?? (item.name as string) ?? id;
				break;
			case "groups":
				label = (item.name as string) ?? id;
				break;
			default:
				label = (item.name as string) ?? (item.title as string) ?? id;
				break;
		}

		return { id, label, group, status };
	});
}

// ── Status dot ─────────────────────────────────────────────────────────────

function StatusDot({ status }: { status?: ResourceOption["status"] }) {
	if (!status) return null;
	const color = status === "online" || status === "running" ? "bg-green-400" : "bg-red-400/70";
	return <span className={`inline-block h-1.5 w-1.5 rounded-full ${color}`} />;
}

// ── Component ──────────────────────────────────────────────────────────────

export function ResourceSelect({
	endpointPath,
	paramName,
	value,
	onChange,
	serverId,
	className,
}: ResourceSelectProps) {
	const { isAuthorized, authHeaders: playgroundAuthHeaders } = usePlaygroundAuth();
	const pbToken = useAuthStore((s) => s.token);
	const [options, setOptions] = useState<ResourceOption[]>([]);
	const [loading, setLoading] = useState(false);
	const fetchedRef = useRef(false);

	const resourceType = detectResourceType(endpointPath);

	const handleOpen = useCallback(
		async (open: boolean) => {
			if (!open || fetchedRef.current || !isAuthorized || !resourceType) return;
			fetchedRef.current = true;

			const cacheKey = `${resourceType}:${serverId ?? ""}`;
			const cached = getCached(cacheKey);
			if (cached) {
				setOptions(cached);
				return;
			}

			setLoading(true);
			try {
				const data = await fetchResources(resourceType, playgroundAuthHeaders, pbToken, serverId);
				cache.set(cacheKey, { data, ts: Date.now() });
				setOptions(data);
			} catch {
				// silently fail — user can type manually
			} finally {
				setLoading(false);
			}
		},
		[playgroundAuthHeaders, isAuthorized, pbToken, resourceType, serverId],
	);

	// Not authorized or unknown resource: plain input
	if (!isAuthorized || !resourceType) {
		return (
			<Input
				value={value}
				onChange={(e) => onChange(e.target.value)}
				placeholder={!isAuthorized ? `${paramName} — authorize to browse` : `Enter ${paramName}`}
				className={className}
			/>
		);
	}

	// Group items
	const grouped = new Map<string, ResourceOption[]>();
	for (const opt of options) {
		const key = opt.group ?? "";
		const arr = grouped.get(key) ?? [];
		arr.push(opt);
		grouped.set(key, arr);
	}

	return (
		<Select value={value} onValueChange={onChange} onOpenChange={handleOpen}>
			<SelectTrigger size="sm" className={className}>
				<SelectValue placeholder={`Select ${paramName}...`} />
			</SelectTrigger>
			<SelectContent position="popper" className="max-h-64">
				{loading && (
					<div className="flex items-center justify-center py-3">
						<Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
					</div>
				)}
				{!loading && options.length === 0 && (
					<div className="px-2 py-3 text-center text-xs text-muted-foreground">
						No resources found
					</div>
				)}
				{!loading &&
					[...grouped.entries()].map(([group, items]) =>
						group ? (
							<SelectGroup key={group}>
								<SelectLabel>{group}</SelectLabel>
								{items.map((item) => (
									<SelectItem key={item.id} value={item.id}>
										<StatusDot status={item.status} />
										<span className="truncate">{item.label}</span>
										<span className="ml-auto font-mono text-[10px] text-muted-foreground">
											{item.id.slice(0, 8)}
										</span>
									</SelectItem>
								))}
							</SelectGroup>
						) : (
							items.map((item) => (
								<SelectItem key={item.id} value={item.id}>
									<StatusDot status={item.status} />
									<span className="truncate">{item.label}</span>
									<span className="ml-auto font-mono text-[10px] text-muted-foreground">
										{item.id.slice(0, 8)}
									</span>
								</SelectItem>
							))
						),
					)}
			</SelectContent>
		</Select>
	);
}
