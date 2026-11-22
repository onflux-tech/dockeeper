import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

vi.mock("@/hooks/use-health-checks", () => ({
	useHealthChecks: () => ({ data: [], isLoading: false }),
	useHealthCheckLogs: () => ({ data: { items: [], total_items: 0 }, isLoading: false }),
	useCreateHealthCheck: () => ({ mutate: vi.fn(), isPending: false }),
	useUpdateHealthCheck: () => ({ mutate: vi.fn(), isPending: false }),
	useDeleteHealthCheck: () => ({ mutate: vi.fn(), isPending: false }),
	useTriggerHealthCheck: () => ({ mutate: vi.fn(), isPending: false }),
}));

vi.mock("@/hooks/use-collection", () => ({
	useCollection: () => ({ data: { items: [], totalItems: 0, totalPages: 0 }, isLoading: false }),
}));

import HealthChecksPage from "../health-checks";

describe("HealthChecksPage", () => {
	it("offers the host app target in the health dialog", async () => {
		const user = userEvent.setup();
		const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

		render(
			<QueryClientProvider client={queryClient}>
				<MemoryRouter>
					<HealthChecksPage />
				</MemoryRouter>
			</QueryClientProvider>,
		);

		await user.click(screen.getByRole("button", { name: /new check/i }));

		const triggers = screen.getAllByRole("combobox");
		await user.click(triggers[0]);

		expect((await screen.findAllByText("Application")).length).toBeGreaterThan(0);
	});
});
