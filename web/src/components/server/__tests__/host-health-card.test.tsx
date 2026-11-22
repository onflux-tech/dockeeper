import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { HostHealthCard } from "../host-health-card";
import type { HealthCheckRecord } from "@/types/pocketbase";

function check(overrides: Partial<HealthCheckRecord>): HealthCheckRecord {
	return {
		id: "hc",
		name: "nginx",
		status: "healthy",
		check_type: "http",
		host: "",
		...overrides,
	} as HealthCheckRecord;
}

describe("HostHealthCard", () => {
	it("shows the real status of each check and the failing count", () => {
		render(
			<HostHealthCard
				checks={[
					check({ id: "a", name: "nginx", status: "healthy" }),
					check({ id: "b", name: "api", status: "unhealthy", check_type: "tcp" }),
					check({ id: "c", name: "worker", status: "unknown", check_type: "process" }),
				]}
			/>,
		);

		expect(screen.getByText("1 failing")).toBeInTheDocument();
		expect(screen.getByText("nginx")).toBeInTheDocument();
		expect(screen.getByText("OK")).toBeInTheDocument();
		expect(screen.getByText("FAIL")).toBeInTheDocument();
		expect(screen.getByText("—")).toBeInTheDocument();
	});

	it("caps the list at three checks", () => {
		render(
			<HostHealthCard
				checks={[1, 2, 3, 4].map((n) => check({ id: String(n), name: `check-${n}` }))}
			/>,
		);

		expect(screen.getByText("check-3")).toBeInTheDocument();
		expect(screen.queryByText("check-4")).not.toBeInTheDocument();
	});

	it("renders the empty state", () => {
		render(<HostHealthCard checks={[]} />);

		expect(screen.getByText("none")).toBeInTheDocument();
		expect(screen.getByText("No health checks configured.")).toBeInTheDocument();
	});
});
