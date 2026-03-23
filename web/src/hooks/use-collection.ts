import { useQuery } from "@tanstack/react-query";
import type { ListResult, RecordListOptions } from "pocketbase";
import pb from "@/lib/pb";
import type { Collections, CollectionName } from "@/types/pocketbase";

export interface UseCollectionOptions extends RecordListOptions {
	page?: number;
	perPage?: number;
	enabled?: boolean;
}

/**
 * Fetches a paginated list of records from a PocketBase collection.
 *
 * @example
 * const { data, isLoading } = useCollection("servers");
 * const { data } = useCollection("containers", { filter: "server = '...'" });
 */
export function useCollection<T extends CollectionName>(
	name: T,
	options: UseCollectionOptions = {},
) {
	const { page = 1, perPage = 50, enabled = true, ...pbOptions } = options;

	return useQuery<ListResult<Collections[T]>>({
		queryKey: ["collection", name, { page, perPage, ...pbOptions }],
		queryFn: () => pb.collection(name).getList(page, perPage, pbOptions),
		enabled,
	});
}
