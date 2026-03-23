export interface AdminData {
	id: string;
	email: string;
	created: string;
	updated: string;
}

export interface AuthResponse {
	token: string;
	refresh: string;
	admin: AdminData;
}

export interface AuthStatus {
	needs_setup: boolean;
}

export interface ApiError {
	error: string;
}
