import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";

vi.mock("@/hooks/use-alerts", () => ({
	useCreateRule: () => ({ mutate: vi.fn(), isPending: false }),
	useUpdateRule: () => ({ mutate: vi.fn(), isPending: false }),
}));

import { RuleDialog } from "../rule-dialog";

describe("RuleDialog", () => {
	it("offers the host app target for metric rules", async () => {
		const user = userEvent.setup();
		render(
			<QueryClientProvider client={new QueryClient()}>
				<RuleDialog
					open
					onOpenChange={() => {}}
					servers={[]}
					containers={[]}
					hostApps={[]}
					channels={[]}
				/>
			</QueryClientProvider>,
		);

		await user.click(screen.getAllByRole("combobox")[1]);

		expect((await screen.findAllByText("Host App")).length).toBeGreaterThan(0);
	});
});
