import { useEffect, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";
import pb from "@/lib/pb";
import { invalidateCollectionQueries } from "@/lib/query-invalidation";
import type { CollectionName } from "@/types/pocketbase";

const DEFAULT_DEBOUNCE_MS = 500;

/**
 * Subscribes to PocketBase realtime events (SSE) for a collection and
 * automatically invalidates the corresponding TanStack Query cache when
 * records are created, updated or deleted.
 *
 * Invalidation is debounced so that burst updates (e.g. agent reporting
 * many records at once) are collapsed into a single refetch.
 *
 * @param name Collection name to subscribe to.
 * @param debounceMs Custom debounce interval. Use higher values (e.g. 3000)
 *   for collections that receive frequent metric-driven updates to reduce
 *   unnecessary re-renders.
 *
 * @example
 * function ServerList() {
 *   useRealtime("servers");
 *   const { data } = useCollection("servers");
 *   // `data` is automatically refreshed when a server record changes.
 * }
 */
export function useRealtime(name: CollectionName, debounceMs?: number) {
	const queryClient = useQueryClient();
	const timerRef = useRef<ReturnType<typeof setTimeout>>(undefined);
	const interval = debounceMs ?? DEFAULT_DEBOUNCE_MS;

	useEffect(() => {
		let unsubscribe: (() => Promise<void>) | undefined;

		pb.collection(name)
			.subscribe("*", () => {
				clearTimeout(timerRef.current);
				timerRef.current = setTimeout(() => {
					invalidateCollectionQueries(queryClient, name);
				}, interval);
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
	}, [name, queryClient, interval]);
}
