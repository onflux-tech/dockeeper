import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

const fixtures = vi.hoisted(() => ({
	apps: [
		{
			id: "app1",
			server: "srv1",
			source: "pm2",
			name: "api-takeflow",
			unit: "",
			pm_id: "3",
			status: "running",
			pids: [4122],
			ports: [3000],
			cpu_percent: 8.2,
			mem_bytes: 2048,
			disk_read_bytes: 0,
			disk_write_bytes: 0,
			restarts: 1,
			uptime_seconds: 3600,
			user: "root",
			cwd: "/srv/api",
			exec_path: "/srv/api/index.js",
			log_path: "",
			last_seen: "",
		},
	],
	servers: [
		{
			id: "srv1",
			name: "TakeFlow PROD",
			hostname: "takeflow",
			status: "online",
			view_scope: "auto",
		},
	],
}));

vi.mock("@/hooks/use-collection", () => ({
	useCollection: (name: string) => {
		if (name === "host_apps") {
			return {
				data: { items: fixtures.apps, totalItems: fixtures.apps.length, totalPages: 1 },
				isLoading: false,
			};
		}
		if (name === "servers") {
			return {
				data: {
					items: fixtures.servers,
					totalItems: fixtures.servers.length,
					totalPages: 1,
				},
				isLoading: false,
			};
		}
		return { data: { items: [], totalItems: 0, totalPages: 0 }, isLoading: false };
	},
}));

vi.mock("@/hooks/use-realtime", () => ({
	useRealtime: () => {},
}));

import AppsPage from "../apps";

describe("AppsPage", () => {
	it("renders an app with its server name", () => {
		const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
		render(
			<QueryClientProvider client={queryClient}>
				<MemoryRouter>
					<AppsPage />
				</MemoryRouter>
			</QueryClientProvider>,
		);
		expect(screen.getByText("api-takeflow")).toBeInTheDocument();
		expect(screen.getByText("TakeFlow PROD")).toBeInTheDocument();
		expect(screen.getByText("3000")).toBeInTheDocument();
	});
});
