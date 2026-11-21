import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { StatsOverview } from "../stats-overview";

describe("StatsOverview", () => {
	it("shows the total container count from the collection total", () => {
		render(<StatsOverview servers={[]} totalContainers={438} isLoading={false} />);
		expect(screen.getByText("Containers")).toBeInTheDocument();
		expect(screen.getByText("438")).toBeInTheDocument();
	});
});
