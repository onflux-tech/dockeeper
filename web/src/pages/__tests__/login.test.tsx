import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router";

// Mock modules before importing the component.
vi.mock("@/stores/auth-store", () => ({
	useAuthStore: () => ({
		isAuthenticated: false,
		login: vi.fn(),
	}),
}));

vi.mock("@/lib/api", () => ({
	authApi: { status: () => Promise.resolve({ needs_setup: false }) },
	ApiRequestError: class ApiRequestError extends Error {
		status: number;
		constructor(msg: string, status: number) {
			super(msg);
			this.status = status;
		}
	},
}));

import LoginPage from "../login";

function renderLogin() {
	return render(
		<MemoryRouter>
			<LoginPage />
		</MemoryRouter>,
	);
}

describe("LoginPage", () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it("renders the Dockeeper heading", () => {
		renderLogin();
		expect(screen.getByText("Dockeeper")).toBeInTheDocument();
	});

	it("renders email and password fields", () => {
		renderLogin();
		expect(screen.getByLabelText("Email")).toBeInTheDocument();
		expect(screen.getByLabelText("Password")).toBeInTheDocument();
	});

	it("renders the sign in button", () => {
		renderLogin();
		expect(screen.getByRole("button", { name: /sign in/i })).toBeInTheDocument();
	});

	it("shows description text", () => {
		renderLogin();
		expect(screen.getByText("Sign in to manage your containers")).toBeInTheDocument();
	});
});
