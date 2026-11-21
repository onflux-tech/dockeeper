import type { ChangelogEntry } from "@/types/docs";

export const changelog: ChangelogEntry[] = [
	{
		version: "Unreleased",
		date: "2026-11-21",
		title: "Next release",
		summary: "Work in progress.",
		changes: [
			{
				type: "fix",
				area: "web",
				description:
					"Fixed the dashboard container count to include every reported container instead of only the first page.",
			},
			{
				type: "fix",
				area: "web",
				description:
					"Show the agent update button only when the latest release is newer than the version reported by the server.",
			},
		],
	},
	{
		version: "0.9.0",
		date: "2026-11-21",
		title: "Agent performance and review harness",
		summary: "Agent performance, review harness and quality fixes since v0.8.0.",
		changes: [
			{
				type: "perf",
				area: "agent",
				description:
					"Stopped measuring Docker when the manager is unreachable, with exponential backoff before retrying.",
			},
			{
				type: "perf",
				area: "agent",
				description:
					"Detected Traefik from container summaries and inspected only the Traefik container.",
			},
			{
				type: "perf",
				area: "agent",
				description:
					"Refreshed container discovery from Docker events, with a full sync every 10 minutes as a safety net.",
			},
			{
				type: "perf",
				area: "agent",
				description:
					"Collected container stats with one-shot requests and up to eight concurrent calls, keeping the CPU delta in the agent.",
			},
			{
				type: "fix",
				area: "agent",
				description:
					"Logged one warning when the command channel drops and another when it reconnects, moving repeated retries to debug.",
			},
			{
				type: "feat",
				area: "infra",
				description:
					"Added an adversarial review refuter and a shared read-only shell guard for Claude Code and OpenCode.",
			},
			{
				type: "fix",
				area: "manager",
				description:
					"Removed pruned images from the store immediately so the listing refreshes after a prune.",
			},
			{
				type: "fix",
				area: "web",
				description: "Fixed container column sorting on the server detail page.",
			},
			{
				type: "fix",
				area: "manager",
				description:
					"Reused existing server records on re-enrollment so agents no longer appear twice after rejoining.",
			},
			{
				type: "fix",
				area: "manager",
				description:
					"Raised text field limits so long compose YAML keeps saving after the PocketBase upgrade.",
			},
			{
				type: "fix",
				area: "infra",
				description:
					"Updated Go and npm dependencies to patched versions, clearing the open Dependabot alerts.",
			},
			{
				type: "fix",
				area: "manager",
				description:
					"Removed images, networks and volumes that disappear from the host so stale records no longer linger.",
			},
			{
				type: "fix",
				area: "agent",
				description:
					"Fixed discovery so containers and images are resent after a failed report instead of staying stale.",
			},
			{
				type: "fix",
				area: "web",
				description: "Fixed the dashboard tests and the frontend lint/format issues.",
			},
			{
				type: "refactor",
				area: "infra",
				description: "Added a CI quality gate and a pre-push hook with lint, format and tests.",
			},
			{
				type: "feat",
				area: "infra",
				description:
					"Added a single agent harness source with sync/check for Claude Code and OpenCode.",
			},
			{
				type: "docs",
				area: "infra",
				description:
					"Added architecture, gotchas, migrations, review, release and e2e documentation.",
			},
			{
				type: "docs",
				area: "infra",
				description:
					"Published a bilingual README with configuration, security and network requirements, under the MIT license.",
			},
			{
				type: "feat",
				area: "web",
				description:
					"Replaced the raster logo with the vector Hull D mark, favicon and README banner.",
			},
		],
	},
	{
		version: "0.8.0",
		date: "2026-04-06",
		title: "Security patch",
		summary: "Dependency vulnerabilities patched.",
		changes: [
			{
				type: "fix",
				area: "infra",
				description: "Patched 11 dependency vulnerabilities reported by Dependabot.",
			},
		],
	},
];

export const latestRelease = changelog.find((entry) => entry.version !== "Unreleased");

export function sortChangelog(entries: ChangelogEntry[]): ChangelogEntry[] {
	return [...entries].sort((a, b) => {
		if (a.version === "Unreleased") return -1;
		if (b.version === "Unreleased") return 1;
		return b.version.localeCompare(a.version, undefined, { numeric: true });
	});
}
