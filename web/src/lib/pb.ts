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

export default pb;
