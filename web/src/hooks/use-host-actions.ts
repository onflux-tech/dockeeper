import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { getAccessToken } from "@/lib/api";

export type HostAction = "start" | "stop" | "restart";

export interface HostActionInput {
	serverID: string;
	source: string;
	name: string;
	action: HostAction;
}

interface HostActionResponse {
	output: string;
}

async function executeHostAction(input: HostActionInput): Promise<HostActionResponse> {
	const token = getAccessToken();
	const res = await fetch(`/api/dk/servers/${input.serverID}/apps/action`, {
		method: "POST",
		headers: {
			"Content-Type": "application/json",
			...(token ? { Authorization: token } : {}),
		},
		body: JSON.stringify({
			source: input.source,
			name: input.name,
			action: input.action,
		}),
	});

	if (!res.ok) {
		const body = await res.json().catch(() => ({ error: res.statusText }));
		throw new Error(body.error || `${input.action} failed`);
	}

	return res.json();
}

export function useHostAction() {
	const queryClient = useQueryClient();

	return useMutation<HostActionResponse, Error, HostActionInput>({
		mutationFn: executeHostAction,
		onSuccess: (data, input) => {
			void queryClient.invalidateQueries({ queryKey: ["collection", "host_apps"] });
			toast.success(data.output?.trim() || `${input.name}: ${input.action} completed`);
		},
		onError: (err, input) => {
			toast.error(`${input.name}: ${input.action} failed — ${err.message}`);
		},
	});
}
