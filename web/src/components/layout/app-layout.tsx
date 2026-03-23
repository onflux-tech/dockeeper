import { useState } from "react";
import { Outlet } from "react-router";
import { SidebarContext } from "./sidebar-context";
import { DesktopSidebar, MobileSidebar } from "./sidebar";
import { Header } from "./header";
import { CommandMenu } from "./command-menu";

export default function AppLayout() {
	const [collapsed, setCollapsed] = useState(false);
	const [mobileOpen, setMobileOpen] = useState(false);

	return (
		<SidebarContext.Provider
			value={{ collapsed, setCollapsed, mobileOpen, setMobileOpen }}
		>
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
