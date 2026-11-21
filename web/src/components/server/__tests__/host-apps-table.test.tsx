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
});
