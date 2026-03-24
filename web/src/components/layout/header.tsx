import React from "react";
import { useLocation } from "react-router";
import { LogOut, Menu, Search, User } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
	Breadcrumb,
	BreadcrumbItem,
	BreadcrumbLink,
	BreadcrumbList,
	BreadcrumbPage,
	BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuSeparator,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { useAuthStore } from "@/stores/auth-store";
import { allNavItems } from "@/lib/navigation";
import { useSidebar } from "./sidebar-context";
import { useCommandMenu } from "./command-menu";

function buildBreadcrumbs(pathname: string) {
	if (pathname === "/") return [{ label: "Dashboard", href: "/" }];

	const segments = pathname.split("/").filter(Boolean);
	return segments.map((seg, i) => {
		const href = "/" + segments.slice(0, i + 1).join("/");
		const navItem = allNavItems.find((n) => n.href === href);
		const label = navItem?.label ?? seg.charAt(0).toUpperCase() + seg.slice(1);
		return { label, href };
	});
}

export function Header() {
	const { pathname } = useLocation();
	const { admin, logout } = useAuthStore();
	const { setMobileOpen } = useSidebar();
	const { open: openCommand } = useCommandMenu();

	const crumbs = buildBreadcrumbs(pathname);
	const initials = admin?.email ? admin.email.substring(0, 2).toUpperCase() : "DK";

	return (
		<header className="flex h-14 items-center gap-3 border-b border-border bg-background px-4">
			{/* Mobile menu toggle */}
			<Button
				variant="ghost"
				size="icon-sm"
				onClick={() => setMobileOpen(true)}
				className="lg:hidden"
			>
				<Menu className="h-5 w-5" />
			</Button>

			{/* Breadcrumb */}
			<Breadcrumb className="hidden md:flex">
				<BreadcrumbList>
					{crumbs.map((crumb, i) => {
						const isLast = i === crumbs.length - 1;
						return (
							<React.Fragment key={crumb.href}>
								{i > 0 && <BreadcrumbSeparator />}
								<BreadcrumbItem>
									{isLast ? (
										<BreadcrumbPage>{crumb.label}</BreadcrumbPage>
									) : (
										<BreadcrumbLink href={crumb.href}>{crumb.label}</BreadcrumbLink>
									)}
								</BreadcrumbItem>
							</React.Fragment>
						);
					})}
				</BreadcrumbList>
			</Breadcrumb>

			{/* Spacer */}
			<div className="flex-1" />

			{/* Search / cmd+k */}
			<Button
				variant="outline"
				size="sm"
				onClick={openCommand}
				className="hidden gap-2 text-muted-foreground sm:flex"
			>
				<Search className="h-4 w-4" />
				<span className="text-xs">Search...</span>
				<kbd className="pointer-events-none ml-2 rounded border border-border bg-muted px-1.5 text-[10px] font-medium text-muted-foreground">
					⌘K
				</kbd>
			</Button>
			<Button variant="ghost" size="icon-sm" onClick={openCommand} className="sm:hidden">
				<Search className="h-4 w-4" />
			</Button>

			{/* User menu */}
			<DropdownMenu>
				<DropdownMenuTrigger asChild>
					<Button variant="ghost" size="icon-sm" className="rounded-full">
						<Avatar className="h-7 w-7">
							<AvatarFallback className="text-xs">{initials}</AvatarFallback>
						</Avatar>
					</Button>
				</DropdownMenuTrigger>
				<DropdownMenuContent align="end" className="w-48">
					<div className="px-2 py-1.5">
						<p className="text-sm font-medium">{admin?.email}</p>
						<p className="text-xs text-muted-foreground">Administrator</p>
					</div>
					<DropdownMenuSeparator />
					<DropdownMenuItem>
						<User className="mr-2 h-4 w-4" />
						Profile
					</DropdownMenuItem>
					<DropdownMenuSeparator />
					<DropdownMenuItem onClick={() => void logout()}>
						<LogOut className="mr-2 h-4 w-4" />
						Logout
					</DropdownMenuItem>
				</DropdownMenuContent>
			</DropdownMenu>
		</header>
	);
}
