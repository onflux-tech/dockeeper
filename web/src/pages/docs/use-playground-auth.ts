import { createContext, useCallback, useContext, useMemo, useState } from "react";

const STORAGE_KEY = "dk-playground-key";

export interface PlaygroundAuthCtx {
	apiKey: string;
	isAuthorized: boolean;
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

	return useMemo(
		() => ({ apiKey, isAuthorized: apiKey.length > 0, authorize, deauthorize }),
		[apiKey, authorize, deauthorize],
	);
}
