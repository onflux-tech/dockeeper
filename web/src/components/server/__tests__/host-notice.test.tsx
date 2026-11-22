import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { MemoryRouter } from "react-router";
import { HostNotice } from "../host-notice";

describe("HostNotice", () => {
	it("explains the host-only mode and links to the docs", () => {
		render(
			<MemoryRouter>
				<HostNotice />
			</MemoryRouter>,
		);
		expect(screen.getByText(/No container runtime detected/)).toBeInTheDocument();
		expect(screen.getByText(/monitored in host-only mode/)).toBeInTheDocument();
		expect(screen.getByRole("link", { name: "Why?" })).toHaveAttribute("href", "/docs");
	});
});
