import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { getAccessToken } from "@/lib/api";

interface VolumeResponse {
	name?: string;
	success: boolean;
	error?: string;
}

interface InspectResponse {
	name: string;
	driver: string;
	mountpoint: string;
	size_bytes: number;
	labels: Record<string, string>;
	created_at: string;
	success: boolean;
	error?: string;
}

interface CreateVolumeInput {
	server_id: string;
	name: string;
	driver: string;
}

export function useCreateVolume() {
	const queryClient = useQueryClient();

	return useMutation<VolumeResponse, Error, CreateVolumeInput>({
		mutationFn: async (input) => {
			const token = getAccessToken();
			const res = await fetch("/api/dk/volumes", {
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
			void queryClient.invalidateQueries({ queryKey: ["collection", "volumes"] });
			toast.success("Volume created");
		},
		onError: (err) => {
			toast.error(`Create failed: ${err.message}`);
		},
	});
}

export function useRemoveVolume() {
	const queryClient = useQueryClient();

	return useMutation<VolumeResponse, Error, { id: string; force?: boolean }>({
		mutationFn: async ({ id, force }) => {
			const token = getAccessToken();
			const res = await fetch(`/api/dk/volumes/${id}`, {
				method: "DELETE",
				headers: {
					"Content-Type": "application/json",
					...(token ? { Authorization: token } : {}),
				},
				body: JSON.stringify({ force: force ?? false }),
			});
			if (!res.ok) {
				const body = await res.json().catch(() => ({ error: res.statusText }));
				throw new Error(body.error || "Remove failed");
			}
			return res.json();
		},
		onSuccess: () => {
			void queryClient.invalidateQueries({ queryKey: ["collection", "volumes"] });
			toast.success("Volume removed");
		},
		onError: (err) => {
			toast.error(`Remove failed: ${err.message}`);
		},
	});
}

export function useInspectVolume() {
	return useMutation<InspectResponse, Error, string>({
		mutationFn: async (id) => {
			const token = getAccessToken();
			const res = await fetch(`/api/dk/volumes/${id}/inspect`, {
				headers: token ? { Authorization: token } : {},
			});
			if (!res.ok) {
				const body = await res.json().catch(() => ({ error: res.statusText }));
				throw new Error(body.error || "Inspect failed");
			}
			return res.json();
		},
	});
}
