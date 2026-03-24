import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { getAccessToken } from "@/lib/api";

interface CreateNetworkInput {
	server_id: string;
	name: string;
	driver: string;
	subnet?: string;
	gateway?: string;
	internal?: boolean;
}

interface NetworkResponse {
	network_id?: string;
	success: boolean;
	error?: string;
}

export function useCreateNetwork() {
	const queryClient = useQueryClient();

	return useMutation<NetworkResponse, Error, CreateNetworkInput>({
		mutationFn: async (input) => {
			const token = getAccessToken();
			const res = await fetch("/api/dk/networks", {
				method: "POST",
				headers: {
					"Content-Type": "application/json",
					...(token ? { Authorization: token } : {}),
				},
				body: JSON.stringify(input),
			});
			if (!res.ok) {
				const body = await res.json().catch(() => ({ error: res.statusText }));
				throw new Error(body.error || "Create failed");
			}
			return res.json();
		},
		onSuccess: () => {
			void queryClient.invalidateQueries({ queryKey: ["collection", "networks"] });
			toast.success("Network created");
		},
		onError: (err) => {
			toast.error(`Create failed: ${err.message}`);
		},
	});
}

export function useRemoveNetwork() {
	const queryClient = useQueryClient();

	return useMutation<NetworkResponse, Error, string>({
		mutationFn: async (id) => {
			const token = getAccessToken();
			const res = await fetch(`/api/dk/networks/${id}`, {
				method: "DELETE",
				headers: token ? { Authorization: token } : {},
			});
			if (!res.ok) {
				const body = await res.json().catch(() => ({ error: res.statusText }));
				throw new Error(body.error || "Remove failed");
			}
			return res.json();
		},
		onSuccess: () => {
			void queryClient.invalidateQueries({ queryKey: ["collection", "networks"] });
			toast.success("Network removed");
		},
		onError: (err) => {
			toast.error(`Remove failed: ${err.message}`);
		},
	});
}

export function useConnectContainer() {
	const queryClient = useQueryClient();

	return useMutation<NetworkResponse, Error, { networkId: string; containerId: string }>({
		mutationFn: async ({ networkId, containerId }) => {
			const token = getAccessToken();
			const res = await fetch(`/api/dk/networks/${networkId}/connect`, {
				method: "POST",
				headers: {
					"Content-Type": "application/json",
					...(token ? { Authorization: token } : {}),
				},
				body: JSON.stringify({ container_id: containerId }),
			});
			if (!res.ok) {
				const body = await res.json().catch(() => ({ error: res.statusText }));
				throw new Error(body.error || "Connect failed");
			}
			return res.json();
		},
		onSuccess: () => {
			void queryClient.invalidateQueries({ queryKey: ["collection", "networks"] });
			void queryClient.invalidateQueries({ queryKey: ["collection", "containers"] });
			toast.success("Container connected to network");
		},
		onError: (err) => {
			toast.error(`Connect failed: ${err.message}`);
		},
	});
}

export function useDisconnectContainer() {
	const queryClient = useQueryClient();

	return useMutation<
		NetworkResponse,
		Error,
		{ networkId: string; containerId: string; force?: boolean }
	>({
		mutationFn: async ({ networkId, containerId, force }) => {
			const token = getAccessToken();
			const res = await fetch(`/api/dk/networks/${networkId}/disconnect`, {
				method: "POST",
				headers: {
					"Content-Type": "application/json",
					...(token ? { Authorization: token } : {}),
				},
				body: JSON.stringify({ container_id: containerId, force: force ?? false }),
			});
			if (!res.ok) {
				const body = await res.json().catch(() => ({ error: res.statusText }));
				throw new Error(body.error || "Disconnect failed");
			}
			return res.json();
		},
		onSuccess: () => {
			void queryClient.invalidateQueries({ queryKey: ["collection", "networks"] });
			void queryClient.invalidateQueries({ queryKey: ["collection", "containers"] });
			toast.success("Container disconnected from network");
		},
		onError: (err) => {
			toast.error(`Disconnect failed: ${err.message}`);
		},
	});
}
