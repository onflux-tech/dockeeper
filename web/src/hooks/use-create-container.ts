import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { getAccessToken } from "@/lib/api";
import type { ContainerWizardData } from "@/components/container/wizard/schema";

interface CreateContainerResponse {
	success: boolean;
	error?: string;
	output?: string;
}

function transformFormData(data: ContainerWizardData) {
	const env: Record<string, string> = {};
	for (const { key, value } of data.env) {
		env[key] = value;
	}

	const labels: Record<string, string> = {};
	for (const { key, value } of data.labels) {
		labels[key] = value;
	}

	return {
		server_id: data.server_id,
		name: data.name,
		image: data.image,
		auto_start: data.auto_start,
		ports: data.ports,
		volumes: data.volumes,
		env,
		labels,
		network: data.network || undefined,
		restart_policy: data.restart_policy || undefined,
		command: data.command ? data.command.split(/\s+/).filter(Boolean) : undefined,
		entrypoint: data.entrypoint ? data.entrypoint.split(/\s+/).filter(Boolean) : undefined,
		cpu_limit: data.cpu_limit || undefined,
		mem_limit: data.mem_limit ? data.mem_limit * 1024 * 1024 : undefined, // MB → bytes
		pids_limit: data.pids_limit || undefined,
	};
}

async function createContainer(data: ContainerWizardData): Promise<CreateContainerResponse> {
	const token = getAccessToken();
	const body = transformFormData(data);

	const res = await fetch("/api/dk/containers/create", {
		method: "POST",
		headers: {
			"Content-Type": "application/json",
			...(token ? { Authorization: token } : {}),
		},
		body: JSON.stringify(body),
	});

	if (!res.ok) {
		const err = await res.json().catch(() => ({ error: res.statusText }));
		throw new Error(err.error || "Failed to create container");
	}

	return res.json();
}

export function useCreateContainer() {
	const queryClient = useQueryClient();

	return useMutation<CreateContainerResponse, Error, ContainerWizardData>({
		mutationFn: createContainer,
		onSuccess: () => {
			void queryClient.invalidateQueries({ queryKey: ["collection", "containers"] });
			toast.success("Container created successfully");
		},
		onError: (err) => {
			toast.error(err.message);
		},
	});
}
