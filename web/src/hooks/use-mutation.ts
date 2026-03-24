import { useMutation, useQueryClient } from "@tanstack/react-query";
import pb from "@/lib/pb";
import { invalidateCollectionQueries } from "@/lib/query-invalidation";
import type { Collections, CollectionName } from "@/types/pocketbase";

/**
 * Creates a new record in a PocketBase collection.
 * Invalidates the collection's query cache on success.
 *
 * @example
 * const create = useCreateRecord("alert_rules");
 * create.mutate({ name: "High CPU", metric: "cpu_percent", ... });
 */
export function useCreateRecord<T extends CollectionName>(name: T) {
	const queryClient = useQueryClient();

	return useMutation<Collections[T], Error, Partial<Collections[T]>>({
		mutationFn: (data) => pb.collection(name).create(data),
		onSuccess: () => {
			invalidateCollectionQueries(queryClient, name);
		},
	});
}

/**
 * Updates an existing record in a PocketBase collection.
 * Invalidates the collection's query cache on success.
 *
 * @example
 * const update = useUpdateRecord("servers");
 * update.mutate({ id: "abc123", name: "new-name" });
 */
export function useUpdateRecord<T extends CollectionName>(name: T) {
	const queryClient = useQueryClient();

	return useMutation<Collections[T], Error, { id: string } & Partial<Collections[T]>>({
		mutationFn: ({ id, ...data }) => pb.collection(name).update(id, data),
		onSuccess: () => {
			invalidateCollectionQueries(queryClient, name);
		},
	});
}

/**
 * Deletes a record from a PocketBase collection.
 * Invalidates the collection's query cache on success.
 *
 * @example
 * const remove = useDeleteRecord("containers");
 * remove.mutate("record-id");
 */
export function useDeleteRecord(name: CollectionName) {
	const queryClient = useQueryClient();

	return useMutation<boolean, Error, string>({
		mutationFn: (id) => pb.collection(name).delete(id),
		onSuccess: () => {
			invalidateCollectionQueries(queryClient, name);
		},
	});
}
