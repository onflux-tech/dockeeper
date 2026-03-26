import { renderHook, act } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter } from "react-router";
import type { ReactNode } from "react";

// We need to mock the pocketbase-filters module
vi.mock("@/lib/pocketbase-filters", () => ({
	andFilters: (parts: string[]) => parts.filter(Boolean).join(" && "),
	containsAnyFilter: (fields: string[], value?: string) => {
		if (!value) return "";
		return fields.map((f) => `${f} ~ '${value}'`).join(" || ");
	},
	eqFilter: (field: string, value: string) => `${field} = '${value}'`,
}));

import { useListFilters } from "../use-list-filters";

function wrapper({ children }: { children: ReactNode }) {
	return <MemoryRouter>{children}</MemoryRouter>;
}

function wrapperWithParams(initialEntries: string[]) {
	return function Wrapper({ children }: { children: ReactNode }) {
		return <MemoryRouter initialEntries={initialEntries}>{children}</MemoryRouter>;
	};
}

describe("useListFilters", () => {
	beforeEach(() => {
		vi.useFakeTimers();
	});

	it("returns default values on mount", () => {
		const { result } = renderHook(
			() =>
				useListFilters({
					defaultSort: "name",
					searchFields: ["name"],
				}),
			{ wrapper },
		);

		expect(result.current.filters.search).toBe("");
		expect(result.current.filters.page).toBe(1);
		expect(result.current.filters.perPage).toBe(25);
		expect(result.current.filters.sort).toBe("name");
		expect(result.current.pbOptions.filter).toBe("");
	});

	it("reads URL params on mount", () => {
		const { result } = renderHook(
			() =>
				useListFilters({
					defaultSort: "name",
					searchFields: ["name"],
				}),
			{ wrapper: wrapperWithParams(["/?search=test&page=3&perPage=50"]) },
		);

		expect(result.current.filters.search).toBe("test");
		expect(result.current.filters.page).toBe(3);
		expect(result.current.filters.perPage).toBe(50);
	});

	it("clamps invalid page to 1", () => {
		const { result } = renderHook(
			() =>
				useListFilters({
					defaultSort: "name",
					searchFields: ["name"],
				}),
			{ wrapper: wrapperWithParams(["/?page=-5"]) },
		);

		expect(result.current.filters.page).toBe(1);
	});

	it("clamps invalid perPage to 25", () => {
		const { result } = renderHook(
			() =>
				useListFilters({
					defaultSort: "name",
					searchFields: ["name"],
				}),
			{ wrapper: wrapperWithParams(["/?perPage=999"]) },
		);

		expect(result.current.filters.perPage).toBe(25);
	});

	it("setPage updates page", () => {
		const { result } = renderHook(
			() =>
				useListFilters({
					defaultSort: "name",
					searchFields: ["name"],
				}),
			{ wrapper },
		);

		act(() => result.current.setPage(5));
		expect(result.current.filters.page).toBe(5);
	});

	it("setPerPage updates perPage and resets page", () => {
		const { result } = renderHook(
			() =>
				useListFilters({
					defaultSort: "name",
					searchFields: ["name"],
				}),
			{ wrapper: wrapperWithParams(["/?page=3"]) },
		);

		act(() => result.current.setPerPage(100));
		expect(result.current.filters.perPage).toBe(100);
		expect(result.current.filters.page).toBe(1);
	});

	it("setSearch debounces and resets page", () => {
		const { result } = renderHook(
			() =>
				useListFilters({
					defaultSort: "name",
					searchFields: ["name"],
				}),
			{ wrapper },
		);

		act(() => result.current.setSearch("test"));
		// Local search updates immediately
		expect(result.current.filters.search).toBe("test");

		// URL hasn't updated yet (debounced)
		// After debounce fires
		act(() => vi.advanceTimersByTime(300));
		expect(result.current.pbOptions.filter).toContain("test");
	});

	it("setCustom updates custom filter and resets page", () => {
		const { result } = renderHook(
			() =>
				useListFilters({
					defaultSort: "name",
					searchFields: ["name"],
					customFilters: { server: "server" },
				}),
			{ wrapper: wrapperWithParams(["/?page=3"]) },
		);

		act(() => result.current.setCustom("server", "srv123"));
		expect(result.current.filters.custom.server).toBe("srv123");
		expect(result.current.filters.page).toBe(1);
		expect(result.current.pbOptions.filter).toContain("server = 'srv123'");
	});

	it("setCustom with 'all' clears the filter", () => {
		const { result } = renderHook(
			() =>
				useListFilters({
					defaultSort: "name",
					searchFields: ["name"],
					customFilters: { server: "server" },
				}),
			{ wrapper: wrapperWithParams(["/?server=srv123"]) },
		);

		act(() => result.current.setCustom("server", "all"));
		expect(result.current.filters.custom.server).toBe("all");
		expect(result.current.pbOptions.filter).not.toContain("server");
	});

	it("pbOptions contains correct page/perPage/sort", () => {
		const { result } = renderHook(
			() =>
				useListFilters({
					defaultSort: "-created",
					searchFields: ["name"],
				}),
			{ wrapper: wrapperWithParams(["/?page=2&perPage=50&sort=name"]) },
		);

		expect(result.current.pbOptions.page).toBe(2);
		expect(result.current.pbOptions.perPage).toBe(50);
		expect(result.current.pbOptions.sort).toBe("name");
	});
});
