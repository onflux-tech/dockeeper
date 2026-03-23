import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { getAccessToken } from "@/lib/api";

type ContainerAction =
	| "start"
	| "stop"
	| "restart"
	| "remove"
	| "pause"
	| "unpause";

interface ActionResponse {
	success: boolean;
	error?: string;
	output?: string;
}

async function executeAction(
	containerId: string,
	action: ContainerAction,
	params?: Record<string, string>,
): Promise<ActionResponse> {
	const token = getAccessToken();
	const res = await fetch(`/api/dk/containers/${containerId}/${action}`, {
		method: "POST",
		headers: {
			"Content-Type": "application/json",
			...(token ? { Authorization: token } : {}),
		},
		body: JSON.stringify({ params: params ?? {} }),
	});

	if (!res.ok) {
		const body = await res.json().catch(() => ({ error: res.statusText }));
		throw new Error(body.error || `${action} failed`);
	}

	return res.json();
}

function useContainerAction(action: ContainerAction) {
	const queryClient = useQueryClient();

	return useMutation<ActionResponse, Error, string>({
		mutationFn: (containerId) => executeAction(containerId, action),
		onSuccess: () => {
			void queryClient.invalidateQueries({
				queryKey: ["collection", "containers"],
			});
			toast.success(`Container ${action} succeeded`);
		},
		onError: (err) => {
			toast.error(`Container ${action} failed: ${err.message}`);
		},
	});
}

/** Hook returning mutation objects for each container action. */
export function useContainerActions() {
	return {
		start: useContainerAction("start"),
		stop: useContainerAction("stop"),
		restart: useContainerAction("restart"),
		remove: useContainerAction("remove"),
		pause: useContainerAction("pause"),
		unpause: useContainerAction("unpause"),
	};
}

/** Execute a bulk action on multiple containers sequentially. */
export async function executeBulkAction(
	ids: string[],
	action: ContainerAction,
): Promise<{ succeeded: number; failed: number }> {
	let succeeded = 0;
	let failed = 0;

	for (const id of ids) {
		try {
			await executeAction(id, action);
			succeeded++;
		} catch {
			failed++;
		}
	}

	return { succeeded, failed };
}
