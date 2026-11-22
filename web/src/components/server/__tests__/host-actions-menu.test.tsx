import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

vi.mock("@/hooks/use-permissions", () => ({
	useHasCapability: () => true,
}));

const mutate = vi.fn();
vi.mock("@/hooks/use-host-actions", () => ({
	useHostAction: () => ({ mutate }),
}));

import { HostActionsMenu } from "../host-actions-menu";
import type { HostAppRecord } from "@/types/pocketbase";

const app = {
	id: "app1",
	server: "srv1",
	source: "systemd",
	name: "nginx.service",
	unit: "nginx.service",
	pm_id: "",
	status: "running",
	pids: [42],
	ports: [80],
	cpu_percent: 0.5,
	mem_bytes: 1024,
	disk_read_bytes: 0,
	disk_write_bytes: 0,
	restarts: 0,
	uptime_seconds: 100,
	user: "root",
	cwd: "/",
	exec_path: "/usr/sbin/nginx",
	log_path: "",
	last_seen: "",
	collectionId: "",
	collectionName: "host_apps",
	created: "",
	updated: "",
} as HostAppRecord;

describe("HostActionsMenu", () => {
	it("asks for confirmation before stopping an app", async () => {
		const user = userEvent.setup();
		render(<HostActionsMenu serverID="srv1" app={app} serverOnline />);

		await user.click(screen.getByRole("button", { name: "Actions" }));
		await user.click(await screen.findByText("Stop"));

		expect(
			await screen.findByRole("heading", { name: /stop nginx\.service/i }),
		).toBeInTheDocument();
	});

	it("disables the menu when the server is offline", () => {
		render(<HostActionsMenu serverID="srv1" app={app} serverOnline={false} />);
		expect(screen.getByRole("button", { name: "Actions" })).toBeDisabled();
	});

	it("disables the menu when the server state is omitted", () => {
		render(<HostActionsMenu serverID="srv1" app={app} />);
		expect(screen.getByRole("button", { name: "Actions" })).toBeDisabled();
	});

	it("disables the menu for process apps", () => {
		render(<HostActionsMenu serverID="srv1" app={{ ...app, source: "process" }} serverOnline />);
		expect(screen.getByRole("button", { name: "Actions" })).toBeDisabled();
	});
});
