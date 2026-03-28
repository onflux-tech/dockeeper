import {
	Activity,
	Bell,
	Container,
	Globe,
	HardDrive,
	HeartPulse,
	Home,
	Layers,
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
			{ label: "Servers", href: "/servers", icon: Server },
		],
	},
	{
		label: "Resources",
		items: [
			{ label: "Containers", href: "/containers", icon: Container },
			{ label: "Images", href: "/images", icon: Layers },
			{ label: "Networks", href: "/networks", icon: Network },
			{ label: "Volumes", href: "/volumes", icon: HardDrive },
			{ label: "Compose", href: "/compose", icon: Rocket },
		],
	},
	{
		label: "Monitoring",
		items: [
			{ label: "Health Checks", href: "/health-checks", icon: HeartPulse },
			{ label: "Alerts", href: "/alerts", icon: Bell },
			{ label: "Backups", href: "/backups", icon: Archive },
			{ label: "Traefik", href: "/traefik", icon: Globe },
		],
	},
	{
		label: "System",
		items: [
			{ label: "Settings", href: "/settings", icon: Settings },
			{ label: "Users", href: "/users", icon: Users },
			{ label: "Events", href: "/events", icon: Activity },
			{ label: "Audit Logs", href: "/audit-logs", icon: ScrollText },
		],
	},
];

/** Flat list of all navigation items. */
export const allNavItems: NavItem[] = navigation.flatMap((g) => g.items);
