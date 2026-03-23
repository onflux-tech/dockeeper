import { createContext, useContext } from "react";

export interface SidebarContextValue {
	collapsed: boolean;
	setCollapsed: (v: boolean) => void;
	mobileOpen: boolean;
	setMobileOpen: (v: boolean) => void;
}

export const SidebarContext = createContext<SidebarContextValue>({
	collapsed: false,
	setCollapsed: () => {},
	mobileOpen: false,
	setMobileOpen: () => {},
});

export function useSidebar() {
	return useContext(SidebarContext);
}
