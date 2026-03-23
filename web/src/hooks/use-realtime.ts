import { useEffect } from "react";
import { useQueryClient } from "@tanstack/react-query";
import pb from "@/lib/pb";
import type { CollectionName } from "@/types/pocketbase";

/**
 * Subscribes to PocketBase realtime events (SSE) for a collection and
 * automatically invalidates the corresponding TanStack Query cache when
 * records are created, updated or deleted.
 *
 * Call this hook in any component that displays collection data to keep it
 * in sync without manual polling.
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

	useEffect(() => {
		let unsubscribe: (() => Promise<void>) | undefined;

		pb.collection(name)
			.subscribe("*", () => {
				void queryClient.invalidateQueries({
					queryKey: ["collection", name],
				});
			})
			.then((unsub) => {
				unsubscribe = unsub;
			});

		return () => {
			if (unsubscribe) {
				void unsubscribe();
			}
		};
	}, [name, queryClient]);
}
