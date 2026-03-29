import type { ChangelogEntry } from "@/types/docs";

export const changelog: ChangelogEntry[] = [
  {
    version: "1.0.0",
    date: "2026-04-01",
    title: "Dockeeper 1.0 — Open Source Release",
    summary:
      "The first public release of Dockeeper: a self-hosted, agent-based platform for managing and monitoring Docker containers across multiple servers. " +
      "Includes a Go backend powered by PocketBase, gRPC agents with mTLS encryption, and a modern React dashboard with real-time updates.",
    changes: [
      // ── Manager ──────────────────────────────────────────────
      {
        type: "feat",
        area: "manager",
        description:
          "PocketBase backend with 22 collections, ECDSA P-256 certificate authority, and mTLS agent enrollment.",
      },
      {
        type: "feat",
        area: "manager",
        description:
          "Server provisioning wizard with per-server enrollment tokens and one-command agent install script.",
      },
      {
        type: "feat",
        area: "manager",
        description:
          "Metrics receiver and storage with upsert pattern, hourly/daily aggregation jobs, and configurable retention (raw 48h, hourly 30d, daily 365d).",
      },
      {
        type: "feat",
        area: "manager",
        description:
          "Backup system with tar archives, S3 upload/download, scheduler policies with cron expressions, restore support, and retention cleanup.",
      },
      {
        type: "feat",
        area: "manager",
        description:
          "Alert engine with 30-second evaluation cycle, state machine (OK, FIRING, RESOLVED), and configurable thresholds per metric.",
      },
      {
        type: "feat",
        area: "manager",
        description:
          "Notification dispatch to WhatsApp, Telegram, and Webhook channels with 3x retry and delivery logging.",
      },
      {
        type: "feat",
        area: "manager",
        description:
          "Audit logging via PocketBase hooks on critical collections with paginated/filtered query endpoint and 90-day cleanup cron.",
      },
      {
        type: "feat",
        area: "manager",
        description:
          "Update orchestrator with saga pattern — check, download, distribute, verify, commit or automatic rollback.",
      },
      {
        type: "feat",
        area: "manager",
        description:
          "Registry credentials CRUD with AES-256-GCM encryption at rest and per-agent scoping for private registries.",
      },
      {
        type: "feat",
        area: "manager",
        description:
          "In-process EventBus with topic subscriptions, events collection storage, and SSE delivery endpoint.",
      },
      {
        type: "feat",
        area: "manager",
        description:
          "Smart cascade delete with dependency graph analysis — resolves cross-collection references before resource removal.",
      },
      {
        type: "feat",
        area: "manager",
        description:
          "WebSocket-to-gRPC terminal proxy with session tracking, audit trails, and automatic cleanup on disconnect.",
      },
      {
        type: "feat",
        area: "manager",
        description:
          "gRPC connection pool with health checking reduces reconnection overhead for multi-agent communication.",
      },
      // ── Agent ────────────────────────────────────────────────
      {
        type: "feat",
        area: "agent",
        description:
          "Docker SDK integration with automatic orchestrator detection, resource discovery (containers, images, networks, volumes), and 30-second diff-based detection cycle.",
      },
      {
        type: "feat",
        area: "agent",
        description:
          "System and container metrics collector using gopsutil and Docker Stats API with 10-second gRPC client-streaming cycle.",
      },
      {
        type: "feat",
        area: "agent",
        description:
          "Container lifecycle commands — start, stop, restart, remove, rename, pause, unpause — with gRPC handler and error propagation.",
      },
      {
        type: "feat",
        area: "agent",
        description:
          "Container creation with automatic image pull, resource limits (CPU, memory, storage), environment variables, port mappings, and volume mounts.",
      },
      {
        type: "feat",
        area: "agent",
        description:
          "Bidirectional gRPC exec streaming for interactive terminal sessions — Docker exec attach with TTY resize support.",
      },
      {
        type: "feat",
        area: "agent",
        description:
          "Image operations — pull with progress streaming, remove, and bulk prune of unused images.",
      },
      {
        type: "feat",
        area: "agent",
        description:
          "Network CRUD — create, remove, connect containers, disconnect containers with driver and subnet configuration.",
      },
      {
        type: "feat",
        area: "agent",
        description:
          "Volume CRUD — create with driver options, remove, and detailed inspect with mount point information.",
      },
      {
        type: "feat",
        area: "agent",
        description:
          "Traefik route collector — reads Traefik API v2 with Docker label fallback, 60-second cycle, and gRPC reporting.",
      },
      {
        type: "feat",
        area: "agent",
        description:
          "Self-update operations with binary snapshot, atomic replace, health verification, and automatic rollback on failure.",
      },
      {
        type: "feat",
        area: "agent",
        description:
          "YAML-based configuration, CLI flags (--manager-url, --token, --data-dir), certificate manager, and graceful shutdown.",
      },
      // ── Web ──────────────────────────────────────────────────
      {
        type: "feat",
        area: "web",
        description:
          "React 19 dashboard with Tailwind CSS v4 dark theme, shadcn/ui components, real-time PocketBase subscriptions, and TanStack Query.",
      },
      {
        type: "feat",
        area: "web",
        description:
          "App shell with collapsible sidebar, breadcrumb header, Cmd+K command palette, and mobile-responsive Sheet navigation.",
      },
      {
        type: "feat",
        area: "web",
        description:
          "Dashboard page with stats overview cards, server status grid, CPU/RAM sparkline charts, and real-time server-sent events.",
      },
      {
        type: "feat",
        area: "web",
        description:
          "Server detail page with tabbed layout (Overview, Containers), metrics charts (CPU, Memory, Disk, Network I/O), and configurable time range selector.",
      },
      {
        type: "feat",
        area: "web",
        description:
          "Container list with sort, search, status filter, bulk actions (start/stop/restart/remove), toast notifications, and real-time updates.",
      },
      {
        type: "feat",
        area: "web",
        description:
          "Container detail page with info cards, live metrics, resource limit sliders, environment variables editor with diff preview, and log viewer.",
      },
      {
        type: "feat",
        area: "web",
        description:
          "xterm.js terminal with WebSocket connection, search addon, resize handling, multi-session tabs, and clickable web links.",
      },
      {
        type: "feat",
        area: "web",
        description:
          "Container creation form with image auto-pull, port mapping, volume mounts, environment variables, resource limits, and restart policy.",
      },
      {
        type: "feat",
        area: "web",
        description:
          "Images page with table view, pull dialog with progress, remove confirmation, and bulk prune of unused images.",
      },
      {
        type: "feat",
        area: "web",
        description:
          "Compose stacks page with stack CRUD, integrated YAML editor, up/down/pull controls, and linked container display.",
      },
      {
        type: "feat",
        area: "web",
        description:
          "Networks and Volumes pages — full CRUD with create dialogs, inspect detail views, and container connection management.",
      },
      {
        type: "feat",
        area: "web",
        description:
          "Backup management with policy CRUD, records table, manual trigger, restore with double-confirmation, and cron schedule display.",
      },
      {
        type: "feat",
        area: "web",
        description:
          "Alerts page with tabbed layout — alert rules CRUD, notification channels with test send, and notification history log.",
      },
      {
        type: "feat",
        area: "web",
        description:
          "Traefik routes page with server filter, route table, and settings page with S3 configuration, retention policies, and profile management.",
      },
      {
        type: "feat",
        area: "web",
        description:
          "Users management page with Users, Groups, and API Keys tabs — full RBAC administration with capability assignment.",
      },
      {
        type: "feat",
        area: "web",
        description:
          "Registry credentials management and agent update UI with fleet-wide progress tracking.",
      },
      // ── API ──────────────────────────────────────────────────
      {
        type: "feat",
        area: "api",
        description:
          "113 REST API endpoints across 18 groups — authentication, servers, containers, images, compose, networks, volumes, backups, alerts, notifications, health checks, users, tags, batch operations, monitoring, and settings.",
      },
      {
        type: "feat",
        area: "api",
        description:
          "Three authentication methods — JWT tokens (15min access + 7d refresh), API keys with dk_ prefix and bcrypt hashing, and PocketBase SDK direct auth.",
      },
      {
        type: "feat",
        area: "api",
        description:
          "Capability-based RBAC with 26 capabilities across 13 domains — RequireCapability middleware on every protected endpoint.",
      },
      {
        type: "feat",
        area: "api",
        description:
          "Category rate limiting — auth 5/min, read 120/min, write 30/min, destructive 10/min — with per-IP tracking.",
      },
      {
        type: "feat",
        area: "api",
        description:
          "SSE streaming endpoints for container logs, image pull progress, and compose operations with real-time output.",
      },
      {
        type: "feat",
        area: "api",
        description:
          "Public API reference page at /docs with scroll-spy navigation, collapsible sidebar, code examples, and print-optimized CSS.",
      },
      // ── Compose ──────────────────────────────────────────────
      {
        type: "feat",
        area: "compose",
        description:
          "Compose stack operations — up, down, pull — with YAML validation, CLI execution, and automatic container-stack linking.",
      },
      // ── Infrastructure ───────────────────────────────────────
      {
        type: "feat",
        area: "infra",
        description:
          "Multi-stage Dockerfiles for manager and agent with go:embed frontend bundling and SPA fallback routing.",
      },
      {
        type: "feat",
        area: "infra",
        description:
          "GoReleaser configuration for multi-arch builds (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64).",
      },
      {
        type: "feat",
        area: "infra",
        description:
          "docker-compose.yml for single-command deployment with manager, volume persistence, and configurable environment.",
      },
      {
        type: "feat",
        area: "infra",
        description:
          "Proto/gRPC scaffolding with buf configuration, 6 gRPC services, and the agentpb generated package.",
      },
      {
        type: "feat",
        area: "infra",
        description:
          "Makefile with build-manager, build-agent, dev, dev-web, build-web, lint-web, format-web, and test targets.",
      },
      {
        type: "feat",
        area: "infra",
        description:
          "SSRF protection for webhook URLs blocking private, loopback, and link-local addresses. Input sanitization for container, volume, network names, and Docker image references.",
      },
      {
        type: "feat",
        area: "infra",
        description:
          "E2E smoke tests with Go testify and frontend tests with Vitest + React Testing Library.",
      },
    ],
  },
];
