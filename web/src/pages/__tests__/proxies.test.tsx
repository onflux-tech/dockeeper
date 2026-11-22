import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

vi.mock("@/hooks/use-collection", () => ({
	useCollection: (name: string) => {
		if (name === "servers") {
			return {
				data: {
					items: [{ id: "srv1", name: "TakeFlow PROD", hostname: "takeflow" }],
					totalItems: 1,
				},
				isLoading: false,
			};
		}
		return { data: { items: [], totalItems: 0 }, isLoading: false };
	},
}));

vi.mock("@/hooks/use-proxies", () => ({
	useProxyInstances: () => ({
		data: {
			items: [
				{
					id: "inst1",
					server: "srv1",
					kind: "nginx",
					source: "systemd",
					app: "nginx.service",
					version: "1.24",
					config_path: "/etc/nginx/nginx.conf",
					admin_url: "",
					status: "healthy",
					last_seen: "2026-11-21T20:00:00Z",
				},
			],
			totalItems: 1,
			totalPages: 1,
			page: 1,
			perPage: 25,
		},
		isLoading: false,
	}),
	useProxyRoutes: () => ({
		data: {
			items: [
				{
					id: "route1",
					instance: "inst1",
					server: "srv1",
					name: "app",
					hosts: ["app.example.com"],
					paths: ["/"],
					upstreams: ["127.0.0.1:3000"],
					tls: true,
					middlewares: [],
					status: "healthy",
					provider: "nginx",
				},
			],
			totalItems: 1,
			totalPages: 1,
			page: 1,
			perPage: 25,
		},
		isLoading: false,
	}),
}));

import ProxiesPage from "../proxies";

describe("ProxiesPage", () => {
	it("lists proxy instances and routes", () => {
		render(
			<MemoryRouter>
				<ProxiesPage />
			</MemoryRouter>,
		);

		expect(screen.getAllByText("nginx").length).toBeGreaterThan(0);
		expect(screen.getByText("app.example.com")).toBeInTheDocument();
		expect(screen.getByText("127.0.0.1:3000")).toBeInTheDocument();
	});
});
