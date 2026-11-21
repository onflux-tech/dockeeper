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

vi.mock("@/hooks/use-dashboard", () => ({
	useDashboardLayout: () => ({ data: { widgets: [] }, isLoading: false }),
	useSaveDashboardLayout: () => ({ mutate: vi.fn(), isPending: false }),
	useResetDashboardLayout: () => ({ mutate: vi.fn(), isPending: false }),
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

	it("renders the servers summary", () => {
		renderDashboard();
		expect(screen.getByText("Total Servers")).toBeInTheDocument();
	});

	it("shows empty state when no servers", () => {
		renderDashboard();
		expect(screen.getByText("No servers registered yet.")).toBeInTheDocument();
	});

	it("shows the dashboard customization control", () => {
		renderDashboard();
		expect(screen.getByRole("button", { name: /customize/i })).toBeInTheDocument();
	});
});
