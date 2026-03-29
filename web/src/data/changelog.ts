import type { ChangelogEntry } from "@/types/docs";

export const changelog: ChangelogEntry[] = [
  {
    version: "0.3.0",
    date: "2026-03-29",
    title: "API Documentation, Reverse Tunnel Preparation and Security Hardening",
    summary:
      "Public API reference page with 113 endpoints documented, scroll-spy navigation and print support. " +
      "Proto definitions for reverse tunnel architecture (zero inbound ports). " +
      "Smart cascade delete with dependency graph analysis across all resource types.",
    changes: [
      {
        type: "feat",
        area: "web",
        description:
          "Public API reference page at /docs with 18 groups, 113 endpoints, scroll-spy sidebar, code examples, and print CSS.",
      },
      {
        type: "feat",
        area: "api",
        description:
          "Getting Started section with JWT, API Key, and PocketBase SDK authentication guides.",
      },
      {
        type: "feat",
        area: "api",
        description:
          "RBAC capabilities reference table covering 17 domains and 29 capabilities.",
      },
      {
        type: "feat",
        area: "manager",
        description:
          "Smart cascade delete with dependency graph — analyzes cross-collection references before removal.",
      },
      {
        type: "feat",
        area: "infra",
        description:
          "Proto definitions for AgentCommandChannel, AgentDataStream, and AgentExecStream reverse tunnel services.",
      },
      {
        type: "fix",
        area: "web",
        description:
          "Resolved IntersectionObserver mismatch with ScrollArea causing scroll-spy jumps and freezing.",
      },
      {
        type: "perf",
        area: "web",
        description:
          "Instant scroll navigation and sidebar active state tracking without debounce delays.",
      },
    ],
  },
  {
    version: "0.2.0",
    date: "2026-03-22",
    title: "Auto-Updates, Registry Credentials and Multi-User Events",
    summary:
      "Agent self-update with snapshot/rollback saga pattern. " +
      "Container registry credential management with AES-256-GCM encryption. " +
      "In-process EventBus with SSE delivery and notification dispatch.",
    changes: [
      {
        type: "feat",
        area: "agent",
        description:
          "Agent self-update operations — binary snapshot, replace, and automatic rollback on failure.",
      },
      {
        type: "feat",
        area: "manager",
        description:
          "Update orchestrator with saga pattern — check, download, distribute, verify, commit or rollback.",
      },
      {
        type: "feat",
        area: "manager",
        description:
          "Registry credentials CRUD with AES-256-GCM encryption at rest and per-agent scoping.",
      },
      {
        type: "feat",
        area: "manager",
        description:
          "In-process EventBus with topic subscriptions, events collection, and SSE delivery endpoint.",
      },
      {
        type: "feat",
        area: "api",
        description:
          "Update check and apply endpoints for coordinated fleet-wide agent updates.",
      },
      {
        type: "feat",
        area: "web",
        description:
          "Registry credentials management page and agent update UI with progress tracking.",
      },
      {
        type: "fix",
        area: "compose",
        description:
          "Compose-container linking now correctly associates discovered containers with their stack.",
      },
      {
        type: "perf",
        area: "manager",
        description:
          "gRPC connection pool reduces reconnection overhead for multi-agent communication.",
      },
      {
        type: "refactor",
        area: "manager",
        description:
          "Registry adapter interface decouples Docker Hub, GHCR, and private registry implementations.",
      },
    ],
  },
  {
    version: "0.1.2",
    date: "2026-03-15",
    title: "Security Foundation and RBAC System",
    summary:
      "Capability-based RBAC with 26 capabilities across 13 domains. " +
      "API key authentication with bcrypt hashing. " +
      "Category rate limiting and SSRF protection for webhook URLs.",
    changes: [
      {
        type: "feat",
        area: "manager",
        description:
          "RBAC system with groups, membership management, and 26 capability constants across 13 domains.",
      },
      {
        type: "feat",
        area: "api",
        description:
          "API key authentication — bcrypt-hashed keys with dk_ prefix, create/validate/list/delete endpoints.",
      },
      {
        type: "feat",
        area: "manager",
        description:
          "Category rate limiting — auth 5/min, read 120/min, write 30/min, destructive 10/min.",
      },
      {
        type: "feat",
        area: "manager",
        description:
          "SSRF protection validates webhook URLs blocking private, loopback, and link-local addresses.",
      },
      {
        type: "feat",
        area: "web",
        description:
          "Users management page with Users, Groups, and API Keys tabs.",
      },
      {
        type: "feat",
        area: "manager",
        description:
          "Input sanitization for container names, volume names, network names, and Docker image references.",
      },
      {
        type: "docs",
        area: "api",
        description:
          "RequireCapability middleware documentation with capability-to-endpoint mapping.",
      },
    ],
  },
  {
    version: "0.1.1",
    date: "2026-03-08",
    title: "Monitoring, Alerts and Traefik Integration",
    summary:
      "Alert engine with state machine (OK, FIRING, RESOLVED) and multi-channel notifications. " +
      "Traefik route discovery via API and Docker labels. " +
      "Audit logging with PocketBase hooks and 90-day retention.",
    changes: [
      {
        type: "feat",
        area: "manager",
        description:
          "Alert engine with 30s evaluation cycle, state machine transitions, and configurable thresholds.",
      },
      {
        type: "feat",
        area: "manager",
        description:
          "Notification dispatch to WhatsApp, Telegram, and Webhook channels with 3x retry.",
      },
      {
        type: "feat",
        area: "agent",
        description:
          "Traefik route collector — reads Traefik API v2 with Docker label fallback on 60s cycle.",
      },
      {
        type: "feat",
        area: "manager",
        description:
          "Audit logging via PocketBase hooks on 5 collections with paginated, filtered GET endpoint.",
      },
      {
        type: "feat",
        area: "web",
        description:
          "Alerts page with Rules, Channels, and History tabs — CRUD, test send, and toggle controls.",
      },
      {
        type: "feat",
        area: "web",
        description:
          "Traefik routes page with server filter and settings page (S3, retention, profile, about).",
      },
      {
        type: "fix",
        area: "manager",
        description:
          "Notification log cleanup cron now correctly respects 90-day retention boundary.",
      },
      {
        type: "perf",
        area: "agent",
        description:
          "Traefik collector caches route state to avoid redundant gRPC reports on unchanged routes.",
      },
    ],
  },
  {
    version: "0.1.0",
    date: "2026-03-01",
    title: "Initial Release",
    summary:
      "PocketBase backend with 22 collections and gRPC agent architecture. " +
      "Docker SDK integration for container lifecycle, metrics, and terminal access. " +
      "React dashboard with real-time updates and xterm.js terminal.",
    changes: [
      {
        type: "feat",
        area: "manager",
        description:
          "PocketBase backend with 22 collections, ECDSA P-256 CA, mTLS agent enrollment, and offline detection.",
      },
      {
        type: "feat",
        area: "agent",
        description:
          "Docker SDK integration — resource discovery, system/container metrics, and container lifecycle commands.",
      },
      {
        type: "feat",
        area: "web",
        description:
          "React dashboard with server overview, container management, image/compose/network/volume pages.",
      },
      {
        type: "feat",
        area: "web",
        description:
          "xterm.js terminal with WebSocket proxy, bidirectional gRPC exec streaming, and multi-session support.",
      },
      {
        type: "feat",
        area: "api",
        description:
          "REST API with JWT and API Key authentication, protected routes, and Zustand auth store.",
      },
      {
        type: "feat",
        area: "manager",
        description:
          "Backup system with tar archives, S3 upload/download, scheduler policies, and retention cleanup.",
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
          "README, CONTRIBUTING guide, multi-stage Dockerfiles, docker-compose, and GoReleaser multi-arch config.",
      },
    ],
  },
];
