import { useAuthStore } from "@/stores/auth-store";

/** Returns true if the user has a specific capability (or is a superuser). */
export function useHasCapability(capability: string): boolean {
	const admin = useAuthStore((s) => s.admin);
	if (!admin) return false;
	if (admin.is_superuser) return true;

	// "x.manage" implies "x.view"
	if (admin.capabilities.includes(capability)) return true;
	if (capability.endsWith(".view")) {
		const manageCap = capability.replace(/\.view$/, ".manage");
		return admin.capabilities.includes(manageCap);
	}
	return false;
}

/** Returns all capabilities for the current user. */
export function useCapabilities(): string[] {
	const admin = useAuthStore((s) => s.admin);
	return admin?.capabilities ?? [];
}

/** Returns true if the current user is a PB superuser (full admin). */
export function useIsSuperuser(): boolean {
	const admin = useAuthStore((s) => s.admin);
	return admin?.is_superuser ?? false;
}

/** Non-hook version for use outside of React components. */
export function hasCapability(capability: string): boolean {
	const admin = useAuthStore.getState().admin;
	if (!admin) return false;
	if (admin.is_superuser) return true;
	if (admin.capabilities.includes(capability)) return true;
	if (capability.endsWith(".view")) {
		const manageCap = capability.replace(/\.view$/, ".manage");
		return admin.capabilities.includes(manageCap);
	}
	return false;
}
