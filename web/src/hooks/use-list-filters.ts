import { useCallback, useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router";
import {
	andFilters,
	containsAnyFilter,
	eqFilter,
} from "@/lib/pocketbase-filters";

const VALID_PER_PAGE = [25, 50, 100] as const;
type PerPage = (typeof VALID_PER_PAGE)[number];

export interface FilterConfig {
	/** Default sort field (e.g. "name", "-created") */
	defaultSort: string;
	/** Fields to search with `containsAnyFilter` */
	searchFields: string[];
	/** Custom filter keys that map to `eqFilter` (e.g. { server: "server", status: "status" }) */
	customFilters?: Record<string, string>;
}

export interface ListFilters {
	search: string;
	page: number;
	perPage: PerPage;
	sort: string;
	custom: Record<string, string>;
}

export interface UseListFiltersReturn {
	filters: ListFilters;
	setSearch: (value: string) => void;
	setPage: (page: number) => void;
	setPerPage: (perPage: number) => void;
	setSort: (sort: string) => void;
	setCustom: (key: string, value: string) => void;
	/** Ready-to-use options for useCollection */
	pbOptions: {
		page: number;
		perPage: number;
		sort: string;
		filter: string;
	};
}

function clampPage(value: number): number {
	return Math.max(1, Math.floor(value) || 1);
}

function clampPerPage(value: number): PerPage {
	if (VALID_PER_PAGE.includes(value as PerPage)) return value as PerPage;
	return 25;
}

export function useListFilters(config: FilterConfig): UseListFiltersReturn {
	const [searchParams, setSearchParams] = useSearchParams();
	const debounceRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

	// Read current values from URL
	const rawSearch = searchParams.get("search") ?? "";
	const page = clampPage(Number(searchParams.get("page")) || 1);
	const perPage = clampPerPage(Number(searchParams.get("perPage")) || 25);
	const sort = searchParams.get("sort") || config.defaultSort;

	// Read custom filters from URL
	const custom: Record<string, string> = {};
	if (config.customFilters) {
		for (const key of Object.keys(config.customFilters)) {
			custom[key] = searchParams.get(key) ?? "all";
		}
	}

	// Local search state for immediate input feedback while debouncing URL updates
	const [localSearch, setLocalSearch] = useState(rawSearch);

	// Sync local search when URL changes externally (e.g. back/forward navigation)
	useEffect(() => {
		setLocalSearch(rawSearch);
	}, [rawSearch]);

	// Cleanup debounce on unmount
	useEffect(() => {
		return () => {
			if (debounceRef.current) clearTimeout(debounceRef.current);
		};
	}, []);

	const updateParams = useCallback(
		(updates: Record<string, string | null>) => {
			setSearchParams(
				(prev) => {
					const next = new URLSearchParams(prev);
					for (const [key, value] of Object.entries(updates)) {
						if (value === null || value === "") {
							next.delete(key);
						} else {
							next.set(key, value);
						}
					}
					// Clean defaults
					if (next.get("page") === "1") next.delete("page");
					if (next.get("perPage") === "25") next.delete("perPage");
					if (next.get("sort") === config.defaultSort) next.delete("sort");
					if (config.customFilters) {
						for (const key of Object.keys(config.customFilters)) {
							if (next.get(key) === "all") next.delete(key);
						}
					}
					return next;
				},
				{ replace: true },
			);
		},
		[setSearchParams, config.defaultSort, config.customFilters],
	);

	const setSearch = useCallback(
		(value: string) => {
			setLocalSearch(value);
			if (debounceRef.current) clearTimeout(debounceRef.current);
			debounceRef.current = setTimeout(() => {
				updateParams({ search: value || null, page: null });
			}, 500);
		},
		[updateParams],
	);

	const setPage = useCallback(
		(p: number) => {
			updateParams({ page: String(clampPage(p)) });
		},
		[updateParams],
	);

	const setPerPage = useCallback(
		(pp: number) => {
			updateParams({ perPage: String(clampPerPage(pp)), page: null });
		},
		[updateParams],
	);

	const setSort = useCallback(
		(s: string) => {
			updateParams({ sort: s, page: null });
		},
		[updateParams],
	);

	const setCustom = useCallback(
		(key: string, value: string) => {
			updateParams({ [key]: value === "all" ? null : value, page: null });
		},
		[updateParams],
	);

	// Build PocketBase filter string
	const searchFilter = containsAnyFilter(config.searchFields, localSearch || undefined);
	const customFilterParts: string[] = [];
	if (config.customFilters) {
		for (const [key, field] of Object.entries(config.customFilters)) {
			const val = custom[key];
			if (val && val !== "all") {
				customFilterParts.push(eqFilter(field, val));
			}
		}
	}

	const filter = andFilters([searchFilter, ...customFilterParts]);

	return {
		filters: { search: localSearch, page, perPage, sort, custom },
		setSearch,
		setPage,
		setPerPage,
		setSort,
		setCustom,
		pbOptions: { page, perPage, sort, filter },
	};
}
