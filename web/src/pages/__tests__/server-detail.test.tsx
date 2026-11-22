import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Route, Routes } from "react-router";

const mocks = vi.hoisted(() => ({
	agentVersion: "v0.9.0",
	latest: "v0.9.0",
	orchestrator: "docker",
}));

vi.mock("@/hooks/use-record", () => ({
	useRecord: () => ({
		data: {
			id: "abc",
			name: "srv",
			status: "online",
			agent_version: mocks.agentVersion,
			orchestrator: mocks.orchestrator,
		},
		isLoading: false,
	}),
}));

vi.mock("@/hooks/use-collection", () => ({
	useCollection: () => ({ data: { items: [], totalItems: 0 }, isLoading: false }),
}));

vi.mock("@/hooks/use-realtime", () => ({
	useRealtime: () => {},
}));

vi.mock("@/hooks/use-server-metrics", () => ({
	useServerMetrics: () => ({ data: [], isLoading: false }),
}));

vi.mock("@/hooks/use-settings", () => ({
	useAgentUpdate: () => ({ mutate: vi.fn(), isPending: false }),
	useLatestRelease: () => ({ data: { latest: mocks.latest } }),
}));

vi.mock("@/components/server/server-info", () => ({ ServerInfo: () => null }));
vi.mock("@/components/server/metrics-charts", () => ({ MetricsCharts: () => null }));
vi.mock("@/components/server/container-table", () => ({ ContainerTable: () => null }));
vi.mock("@/components/server/edit-server-dialog", () => ({ EditServerDialog: () => null }));
vi.mock("@/components/tag-selector", () => ({ TagSelector: () => null }));

import ServerDetailPage from "../server-detail";

function renderPage() {
	const queryClient = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	return render(
		<QueryClientProvider client={queryClient}>
			<MemoryRouter initialEntries={["/servers/abc"]}>
				<Routes>
					<Route path="/servers/:id" element={<ServerDetailPage />} />
				</Routes>
			</MemoryRouter>
		</QueryClientProvider>,
	);
}

describe("ServerDetailPage agent update", () => {
	it("hides the update button when the agent runs the latest release", () => {
		mocks.agentVersion = "v0.9.0";
		mocks.latest = "v0.9.0";
		renderPage();
		expect(screen.queryByText("Update Agent")).not.toBeInTheDocument();
	});

	it("shows the update button when the agent is older than the latest release", () => {
		mocks.agentVersion = "develop";
		mocks.latest = "v0.9.0";
		renderPage();
		expect(screen.getByText("Update Agent")).toBeInTheDocument();
	});

	it("shows the host tab when the server has no container runtime", () => {
		mocks.orchestrator = "none";
		renderPage();
		expect(screen.getByRole("tab", { name: "Host" })).toBeInTheDocument();
	});

	it("replaces the containers table with the no runtime state", async () => {
		const user = userEvent.setup();
		mocks.orchestrator = "none";
		renderPage();
		await user.click(screen.getByRole("tab", { name: /Containers/ }));
		expect(screen.getByText("No container runtime")).toBeInTheDocument();
	});

	it("hides the host tab and keeps the container count on docker servers", () => {
		mocks.orchestrator = "docker";
		renderPage();
		expect(screen.queryByRole("tab", { name: "Host" })).not.toBeInTheDocument();
		expect(screen.getByRole("tab", { name: "Containers (0)" })).toBeInTheDocument();
	});

	it("omits the misleading container count when there is no runtime", () => {
		mocks.orchestrator = "none";
		renderPage();
		expect(screen.getByRole("tab", { name: "Containers" })).toBeInTheDocument();
		expect(screen.queryByRole("tab", { name: /Containers \(/ })).not.toBeInTheDocument();
	});
});
