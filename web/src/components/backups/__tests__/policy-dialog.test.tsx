import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

vi.mock("@/hooks/use-backups", () => ({
	useCreatePolicy: () => ({ mutate: vi.fn(), isPending: false }),
	useUpdatePolicy: () => ({ mutate: vi.fn(), isPending: false }),
}));

import { PolicyDialog } from "../policy-dialog";
import type { HostAppRecord } from "@/types/pocketbase";

const hostApps = [
	{
		id: "app1",
		collectionId: "col1",
		collectionName: "host_apps",
		server: "srv1",
		source: "systemd",
		name: "nginx.service",
		unit: "nginx.service",
		pm_id: "",
		status: "running",
		pids: [],
		ports: [80],
		cpu_percent: 0,
		mem_bytes: 0,
		disk_read_bytes: 0,
		disk_write_bytes: 0,
		restarts: 0,
		uptime_seconds: 0,
		user: "root",
		cwd: "/",
		exec_path: "/usr/sbin/nginx",
		log_path: "",
		last_seen: "",
	} as HostAppRecord,
];

describe("PolicyDialog", () => {
	it("offers the host app target fields", async () => {
		const user = userEvent.setup();

		render(
			<PolicyDialog
				open
				onOpenChange={() => {}}
				serverID="srv1"
				volumes={[]}
				hostApps={hostApps}
			/>,
		);

		await user.click(screen.getAllByRole("combobox")[0]);
		await user.click(await screen.findByText("Host app"));

		expect(screen.getByText("Application")).toBeInTheDocument();

		const combos = screen.getAllByRole("combobox");
		await user.click(combos[2]);
		expect(await screen.findByText("nginx.service")).toBeInTheDocument();
		expect(screen.getByPlaceholderText(/etc\/nginx/)).toBeInTheDocument();
	});
});
