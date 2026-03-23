import type { AuthResponse, AdminData, AuthStatus } from "@/types/auth";

const BASE = "/api/dk/auth";

let accessToken: string | null = null;

export function setAccessToken(token: string | null) {
	accessToken = token;
}

export function getAccessToken() {
	return accessToken;
}

async function request<T>(url: string, options: RequestInit = {}): Promise<T> {
	const headers: Record<string, string> = {
		"Content-Type": "application/json",
		...(options.headers as Record<string, string>),
	};

	if (accessToken) {
		headers["Authorization"] = accessToken;
	}

	const res = await fetch(url, { ...options, headers });

	if (!res.ok) {
		const body = await res.json().catch(() => ({ error: res.statusText }));
		throw new ApiRequestError(body.error || res.statusText, res.status);
	}

	if (res.status === 204) return undefined as T;

	return res.json();
}

export class ApiRequestError extends Error {
	status: number;
	constructor(message: string, status: number) {
		super(message);
		this.name = "ApiRequestError";
		this.status = status;
	}
}

export const authApi = {
	status: () => request<AuthStatus>(`${BASE}/status`),

	setup: (email: string, password: string) =>
		request<AuthResponse>(`${BASE}/setup`, {
			method: "POST",
			body: JSON.stringify({ email, password }),
		}),

	login: (email: string, password: string) =>
		request<AuthResponse>(`${BASE}/login`, {
			method: "POST",
			body: JSON.stringify({ email, password }),
		}),

	refresh: (refreshToken: string) => {
		const prev = accessToken;
		accessToken = refreshToken;
		return request<AuthResponse>(`${BASE}/refresh`, { method: "POST" }).finally(() => {
			accessToken = prev;
		});
	},

	logout: () => request<void>(`${BASE}/logout`, { method: "POST" }),

	me: () => request<AdminData>(`${BASE}/me`),
};
