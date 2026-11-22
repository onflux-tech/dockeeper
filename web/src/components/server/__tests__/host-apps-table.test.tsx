import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { HostAppsTable } from "../host-apps-table";
import type { HostAppRecord } from "@/types/pocketbase";

const appFixture: HostAppRecord = {
	id: "app1",
	collectionId: "host_apps",
	collectionName: "host_apps",
	created: "",
	updated: "",
	server: "srv1",
	source: "systemd",
	name: "nginx.service",
	unit: "nginx.service",
	pm_id: "",
	status: "running",
	pids: [42],
	ports: [80, 443],
	cpu_percent: 1.2,
	mem_bytes: 2048,
	disk_read_bytes: 0,
	disk_write_bytes: 0,
	restarts: 0,
	uptime_seconds: 100,
	user: "root",
	cwd: "/",
	exec_path: "/usr/sbin/nginx",
	log_path: "",
	last_seen: "",
};

describe("HostAppsTable", () => {
	it("renders an app row with source, status and ports", () => {
		render(<HostAppsTable apps={[appFixture]} isLoading={false} />);
		expect(screen.getByText("nginx.service")).toBeInTheDocument();
		expect(screen.getByText("systemd")).toBeInTheDocument();
		expect(screen.getByText("running")).toBeInTheDocument();
		expect(screen.getByText("80, 443")).toBeInTheDocument();
	});

	it("renders the empty state", () => {
		render(<HostAppsTable apps={[]} isLoading={false} />);
		expect(screen.getByText("No applications reported.")).toBeInTheDocument();
	});

	it("limits rows and shows the total when a limit is set", () => {
		const apps = Array.from({ length: 8 }, (_, index) => ({
			...appFixture,
			id: `app${index}`,
			name: `app-${index}.service`,
		}));
		render(<HostAppsTable apps={apps} isLoading={false} limit={6} />);

		expect(screen.getByText("app-5.service")).toBeInTheDocument();
		expect(screen.queryByText("app-6.service")).not.toBeInTheDocument();
		expect(screen.getByText("Showing 6 of 8")).toBeInTheDocument();
	});
});
