import PocketBase from "pocketbase";
import type { RecordService } from "pocketbase";
import type { Collections } from "@/types/pocketbase";

/**
 * TypedPocketBase overrides `collection()` so callers get a fully-typed
 * `RecordService<T>` when a known collection name is passed.
 */
export type TypedPocketBase = PocketBase & {
	collection<T extends keyof Collections>(
		name: T,
	): RecordService<Collections[T]>;
};

const pb = new PocketBase("/") as TypedPocketBase;

/** Sync the superuser token into the PB SDK so collection queries authenticate. */
export function syncPbAuth(token: string | null) {
	if (token) {
		pb.authStore.save(token, null);
	} else {
		pb.authStore.clear();
	}
}

export default pb;
