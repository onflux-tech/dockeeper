import type { QueryClient } from "@tanstack/react-query";
import type { CollectionName } from "@/types/pocketbase";

const collectionKeyMap: Partial<Record<CollectionName, readonly unknown[][]>> = {
	servers: [["servers"], ["install-script"]],
	alert_rules: [["alert-rules"]],
	notification_channels: [["notification-channels"]],
	notification_logs: [["notification-logs"]],
	backup_policies: [["backup-policies"]],
	backup_records: [["backup-records"]],
	settings: [["settings"], ["enrollment-token"]],
	traefik_routes: [["traefik-routes"]],
	server_metrics_raw: [["server-metrics"]],
	server_metrics_hourly: [["server-metrics"]],
	server_metrics_daily: [["server-metrics"]],
	container_metrics_raw: [["container-metrics"]],
	container_metrics_hourly: [["container-metrics"]],
	container_metrics_daily: [["container-metrics"]],
};

export function invalidateCollectionQueries(queryClient: QueryClient, name: CollectionName) {
	void queryClient.invalidateQueries({
		queryKey: ["collection", name],
	});

	for (const queryKey of collectionKeyMap[name] ?? []) {
		void queryClient.invalidateQueries({ queryKey: [...queryKey] });
	}
}
