import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { getAccessToken } from "@/lib/api";

interface DesiredStateInput {
	id: string;
	desiredState: string;
}

interface DesiredStateResponse {
	id: string;
	desired_state: string;
}

export function useDesiredState() {
	const queryClient = useQueryClient();

	return useMutation<DesiredStateResponse, Error, DesiredStateInput>({
		mutationFn: async ({ id, desiredState }) => {
			const token = getAccessToken();
			const res = await fetch(`/api/dk/containers/${id}/desired-state`, {
				method: "PUT",
				headers: {
					"Content-Type": "application/json",
					...(token ? { Authorization: token } : {}),
				},
				body: JSON.stringify({ desired_state: desiredState }),
			});
			if (!res.ok) {
				const body = await res.json().catch(() => ({ error: res.statusText }));
				throw new Error(body.error || "Failed to set desired state");
			}
			return res.json();
		},
		onSuccess: (_data, { desiredState }) => {
			void queryClient.invalidateQueries({ queryKey: ["collection", "containers"] });
			const label =
				desiredState === "none"
					? "No enforcement"
					: `Should ${desiredState === "running" ? "run" : "stop"}`;
			toast.success(`Desired state: ${label}`);
		},
		onError: (err) => {
			toast.error(`Failed: ${err.message}`);
		},
	});
}
