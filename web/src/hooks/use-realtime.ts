import { useEffect, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";
import pb from "@/lib/pb";
import { invalidateCollectionQueries } from "@/lib/query-invalidation";
import type { CollectionName } from "@/types/pocketbase";

const DEBOUNCE_MS = 500;

/**
 * Subscribes to PocketBase realtime events (SSE) for a collection and
 * automatically invalidates the corresponding TanStack Query cache when
 * records are created, updated or deleted.
 *
 * Invalidation is debounced so that burst updates (e.g. agent reporting
 * many records at once) are collapsed into a single refetch.
 *
 * @example
 * function ServerList() {
 *   useRealtime("servers");
 *   const { data } = useCollection("servers");
 *   // `data` is automatically refreshed when a server record changes.
 * }
 */
export function useRealtime(name: CollectionName) {
	const queryClient = useQueryClient();
	const timerRef = useRef<ReturnType<typeof setTimeout>>(undefined);

	useEffect(() => {
		let unsubscribe: (() => Promise<void>) | undefined;

		pb.collection(name)
			.subscribe("*", () => {
				clearTimeout(timerRef.current);
				timerRef.current = setTimeout(() => {
					invalidateCollectionQueries(queryClient, name);
				}, DEBOUNCE_MS);
			})
			.then((unsub) => {
				unsubscribe = unsub;
			});

		return () => {
			clearTimeout(timerRef.current);
			if (unsubscribe) {
				void unsubscribe();
			}
		};
	}, [name, queryClient]);
}
