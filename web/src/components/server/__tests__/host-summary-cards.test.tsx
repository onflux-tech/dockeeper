import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { HostSummaryCards } from "../host-summary-cards";
import type { MetricPoint } from "@/hooks/use-server-metrics";

const metric: MetricPoint = {
	timestamp: "",
	cpu_percent: 45,
	mem_percent: 60,
	disk_used: 1024,
	disk_total: 4096,
	net_rx: 512,
	net_tx: 256,
	disk_io_read: 0,
	disk_io_write: 0,
	swap_used: 0,
	swap_total: 0,
	mem_available: 2048,
	partitions: [],
	temperatures: [],
};

describe("HostSummaryCards", () => {
	it("shows the value with the metric name and detail below it", () => {
		render(<HostSummaryCards metrics={[metric]} />);

		expect(screen.getByText("45.0%")).toBeInTheDocument();
		expect(screen.getByText(/CPU · 1 samples in range/)).toBeInTheDocument();
		expect(screen.getByText(/Network · ↑/)).toBeInTheDocument();
	});

	it("renders the empty state when there are no metrics", () => {
		render(<HostSummaryCards metrics={[]} />);
		expect(screen.getAllByText(/No metrics yet/)).toHaveLength(4);
	});
});
