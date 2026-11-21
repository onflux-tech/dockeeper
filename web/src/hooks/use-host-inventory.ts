import { useCollection } from "@/hooks/use-collection";
import { useRealtime } from "@/hooks/use-realtime";
import { eqFilter } from "@/lib/pocketbase-filters";
import type { HostAppRecord, HostProcessRecord } from "@/types/pocketbase";

export function useHostApps(serverID: string | undefined) {
	useRealtime("host_apps");
	const { data, isLoading } = useCollection("host_apps", {
		filter: eqFilter("server", serverID),
		sort: "name",
		perPage: 200,
		enabled: !!serverID,
	});
	return { data: data?.items ?? ([] as HostAppRecord[]), isLoading };
}

export function useHostProcesses(serverID: string | undefined) {
	useRealtime("host_processes");
	const { data, isLoading } = useCollection("host_processes", {
		filter: eqFilter("server", serverID),
		sort: "-cpu_percent",
		perPage: 50,
		enabled: !!serverID,
	});
	return { data: data?.items ?? ([] as HostProcessRecord[]), isLoading };
}
