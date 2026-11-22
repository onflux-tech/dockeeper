import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { BackupPolicyRecord, BackupRecordRecord } from "@/types/pocketbase";

const state = vi.hoisted(() => ({
	collections: {} as Record<string, { items: unknown[]; totalItems: number }>,
}));

vi.mock("@/hooks/use-collection", () => ({
	useCollection: (name: string) => ({
		data: state.collections[name] ?? { items: [], totalItems: 0 },
		isLoading: false,
	}),
}));

vi.mock("@/hooks/use-permissions", () => ({
	useHasCapability: () => true,
}));

vi.mock("@/hooks/use-backups", () => ({
	useTriggerBackup: () => ({ mutate: vi.fn(), isPending: false }),
}));

vi.mock("@/components/backups/policy-dialog", () => ({
	PolicyDialog: () => null,
}));

import { HostBackupsCard } from "../host-backups-card";

describe("HostBackupsCard", () => {
	it("renders the not set state and the add recipe action", () => {
		state.collections = {};
		render(<HostBackupsCard serverID="srv" />);

		expect(screen.getByText("Backups")).toBeInTheDocument();
		expect(screen.getByText("not set")).toBeInTheDocument();
		expect(screen.getByText("No host app backup yet.")).toBeInTheDocument();
		expect(screen.getByRole("button", { name: "Add recipe" })).toBeInTheDocument();
		expect(screen.queryByRole("button", { name: "Back up now" })).not.toBeInTheDocument();
	});

	it("shows the recipe count, the last run and the manual backup action", () => {
		state.collections = {
			backup_policies: {
				items: [{ id: "p1", target_type: "host_app" } as BackupPolicyRecord],
				totalItems: 1,
			},
			backup_records: {
				items: [
					{
						id: "r1",
						target_label: "api",
						status: "failed",
						started_at: new Date().toISOString(),
					} as BackupRecordRecord,
				],
				totalItems: 1,
			},
		};
		render(<HostBackupsCard serverID="srv" />);

		expect(screen.getByText("1 recipe")).toBeInTheDocument();
		expect(screen.getByText("api")).toBeInTheDocument();
		expect(screen.getByText("failed")).toBeInTheDocument();
		expect(screen.getByRole("button", { name: "Back up now" })).toBeInTheDocument();
	});
});
