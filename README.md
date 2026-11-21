<p align="center">
  <h1 align="center">Dockeeper</h1>
</p>

<p align="center">
  <strong>Self-hosted Docker management across servers: containers, compose stacks, images, networks, volumes, metrics and alerts from one dashboard, with agents that connect out to the manager over mTLS.</strong>
</p>

<p align="center">
  <a href="https://github.com/onflux-tech/dockeeper/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/onflux-tech/dockeeper/actions/workflows/ci.yml/badge.svg"></a>
  <a href="AGENTS.md"><img alt="Agent-friendly AGENTS.md" src="https://img.shields.io/badge/Agent--friendly-AGENTS.md-D97757"></a>
</p>

<p align="center">
  <img alt="Go" src="https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white">
  <img alt="React" src="https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=black">
  <img alt="PocketBase" src="https://img.shields.io/badge/PocketBase-SQLite-1F2937">
  <img alt="gRPC" src="https://img.shields.io/badge/gRPC-mTLS-244C5A?logo=grpc&logoColor=white">
  <img alt="Docker" src="https://img.shields.io/badge/Docker-Engine-2496ED?logo=docker&logoColor=white">
</p>

## Table of Contents

- [Overview](#overview)
- [Key Features](#key-features)
- [Architecture](#architecture)
- [Project Status](#project-status)
- [Quick Start](#quick-start)
- [Development](#development)
- [Project Structure](#project-structure)
- [Documentation](#documentation)
- [AI-Assisted Development](#ai-assisted-development)
- [Contributing](#contributing)

## Overview

Dockeeper is a single-binary, self-hosted platform for managing Docker hosts. A manager serves the HTTP API, the embedded React dashboard and the gRPC endpoint; lightweight agents run on each Docker host and keep an outbound mTLS connection to the manager. Nothing needs to listen on the managed hosts, and the manager never dials the agent.

The manager stores its data in PocketBase (SQLite), embeds the built frontend, and ships with authentication, RBAC, audit logs, metrics retention, alerts, notifications and backups.

## Key Features

- **Fleet inventory**: servers, containers, images, networks and volumes across every enrolled host
- **Container control**: start, stop, restart, pause, remove, inspect, edit resources and environment, stream logs and open a terminal
- **Compose stacks**: list, create, import from the host, up, down, pull and inspect services
- **Images**: pull, remove and prune per server
- **Metrics and health**: CPU, memory, disk and network series with hourly and daily retention, plus health checks
- **Alerts and notifications**: rules, blackout windows and notification channels
- **Traefik visibility**: discovered routes, services, entrypoints, TLS and middlewares
- **Operations**: desired-state drift detection, backups, audit log, search, tags and agent self-update
- **Access control**: local users with capabilities, API keys and short-lived JWTs

## Architecture

```text
Docker host                          Manager
┌──────────────────────┐             ┌─────────────────────────────┐
│ dockeeper-agent      │  outbound   │ dockeeper-manager           │
│  collectors          │ ──────────► │  HTTP 8090 + embedded UI    │
│  executor            │  gRPC mTLS  │  gRPC 50051                 │
│  terminal tunnel     │ ◄────────── │  PocketBase (SQLite)        │
└──────────────────────┘             └─────────────────────────────┘
```

- The agent opens the connection. Three gRPC streams carry commands, continuous output and the interactive terminal.
- mTLS identifies the agent by the certificate CN, which is the server record ID.
- The manager generates its own CA and the agent certificates during enrollment.
- The dashboard is compiled into the manager binary with `//go:embed`.

## Project Status

Pre-1.0 and under active development. The latest release is tagged, the next changes are tracked under `Unreleased` in `web/src/data/changelog.ts`, and breaking changes can still happen before 1.0.

The core flows are usable: enrollment, container and compose operations, image management, metrics, alerts, backups, audit, users and RBAC. Treat it as a serious homelab and small-fleet tool, not yet as a drop-in replacement for a managed control plane.

## Quick Start

### Docker

```bash
docker run -d --name dockeeper-manager \
  -p 8090:8090 \
  -p 50051:50051 \
  -v dockeeper-data:/home/dockeeper/pb_data \
  -e DK_GRPC_HOSTS=dockeeper.example.com \
  ghcr.io/onflux-tech/dockeeper-manager:latest
```

Open `http://localhost:8090`, create the admin account, then add a server in the dashboard. The manager generates the install command for the host; run it there as root.

`DK_GRPC_HOSTS` is optional and adds extra DNS names to the manager certificate, which is required when agents reach the manager through a hostname other than `localhost`. Port `50051` must be reachable by the agents; port `8090` is the dashboard and API.

The manager data lives in `/home/dockeeper/pb_data` and contains the SQLite database, the generated CA and the encryption key. Back up the volume, not only the database file.

### From source

Requirements: Go 1.25+, Node.js 22+, npm, Docker and `air`.

```bash
git clone https://github.com/onflux-tech/dockeeper.git
cd dockeeper

cd web && npm ci && cd ..
make hooks
make build-web
make dev-all
```

`make dev-all` starts the manager on `8090`/`50051`, the agent on `localhost:50051` and the Vite dev server on `5173`. The manager uses `dk_data`; the agent uses `dk-agent-data`. Use `http://localhost:5173` for hot reload and `http://localhost:8090` for the embedded build.

## Development

```bash
make dev-all        # manager, agent and web
make verify-fast    # format, harness, changelog, build-web, Go tests, web lint
make verify         # full gate: verify-fast plus Go lint, web format and web tests
make test           # Go tests
make test-web       # Vitest
make lint           # golangci-lint
make proto-gen      # regenerate protobuf code
```

Additional gates:

```bash
make harness-check  # harness ports, docs index, migrations and secrets
make harness-test   # generator and changelog checks
make changelog-check
```

Do not run `make verify` while `make dev-all` is running: `air` keeps `web/dist` busy on Windows and `build-web` fails with `EPERM`. Stop the dev stack first.

## Project Structure

```text
dockeeper/
├── cmd/
│   ├── manager/          # manager entrypoint
│   └── agent/            # agent entrypoint
├── internal/
│   ├── manager/          # HTTP routes, gRPC, services, store, events, jobs, middleware
│   ├── agent/            # enrollment, collectors, executor, tunnel, gRPC client
│   └── shared/           # models, constants, crypto, logging, security
├── migrations/           # PocketBase migrations
├── proto/                # agent.proto and generated code
├── web/                  # React 19 + Vite 8 + Tailwind 4 dashboard
├── harness/              # single source for Claude Code and OpenCode skills, commands and agents
├── scripts/              # harness and changelog checks
└── docs/                 # architecture, gotchas, migrations, release, review, e2e and harness
```

## Documentation

Internal documentation is written in Brazilian Portuguese. Start with [AGENTS.md](AGENTS.md) for the invariants and the area map, then read the document that matches the task:

- [Architecture](docs/architecture.md): topology, streams, auth, EventBus and collections
- [Gotchas](docs/gotchas.md): silent failures around migrations, proto, RBAC, embed, TLS and Docker socket
- [Migrations](docs/migrations.md): migration checklist
- [Review](docs/review.md): review contract and evidence
- [Release](docs/release.md): versioning, promotion, GoReleaser and GHCR
- [E2E](docs/e2e.md): local and staging runbook
- [Harness](docs/harness.md): agent harness source, generated ports and checks

## AI-Assisted Development

The repository ships a harness shared by Claude Code and OpenCode. The source lives in `harness/`; `make harness-sync` generates the ports under `.claude/` and `.opencode/`, and `make harness-check` fails on drift, broken references, docs outside the index, migration gaps or leaked secrets.

The harness includes area skills, a reviewer agent, verification and review commands, and a `pre-push` hook that runs the gates. `AGENTS.md` is the single source of instruction; `CLAUDE.md` only imports it.

## Contributing

1. Read [AGENTS.md](AGENTS.md) and the docs for the area you are changing.
2. Create a branch from `develop`: `feat/...`, `fix/...` or `chore/...`.
3. Follow TDD where a suite exists and keep the harness checks green.
4. Run `make verify-fast` before committing and `make verify` before pushing; fix the findings from `/review`.
5. Open a PR against `develop` with the template filled in and the test plan. Feature PRs are squash-merged; issues close with `Closes #` in the promotion PR to `production`.
6. Use English conventional commits, imperative mood, no co-authorship. Do not use `--no-verify` or force-push a shared branch.

## License

A license has not been published yet. Until one is added, the source is available for viewing only.
