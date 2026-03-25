import { Link, useLocation } from "react-router";
import { Container } from "lucide-react";
import { cn } from "@/lib/utils";
import { navigation } from "@/lib/navigation";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { useSidebar } from "./sidebar-context";

function NavContent({ onNavigate }: { onNavigate?: () => void }) {
	const { pathname } = useLocation();
	const { collapsed } = useSidebar();

	return (
		<ScrollArea className="flex-1 py-2">
			{navigation.map((group, gi) => (
				<div key={group.label}>
					{gi > 0 && <Separator className="my-2 bg-sidebar-border" />}
					{!collapsed && (
						<p className="mb-1 px-4 text-xs font-medium uppercase tracking-wider text-sidebar-foreground/50">
							{group.label}
						</p>
					)}
					<nav className="flex flex-col gap-0.5 px-2">
						{group.items.map((item) => {
							const isActive =
								item.href === "/" ? pathname === "/" : pathname.startsWith(item.href);
							return (
								<Link
									key={item.href}
									to={item.href}
									onClick={onNavigate}
									className={cn(
										"flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
										collapsed && "justify-center p-2",
										isActive
											? "bg-primary/10 text-primary"
											: "text-sidebar-foreground/70 hover:bg-primary/15 hover:text-primary",
									)}
								>
									<item.icon className="h-4 w-4 shrink-0" />
									{!collapsed && <span>{item.label}</span>}
								</Link>
							);
						})}
					</nav>
				</div>
			))}
		</ScrollArea>
	);
}

function SidebarHeader() {
	const { collapsed } = useSidebar();

	return (
		<div
			className={cn(
				"flex items-center border-b border-sidebar-border",
				collapsed ? "justify-center h-14" : "h-14 gap-2 px-4",
			)}
		>
			<Container className="h-5 w-5 shrink-0 text-primary" />
			{!collapsed && (
				<span className="text-lg font-semibold text-sidebar-foreground">Dockeeper</span>
			)}
		</div>
	);
}

/** Desktop sidebar. Hidden on mobile. */
export function DesktopSidebar() {
	const { collapsed } = useSidebar();

	return (
		<aside
			className={cn(
				"hidden flex-col border-r border-sidebar-border bg-sidebar lg:flex",
				"transition-all duration-300",
				collapsed ? "w-16" : "w-64",
			)}
		>
			<SidebarHeader />
			<NavContent />
		</aside>
	);
}

/** Mobile sidebar in a Sheet. */
export function MobileSidebar() {
	const { mobileOpen, setMobileOpen } = useSidebar();

	return (
		<Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
			<SheetContent side="left" className="w-64 bg-sidebar p-0">
				<SheetHeader className="flex h-14 items-center gap-2 border-b border-sidebar-border px-4">
					<Container className="h-5 w-5 text-sidebar-primary" />
					<SheetTitle className="text-lg font-semibold text-sidebar-foreground">
						Dockeeper
					</SheetTitle>
				</SheetHeader>
				<NavContent onNavigate={() => setMobileOpen(false)} />
			</SheetContent>
		</Sheet>
	);
}
