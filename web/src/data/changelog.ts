import type { ChangelogEntry } from "@/types/docs";

export const changelog: ChangelogEntry[] = [
	{
		version: "Unreleased",
		date: "2026-11-20",
		title: "Next release",
		summary:
			"Quality, documentation and agent harness work since v0.8.0. No user-facing feature yet.",
		changes: [
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
