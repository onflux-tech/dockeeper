import { useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { getAccessToken } from "@/lib/api";

export interface SearchResultItem {
	id: string;
	name: string;
	subtitle: string;
	href: string;
}

export interface SearchResultGroup {
	type: string;
	label: string;
	total: number;
	viewAllHref: string;
	items: SearchResultItem[];
}

interface SearchResponse {
	query: string;
	groups: SearchResultGroup[];
}

function authHeaders(): Record<string, string> {
	const token = getAccessToken();
	const h: Record<string, string> = {};
	if (token) h["Authorization"] = token;
	return h;
}

export function useGlobalSearch(query: string) {
	const [debouncedQuery, setDebouncedQuery] = useState("");
	const abortRef = useRef<AbortController | undefined>(undefined);

	// Debounce 300ms
	useEffect(() => {
		const trimmed = query.trim();
		const timer = setTimeout(
			() => setDebouncedQuery(trimmed.length >= 2 ? trimmed : ""),
			trimmed.length >= 2 ? 500 : 0,
		);
		return () => clearTimeout(timer);
	}, [query]);

	const result = useQuery<SearchResponse>({
		queryKey: ["global-search", debouncedQuery],
		queryFn: async () => {
			// Cancel previous in-flight request
			if (abortRef.current) abortRef.current.abort();
			const controller = new AbortController();
			abortRef.current = controller;

			const res = await fetch(`/api/dk/search?q=${encodeURIComponent(debouncedQuery)}&limit=5`, {
				headers: authHeaders(),
				signal: controller.signal,
			});
			if (!res.ok) {
				const body = await res.json().catch(() => ({ error: res.statusText }));
				throw new Error(body.error || res.statusText);
			}
			return res.json();
		},
		enabled: debouncedQuery.length >= 2,
		staleTime: 30_000,
	});

	return {
		results: result.data?.groups ?? [],
		isLoading: result.isFetching,
		isError: result.isError,
	};
}
