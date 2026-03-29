import type { ChangelogEntry } from "@/types/docs";

export const changelog: ChangelogEntry[] = [
  {
    version: "0.1.0",
    date: "2026-03-29",
    changes: [
      {
        type: "feat",
        area: "manager",
        description:
          "Initial release with PocketBase backend, gRPC agent communication, and 22 collections.",
      },
      {
        type: "feat",
        area: "agent",
        description:
          "Docker SDK integration with resource discovery, system metrics, and container management.",
      },
      {
        type: "feat",
        area: "web",
        description:
          "React dashboard with server overview, container management, and xterm.js terminal.",
      },
      {
        type: "feat",
        area: "api",
        description:
          "REST API with JWT + API Key authentication, RBAC capabilities, and rate limiting.",
      },
      {
        type: "fix",
        area: "manager",
        description:
          "Corrected timezone handling in backup scheduler cron evaluation.",
      },
      {
        type: "perf",
        area: "agent",
        description:
          "Reduced metrics streaming overhead with batched gRPC client-streaming.",
      },
      {
        type: "breaking",
        area: "api",
        description:
          "Replaced legacy /api/dk/updates/check with /api/dk/system/update/check.",
      },
      {
        type: "refactor",
        area: "compose",
        description:
          "Unified compose stack detection to link discovered containers automatically.",
      },
      {
        type: "docs",
        area: "infra",
        description:
          "Added README, CONTRIBUTING guide, and GoReleaser multi-arch configuration.",
      },
    ],
  },
];
