import { LogOut, Menu, PanelLeft, PanelLeftClose, Search, User } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuSeparator,
	DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { useAuthStore } from "@/stores/auth-store";
import { useSidebar } from "./sidebar-context";
import { useCommandMenu } from "./use-command-menu";

export function Header() {
	const { admin, logout } = useAuthStore();
	const { collapsed, setCollapsed, setMobileOpen } = useSidebar();
	const { open: openCommand } = useCommandMenu();

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

			{/* Desktop sidebar toggle */}
			<Button
				variant="ghost"
				size="icon-sm"
				onClick={() => setCollapsed(!collapsed)}
				className="hidden text-muted-foreground hover:text-foreground lg:flex"
			>
				{collapsed ? <PanelLeft className="h-4 w-4" /> : <PanelLeftClose className="h-4 w-4" />}
			</Button>

			{/* Centered search / cmd+k */}
			<div className="flex flex-1 justify-center pr-16">
				<Button
					variant="outline"
					size="sm"
					onClick={openCommand}
					className="hidden w-64 gap-2 text-muted-foreground sm:flex"
				>
					<Search className="h-4 w-4" />
					<span className="text-xs">Search...</span>
					<kbd className="pointer-events-none ml-auto rounded border border-border bg-muted px-1.5 py-0.5 text-[10px] font-medium leading-normal text-muted-foreground">
						⌘K
					</kbd>
				</Button>
				<Button variant="ghost" size="icon-sm" onClick={openCommand} className="sm:hidden">
					<Search className="h-4 w-4" />
				</Button>
			</div>

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
