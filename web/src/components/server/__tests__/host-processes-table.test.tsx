import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { HostProcessesTable } from "../host-processes-table";
import type { HostProcessRecord } from "@/types/pocketbase";

const processFixture: HostProcessRecord = {
	id: "proc1",
	collectionId: "host_processes",
	collectionName: "host_processes",
	created: "",
	updated: "",
	server: "srv1",
	pid: 42,
	name: "nginx",
	app: "nginx.service",
	cpu_percent: 1.2,
	mem_bytes: 2048,
	threads: 2,
	last_seen: "",
};

describe("HostProcessesTable", () => {
	it("renders a process row", () => {
		render(<HostProcessesTable processes={[processFixture]} isLoading={false} />);
		expect(screen.getByText("nginx")).toBeInTheDocument();
		expect(screen.getByText("42")).toBeInTheDocument();
		expect(screen.getByText("nginx.service")).toBeInTheDocument();
	});

	it("renders the empty state", () => {
		render(<HostProcessesTable processes={[]} isLoading={false} />);
		expect(screen.getByText("No processes reported.")).toBeInTheDocument();
	});

	it("limits rows and shows the total when a limit is set", () => {
		const processes = Array.from({ length: 8 }, (_, index) => ({
			...processFixture,
			id: `proc${index}`,
			pid: 100 + index,
			name: `process-${index}`,
		}));
		render(<HostProcessesTable processes={processes} isLoading={false} limit={5} />);

		expect(screen.getByText("process-4")).toBeInTheDocument();
		expect(screen.queryByText("process-5")).not.toBeInTheDocument();
		expect(screen.getByText("Showing 5 of 8")).toBeInTheDocument();
	});
});
