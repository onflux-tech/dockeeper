import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router";

// Mock hooks that call PocketBase.
vi.mock("@/hooks/use-collection", () => ({
	useCollection: () => ({ data: null, isLoading: false }),
}));

vi.mock("@/hooks/use-realtime", () => ({
	useRealtime: () => {},
}));

import DashboardPage from "../dashboard";

function renderDashboard() {
	const queryClient = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	return render(
		<QueryClientProvider client={queryClient}>
			<MemoryRouter>
				<DashboardPage />
			</MemoryRouter>
		</QueryClientProvider>,
	);
}

describe("DashboardPage", () => {
	it("renders the Dashboard heading", () => {
		renderDashboard();
		expect(screen.getByText("Dashboard")).toBeInTheDocument();
	});

	it("renders the Servers section", () => {
		renderDashboard();
		expect(screen.getByText("Servers")).toBeInTheDocument();
	});

	it("shows empty state when no servers", () => {
		renderDashboard();
		expect(screen.getByText("No servers registered yet.")).toBeInTheDocument();
	});

	it("shows enrollment prompt in empty state", () => {
		renderDashboard();
		expect(screen.getByText(/enroll an agent/i)).toBeInTheDocument();
	});
});
