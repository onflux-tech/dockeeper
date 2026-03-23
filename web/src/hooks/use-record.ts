import { useQuery } from "@tanstack/react-query";
import type { RecordOptions } from "pocketbase";
import pb from "@/lib/pb";
import type { Collections, CollectionName } from "@/types/pocketbase";

export interface UseRecordOptions extends RecordOptions {
	enabled?: boolean;
}

/**
 * Fetches a single record by ID from a PocketBase collection.
 *
 * @example
 * const { data: server } = useRecord("servers", serverId);
 * const { data } = useRecord("containers", id, { expand: "server" });
 */
export function useRecord<T extends CollectionName>(
	name: T,
	id: string | undefined,
	options: UseRecordOptions = {},
) {
	const { enabled = true, ...pbOptions } = options;

	return useQuery<Collections[T]>({
		queryKey: ["collection", name, id],
		queryFn: () => pb.collection(name).getOne(id!, pbOptions),
		enabled: enabled && !!id,
	});
}
