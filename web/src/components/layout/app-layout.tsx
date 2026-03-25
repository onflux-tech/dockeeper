import { useState } from "react";
import { Outlet } from "react-router";
import { SidebarContext } from "./sidebar-context";
import { DesktopSidebar, MobileSidebar } from "./sidebar";
import { Header } from "./header";
import { CommandMenu } from "./command-menu";

const SIDEBAR_KEY = "dockeeper-sidebar-collapsed";

function readCollapsed(): boolean {
	try {
		return localStorage.getItem(SIDEBAR_KEY) === "true";
	} catch {
		return false;
	}
}

export default function AppLayout() {
	const [collapsed, setCollapsedState] = useState(readCollapsed);
	const [mobileOpen, setMobileOpen] = useState(false);

	const setCollapsed = (v: boolean) => {
		setCollapsedState(v);
		try {
			localStorage.setItem(SIDEBAR_KEY, String(v));
		} catch {
			/* localStorage unavailable */
		}
	};

	return (
		<SidebarContext.Provider value={{ collapsed, setCollapsed, mobileOpen, setMobileOpen }}>
			<div className="flex h-screen overflow-hidden bg-background">
				<DesktopSidebar />
				<MobileSidebar />

				<div className="flex flex-1 flex-col overflow-hidden">
					<Header />
					<main className="flex-1 overflow-y-auto p-6">
						<Outlet />
					</main>
				</div>
			</div>

			<CommandMenu />
		</SidebarContext.Provider>
	);
}
