import {
	Activity,
	Bell,
	Container,
	Globe,
	HardDrive,
	HeartPulse,
	Home,
	Layers,
	LayoutGrid,
	Network,
	Rocket,
	ScrollText,
	Server,
	Settings,
	Archive,
	Users,
} from "lucide-react";
import type { ComponentType } from "react";

export interface NavItem {
	label: string;
	href: string;
	icon: ComponentType<{ className?: string }>;
	/** If set, the user needs this capability to see the item. */
	capability?: string;
}

export interface NavGroup {
	label: string;
	items: NavItem[];
}

export const navigation: NavGroup[] = [
	{
		label: "Overview",
		items: [
			{ label: "Dashboard", href: "/", icon: Home },
			{ label: "Servers", href: "/servers", icon: Server, capability: "servers.view" },
			{ label: "Apps", href: "/apps", icon: LayoutGrid, capability: "host.apps.view" },
		],
	},
	{
		label: "Resources",
		items: [
			{
				label: "Containers",
				href: "/containers",
				icon: Container,
				capability: "containers.view",
			},
			{ label: "Images", href: "/images", icon: Layers, capability: "images.view" },
			{ label: "Networks", href: "/networks", icon: Network, capability: "networks.manage" },
			{ label: "Volumes", href: "/volumes", icon: HardDrive, capability: "volumes.manage" },
			{ label: "Compose", href: "/compose", icon: Rocket, capability: "compose.view" },
		],
	},
	{
		label: "Monitoring",
		items: [
			{
				label: "Health Checks",
				href: "/health-checks",
				icon: HeartPulse,
				capability: "health.view",
			},
			{ label: "Alerts", href: "/alerts", icon: Bell, capability: "alerts.view" },
			{ label: "Backups", href: "/backups", icon: Archive, capability: "backups.view" },
			{ label: "Traefik", href: "/traefik", icon: Globe, capability: "traefik.view" },
		],
	},
	{
		label: "System",
		items: [
			{ label: "Settings", href: "/settings", icon: Settings, capability: "settings.view" },
			{ label: "Users", href: "/users", icon: Users, capability: "users.view" },
			{ label: "Events", href: "/events", icon: Activity, capability: "events.view" },
			{
				label: "Audit Logs",
				href: "/audit-logs",
				icon: ScrollText,
				capability: "audit.view",
			},
		],
	},
];

/** Flat list of all navigation items. */
export const allNavItems: NavItem[] = navigation.flatMap((g) => g.items);
