import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { AdminData } from "@/types/auth";
import { authApi, setAccessToken } from "@/lib/api";
import { syncPbAuth } from "@/lib/pb";

const REFRESH_SKEW_MS = 60_000;
const MIN_REFRESH_DELAY_MS = 10_000;

let refreshTimer: ReturnType<typeof setTimeout> | null = null;

function decodeJwtExp(token: string): number | null {
	const parts = token.split(".");
	if (parts.length < 2) return null;

	try {
		const base64 = parts[1].replace(/-/g, "+").replace(/_/g, "/");
		const padded = base64.padEnd(base64.length + ((4 - (base64.length % 4)) % 4), "=");
		const payload = JSON.parse(atob(padded)) as { exp?: number };
		return typeof payload.exp === "number" ? payload.exp * 1000 : null;
	} catch {
		return null;
	}
}

function clearRefreshTimer() {
	if (refreshTimer) {
		clearTimeout(refreshTimer);
		refreshTimer = null;
	}
}

function scheduleRefresh(token: string | null, refreshToken: (() => Promise<boolean>) | null) {
	clearRefreshTimer();
	if (!token || !refreshToken) return;

	const expMs = decodeJwtExp(token);
	if (!expMs) return;

	const delay = Math.max(expMs - Date.now() - REFRESH_SKEW_MS, MIN_REFRESH_DELAY_MS);
	refreshTimer = setTimeout(() => {
		void refreshToken();
	}, delay);
}

interface AuthState {
	token: string | null;
	refresh: string | null;
	admin: AdminData | null;
	isAuthenticated: boolean;
	isLoading: boolean;

	login: (email: string, password: string) => Promise<void>;
	logout: () => Promise<void>;
	refreshToken: () => Promise<boolean>;
	checkAuth: () => Promise<void>;
	clear: () => void;
}

export const useAuthStore = create<AuthState>()(
	persist(
		(set, get) => ({
			token: null,
			refresh: null,
			admin: null,
			isAuthenticated: false,
			isLoading: true,

			login: async (email, password) => {
				// Check if this is the first-time setup.
				const { needs_setup } = await authApi.status();
				const res = needs_setup
					? await authApi.setup(email, password)
					: await authApi.login(email, password);
				setAccessToken(res.token);
				syncPbAuth(res.token);
				set({
					token: res.token,
					refresh: res.refresh,
					admin: res.admin,
					isAuthenticated: true,
				});
				scheduleRefresh(res.token, get().refreshToken);
			},

			logout: async () => {
				try {
					await authApi.logout();
				} catch {
					// Ignore errors on logout — we clear state regardless.
				}
				setAccessToken(null);
				syncPbAuth(null);
				clearRefreshTimer();
				set({
					token: null,
					refresh: null,
					admin: null,
					isAuthenticated: false,
				});
			},

			refreshToken: async () => {
				const { refresh } = get();
				if (!refresh) return false;

				try {
					const res = await authApi.refresh(refresh);
					setAccessToken(res.token);
					syncPbAuth(res.token);
					set({
						token: res.token,
						refresh: res.refresh,
						admin: res.admin,
						isAuthenticated: true,
					});
					scheduleRefresh(res.token, get().refreshToken);
					return true;
				} catch {
					get().clear();
					return false;
				}
			},

			checkAuth: async () => {
				const { token, refresh } = get();

				if (!token) {
					clearRefreshTimer();
					set({ isLoading: false, isAuthenticated: false });
					return;
				}

				setAccessToken(token);
				syncPbAuth(token);
				scheduleRefresh(token, get().refreshToken);

				try {
					const admin = await authApi.me();
					set({ admin, isAuthenticated: true, isLoading: false });
				} catch {
					// Access token expired — try refresh.
					if (refresh) {
						const ok = await get().refreshToken();
						set({ isLoading: false, isAuthenticated: ok });
					} else {
						get().clear();
						set({ isLoading: false });
					}
				}
			},

			clear: () => {
				setAccessToken(null);
				syncPbAuth(null);
				clearRefreshTimer();
				set({
					token: null,
					refresh: null,
					admin: null,
					isAuthenticated: false,
				});
			},
		}),
		{
			name: "dk-auth",
			partialize: (state) => ({
				token: state.token,
				refresh: state.refresh,
				admin: state.admin,
			}),
		},
	),
);
