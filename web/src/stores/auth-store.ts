import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { AdminData } from "@/types/auth";
import { authApi, setAccessToken } from "@/lib/api";

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
				const res = await authApi.login(email, password);
				setAccessToken(res.token);
				set({
					token: res.token,
					refresh: res.refresh,
					admin: res.admin,
					isAuthenticated: true,
				});
			},

			logout: async () => {
				try {
					await authApi.logout();
				} catch {
					// Ignore errors on logout — we clear state regardless.
				}
				setAccessToken(null);
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
					set({
						token: res.token,
						refresh: res.refresh,
						admin: res.admin,
						isAuthenticated: true,
					});
					return true;
				} catch {
					get().clear();
					return false;
				}
			},

			checkAuth: async () => {
				const { token, refresh } = get();

				if (!token) {
					set({ isLoading: false, isAuthenticated: false });
					return;
				}

				setAccessToken(token);

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
