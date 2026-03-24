import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { getAccessToken } from "@/lib/api";

// ── Stack CRUD ──────────────────────────────────────

interface CreateStackInput {
	server_id: string;
	name: string;
	yaml_content: string;
}

interface StackResponse {
	id: string;
	server: string;
	name: string;
	status: string;
}

export function useCreateStack() {
	const queryClient = useQueryClient();

	return useMutation<StackResponse, Error, CreateStackInput>({
		mutationFn: async (input) => {
			const token = getAccessToken();
			const res = await fetch("/api/dk/compose", {
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
			void queryClient.invalidateQueries({ queryKey: ["collection", "compose_stacks"] });
			toast.success("Stack created");
		},
		onError: (err) => {
			toast.error(`Create failed: ${err.message}`);
		},
	});
}

export function useUpdateStack() {
	const queryClient = useQueryClient();

	return useMutation<StackResponse, Error, { id: string; yaml_content: string }>({
		mutationFn: async ({ id, yaml_content }) => {
			const token = getAccessToken();
			const res = await fetch(`/api/dk/compose/${id}`, {
				method: "PUT",
				headers: {
					"Content-Type": "application/json",
					...(token ? { Authorization: token } : {}),
				},
				body: JSON.stringify({ yaml_content }),
			});
			if (!res.ok) {
				const body = await res.json().catch(() => ({ error: res.statusText }));
				throw new Error(body.error || "Update failed");
			}
			return res.json();
		},
		onSuccess: () => {
			void queryClient.invalidateQueries({ queryKey: ["collection", "compose_stacks"] });
			toast.success("Stack updated");
		},
		onError: (err) => {
			toast.error(`Update failed: ${err.message}`);
		},
	});
}

export function useDeleteStack() {
	const queryClient = useQueryClient();

	return useMutation<unknown, Error, string>({
		mutationFn: async (id) => {
			const token = getAccessToken();
			const res = await fetch(`/api/dk/compose/${id}`, {
				method: "DELETE",
				headers: token ? { Authorization: token } : {},
			});
			if (!res.ok) {
				const body = await res.json().catch(() => ({ error: res.statusText }));
				throw new Error(body.error || "Delete failed");
			}
			return res.json();
		},
		onSuccess: () => {
			void queryClient.invalidateQueries({ queryKey: ["collection", "compose_stacks"] });
			toast.success("Stack deleted");
		},
		onError: (err) => {
			toast.error(`Delete failed: ${err.message}`);
		},
	});
}

// ── Compose SSE Operations ──────────────────────────

export interface ComposeOutputEvent {
	line?: string;
	stream?: string;
	done?: boolean;
	exit_code?: number;
	error?: string;
}

export async function streamComposeOp(
	stackId: string,
	operation: "up" | "down" | "pull",
	onOutput: (ev: ComposeOutputEvent) => void,
	signal?: AbortSignal,
): Promise<void> {
	const token = getAccessToken();
	const res = await fetch(`/api/dk/compose/${stackId}/${operation}`, {
		headers: token ? { Authorization: token } : {},
		signal,
	});

	if (!res.ok) {
		const body = await res.json().catch(() => ({ error: res.statusText }));
		throw new Error(body.error || `Compose ${operation} failed`);
	}

	const reader = res.body?.getReader();
	if (!reader) throw new Error("No response body");

	const decoder = new TextDecoder();
	let buffer = "";

	while (true) {
		const { done, value } = await reader.read();
		if (done) break;

		buffer += decoder.decode(value, { stream: true });
		const lines = buffer.split("\n");
		buffer = lines.pop() ?? "";

		let eventType = "";
		for (const line of lines) {
			if (line.startsWith("event: ")) {
				eventType = line.slice(7);
			} else if (line.startsWith("data: ")) {
				const data = line.slice(6);
				if (eventType === "output" || eventType === "error") {
					try {
						onOutput(JSON.parse(data) as ComposeOutputEvent);
					} catch {
						// ignore
					}
				} else if (eventType === "done") {
					onOutput({ done: true });
				}
				eventType = "";
			}
		}
	}
}
