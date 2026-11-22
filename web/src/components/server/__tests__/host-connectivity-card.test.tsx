import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { HostConnectivityCard } from "../host-connectivity-card";
import type { ServerRecord } from "@/types/pocketbase";

const server = {
	id: "srv",
	name: "srv",
	status: "online",
	connectivity_status: "healthy",
	connectivity_checked_at: new Date().toISOString(),
	connectivity_details: {
		internet: { ok: true, ms: 24 },
		dns: { ok: true, ms: 12 },
		gateway: { ok: false, ms: 1 },
		http: { ok: 3, total: 3, latency_ms: 18 },
	},
} as ServerRecord;

describe("HostConnectivityCard", () => {
	it("renders the probes, the http summary and the last check time", () => {
		render(<HostConnectivityCard server={server} />);

		expect(screen.getByText("Connectivity")).toBeInTheDocument();
		expect(screen.getByText("healthy")).toBeInTheDocument();
		expect(screen.getByText("Internet")).toBeInTheDocument();
		expect(screen.getByText("24 ms")).toBeInTheDocument();
		expect(screen.getByText("3/3 OK · 18 ms")).toBeInTheDocument();
		expect(screen.getByText(/Checked/)).toBeInTheDocument();
	});

	it("shows a dash for probes without data", () => {
		render(
			<HostConnectivityCard
				server={
					{
						id: "srv",
						name: "srv",
						connectivity_status: "unknown",
						connectivity_details: null,
					} as ServerRecord
				}
			/>,
		);

		expect(screen.getByText("unknown")).toBeInTheDocument();
		expect(screen.getAllByText("—")).toHaveLength(3);
	});
});
