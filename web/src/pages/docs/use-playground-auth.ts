import { createContext, useCallback, useContext, useMemo, useState } from "react";
import { useAuthStore } from "@/stores/auth-store";

const STORAGE_KEY = "dk-playground-key";

export type AuthMode = "api-key" | "session" | null;

export interface PlaygroundAuthCtx {
	apiKey: string;
	isAuthorized: boolean;
	authMode: AuthMode;
	authHeaders: Record<string, string>;
	authorize: (key: string) => Promise<{ ok: boolean; error?: string }>;
	deauthorize: () => void;
}

export const PlaygroundAuthContext = createContext<PlaygroundAuthCtx | null>(null);

export function usePlaygroundAuth(): PlaygroundAuthCtx {
	const ctx = useContext(PlaygroundAuthContext);
	if (!ctx) throw new Error("usePlaygroundAuth must be used within PlaygroundAuthProvider");
	return ctx;
}

export function usePlaygroundAuthValue(): PlaygroundAuthCtx {
	const [apiKey, setApiKey] = useState(() => sessionStorage.getItem(STORAGE_KEY) ?? "");
	const sessionToken = useAuthStore((s) => s.token);
	const isSessionAuth = useAuthStore((s) => s.isAuthenticated);

	const authorize = useCallback(async (key: string): Promise<{ ok: boolean; error?: string }> => {
		try {
			const res = await fetch("/api/dk/servers?perPage=1", {
				headers: { "X-API-Key": key },
			});
			if (res.status === 401) {
				return { ok: false, error: "Invalid API key" };
			}
			sessionStorage.setItem(STORAGE_KEY, key);
			setApiKey(key);
			return { ok: true };
		} catch {
			return { ok: false, error: "Network error — could not validate key" };
		}
	}, []);

	const deauthorize = useCallback(() => {
		sessionStorage.removeItem(STORAGE_KEY);
		setApiKey("");
	}, []);

	return useMemo(() => {
		const hasApiKey = apiKey.length > 0;
		const authMode: AuthMode = hasApiKey ? "api-key" : isSessionAuth ? "session" : null;
		const isAuthorized = hasApiKey || isSessionAuth;

		const authHeaders: Record<string, string> = {};
		if (hasApiKey) {
			authHeaders["X-API-Key"] = apiKey;
		} else if (isSessionAuth && sessionToken) {
			authHeaders["Authorization"] = sessionToken;
		}

		return { apiKey, isAuthorized, authMode, authHeaders, authorize, deauthorize };
	}, [apiKey, isSessionAuth, sessionToken, authorize, deauthorize]);
}
