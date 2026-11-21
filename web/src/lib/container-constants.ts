export type SortKey = "name" | "status" | "image" | "created" | "uptime" | "cpu" | "ram";
export type SortDir = "asc" | "desc";

export const CONTAINER_SORT_FIELDS: Record<SortKey, string> = {
	name: "name",
	status: "status",
	image: "image",
	created: "container_created_at",
	uptime: "started_at",
	cpu: "current_cpu",
	ram: "current_ram",
};

export const statusOptions = [
	"all",
	"running",
	"exited",
	"paused",
	"created",
	"restarting",
	"dead",
];
