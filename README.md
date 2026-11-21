<p align="center">
  <img alt="Dockeeper" src="web/public/favicon.png" width="96">
</p>

<h1 align="center">Dockeeper</h1>

<p align="center">
  <strong>Self-hosted Docker management across servers: containers, compose stacks, images, networks, volumes, metrics and alerts from one dashboard, with agents that connect out to the manager over mTLS.</strong>
</p>

<p align="center">
  <a href="https://github.com/onflux-tech/dockeeper/actions/workflows/ci.yml"><picture><source media="(prefers-color-scheme: dark)" srcset="https://www.shieldcn.dev/github/ci/onflux-tech/dockeeper.svg?workflow=ci.yml&amp;branch=production&amp;variant=secondary&amp;size=sm&amp;mode=dark"><img alt="CI" src="https://www.shieldcn.dev/github/ci/onflux-tech/dockeeper.svg?workflow=ci.yml&amp;branch=production&amp;variant=secondary&amp;size=sm&amp;mode=light"></picture></a>
  <a href="#license"><picture><source media="(prefers-color-scheme: dark)" srcset="https://www.shieldcn.dev/badge/License-unpublished-6B7280.svg?variant=secondary&amp;size=sm&amp;mode=dark"><img alt="License" src="https://www.shieldcn.dev/badge/License-unpublished-6B7280.svg?variant=secondary&amp;size=sm&amp;mode=light"></picture></a>
  <a href="https://github.com/onflux-tech/dockeeper/commits/production"><picture><source media="(prefers-color-scheme: dark)" srcset="https://www.shieldcn.dev/github/last-commit/onflux-tech/dockeeper.svg?variant=secondary&amp;size=sm&amp;mode=dark"><img alt="Last commit" src="https://www.shieldcn.dev/github/last-commit/onflux-tech/dockeeper.svg?variant=secondary&amp;size=sm&amp;mode=light"></picture></a>
  <a href="https://github.com/onflux-tech/dockeeper/stargazers"><picture><source media="(prefers-color-scheme: dark)" srcset="https://www.shieldcn.dev/github/stars/onflux-tech/dockeeper.svg?variant=secondary&amp;size=sm&amp;mode=dark"><img alt="GitHub Stars" src="https://www.shieldcn.dev/github/stars/onflux-tech/dockeeper.svg?variant=secondary&amp;size=sm&amp;mode=light"></picture></a>
</p>

<p align="center">
  <a href="https://go.dev/"><picture><source media="(prefers-color-scheme: dark)" srcset="https://www.shieldcn.dev/badge/Language-Go-00ADD8.svg?logo=go&amp;variant=branded&amp;size=sm&amp;mode=dark"><img alt="Language · Go" src="https://www.shieldcn.dev/badge/Language-Go-00ADD8.svg?logo=go&amp;variant=branded&amp;size=sm&amp;mode=light"></picture></a>
  <a href="https://www.npmjs.com/"><picture><source media="(prefers-color-scheme: dark)" srcset="https://www.shieldcn.dev/badge/Package_mgr-npm-CB3837.svg?logo=npm&amp;variant=branded&amp;size=sm&amp;mode=dark"><img alt="Package mgr · npm" src="https://www.shieldcn.dev/badge/Package_mgr-npm-CB3837.svg?logo=npm&amp;variant=branded&amp;size=sm&amp;mode=light"></picture></a>
  <a href="https://react.dev/"><picture><source media="(prefers-color-scheme: dark)" srcset="https://www.shieldcn.dev/badge/Framework-React-61DAFB.svg?logo=react&amp;variant=branded&amp;size=sm&amp;mode=dark"><img alt="Framework · React" src="https://www.shieldcn.dev/badge/Framework-React-61DAFB.svg?logo=react&amp;variant=branded&amp;size=sm&amp;mode=light"></picture></a>
  <a href="https://pocketbase.io/"><picture><source media="(prefers-color-scheme: dark)" srcset="https://www.shieldcn.dev/badge/Database-PocketBase-1F2937.svg?variant=branded&amp;size=sm&amp;mode=dark"><img alt="Database · PocketBase" src="https://www.shieldcn.dev/badge/Database-PocketBase-1F2937.svg?variant=branded&amp;size=sm&amp;mode=light"></picture></a>
  <a href="https://grpc.io/"><picture><source media="(prefers-color-scheme: dark)" srcset="https://www.shieldcn.dev/badge/Transport-gRPC-244C5A.svg?logo=grpc&amp;variant=branded&amp;size=sm&amp;mode=dark"><img alt="Transport · gRPC" src="https://www.shieldcn.dev/badge/Transport-gRPC-244C5A.svg?logo=grpc&amp;variant=branded&amp;size=sm&amp;mode=light"></picture></a>
  <a href="AGENTS.md"><picture><source media="(prefers-color-scheme: dark)" srcset="https://www.shieldcn.dev/badge/Agent--friendly-AGENTS.md-D97757.svg?variant=secondary&amp;size=sm&amp;mode=dark"><img alt="Agent-friendly AGENTS.md" src="https://www.shieldcn.dev/badge/Agent--friendly-AGENTS.md-D97757.svg?variant=secondary&amp;size=sm&amp;mode=light"></picture></a>
</p>

<p align="center">
  <strong><a href="#english">English</a></strong> | <strong><a href="#portugues">Português</a></strong>
</p>

---

<a id="english"></a>

## English

### Table of Contents

- [Overview](#overview)
- [Key Features](#key-features)
- [Project Status](#project-status)
- [Quick Start](#quick-start)
- [Development](#development)
- [Project Structure](#project-structure)
- [Documentation](#documentation)
- [AI-Assisted Development](#ai-assisted-development)
- [Contributing](#contributing)
- [License](#license)

### Overview

Dockeeper is a self-hosted platform for managing Docker hosts from a single dashboard. A manager serves the HTTP API, the embedded React dashboard and the gRPC endpoint; lightweight agents run on each Docker host and keep an outbound mTLS connection to the manager. Managed hosts do not need inbound ports, and the manager never dials an agent.

Data is stored in PocketBase (SQLite), and the manager ships with authentication, RBAC, audit logs, metrics retention, alerts, notifications and backups.

### Key Features

- **Fleet inventory** -- servers, containers, images, networks and volumes across every enrolled host
- **Container control** -- start, stop, restart, pause, remove, inspect, edit resources and environment, stream logs and open a terminal
- **Compose stacks** -- list, create, import from the host, up, down, pull and inspect services
- **Images** -- pull, remove and prune per server
- **Metrics and health** -- CPU, memory, disk and network series with hourly and daily retention, plus health checks
- **Alerts and notifications** -- rules, blackout windows and notification channels
- **Traefik visibility** -- discovered routes, services, entrypoints, TLS and middlewares
- **Operations** -- desired-state drift detection, backups, audit log, search, tags and agent self-update
- **Access control** -- local users with capabilities, API keys and short-lived JWTs

### Project Status

Pre-1.0 and under active development. The latest release is tagged, the next changes are tracked under `Unreleased` in `web/src/data/changelog.ts`, and breaking changes can still happen before 1.0.

The core flows are usable: enrollment, container and compose operations, image management, metrics, alerts, backups, audit, users and RBAC. Treat it as a serious homelab and small-fleet tool, not yet as a drop-in replacement for a managed control plane.

### Quick Start

#### Docker

```bash
docker run -d --name dockeeper-manager \
  -p 8090:8090 \
  -p 50051:50051 \
  -v dockeeper-data:/home/dockeeper/pb_data \
  -e DK_GRPC_HOSTS=dockeeper.example.com \
  ghcr.io/onflux-tech/dockeeper-manager:latest
```

Open `http://localhost:8090`, create the admin account, then add a server in the dashboard. The manager generates the install command for the host; run it there as root.

`DK_GRPC_HOSTS` is optional and adds extra DNS names to the manager certificate, required when agents reach the manager through a hostname other than `localhost`. Port `50051` must be reachable by the agents; port `8090` is the dashboard and API.

The manager data lives in `/home/dockeeper/pb_data` and contains the SQLite database, the generated CA and the encryption key. Back up the volume, not only the database file.

#### From source

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

### Development

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

### Project Structure

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

### Documentation

Internal documentation is written in Brazilian Portuguese. Start with [AGENTS.md](AGENTS.md) for the invariants and the area map, then read the document that matches the task:

- [Architecture](docs/architecture.md): topology, streams, auth, EventBus and collections
- [Gotchas](docs/gotchas.md): silent failures around migrations, proto, RBAC, embed, TLS and Docker socket
- [Migrations](docs/migrations.md): migration checklist
- [Review](docs/review.md): review contract and evidence
- [Release](docs/release.md): versioning, promotion, GoReleaser and GHCR
- [E2E](docs/e2e.md): local and staging runbook
- [Harness](docs/harness.md): agent harness source, generated ports and checks

### AI-Assisted Development

The repository ships a harness shared by Claude Code and OpenCode. The source lives in `harness/`; `make harness-sync` generates the ports under `.claude/` and `.opencode/`, and `make harness-check` fails on drift, broken references, docs outside the index, migration gaps or leaked secrets.

The harness includes area skills, a reviewer agent, verification and review commands, and a `pre-push` hook that runs the gates. `AGENTS.md` is the single source of instruction; `CLAUDE.md` only imports it.

### Contributing

1. Read [AGENTS.md](AGENTS.md) and the docs for the area you are changing.
2. Create a branch from `develop`: `feat/...`, `fix/...` or `chore/...`.
3. Follow TDD where a suite exists and keep the harness checks green.
4. Run `make verify-fast` before committing and `make verify` before pushing; fix the findings from `/review`.
5. Open a PR against `develop` with the template filled in and the test plan. Feature PRs are squash-merged; issues close with `Closes #` in the promotion PR to `production`.
6. Use English conventional commits, imperative mood, no co-authorship. Do not use `--no-verify` or force-push a shared branch.

### License

A license has not been published yet. Until one is added, the source is available for viewing only.

---

<a id="portugues"></a>

## Português

### Índice

- [Visão geral](#visão-geral)
- [Principais recursos](#principais-recursos)
- [Estado do projeto](#estado-do-projeto)
- [Início rápido](#início-rápido)
- [Desenvolvimento](#desenvolvimento)
- [Estrutura do projeto](#estrutura-do-projeto)
- [Documentação](#documentação)
- [Desenvolvimento com agentes de IA](#desenvolvimento-com-agentes-de-ia)
- [Contribuindo](#contribuindo)
- [Licença](#licença)

### Visão geral

O Dockeeper é uma plataforma self-hosted para gerenciar hosts Docker a partir de um único dashboard. O manager serve a API HTTP, o dashboard React embutido e o endpoint gRPC; agents leves rodam em cada host Docker e mantêm uma conexão de saída mTLS com o manager. Os hosts gerenciados não precisam de portas de entrada, e o manager nunca disca para o agent.

Os dados ficam no PocketBase (SQLite), e o manager traz autenticação, RBAC, logs de auditoria, retenção de métricas, alertas, notificações e backups.

### Principais recursos

- **Inventário da frota** -- servers, containers, imagens, redes e volumes de todos os hosts inscritos
- **Controle de containers** -- iniciar, parar, reiniciar, pausar, remover, inspecionar, editar recursos e variáveis de ambiente, acompanhar logs e abrir terminal
- **Stacks de compose** -- listar, criar, importar do host, subir, derrubar, puxar e inspecionar serviços
- **Imagens** -- puxar, remover e fazer prune por server
- **Métricas e saúde** -- séries de CPU, memória, disco e rede com retenção horária e diária, além de health checks
- **Alertas e notificações** -- regras, janelas de silêncio e canais de notificação
- **Visibilidade do Traefik** -- rotas, serviços, entrypoints, TLS e middlewares descobertos
- **Operação** -- detecção de drift do estado desejado, backups, audit log, busca, tags e self-update do agent
- **Controle de acesso** -- usuários locais com capabilities, API keys e JWTs de curta duração

### Estado do projeto

Pré-1.0 e em desenvolvimento ativo. A última release está tagueada, as próximas mudanças ficam em `Unreleased` em `web/src/data/changelog.ts`, e ainda pode haver breaking changes antes da 1.0.

Os fluxos centrais são utilizáveis: enrollment, operações de containers e compose, gestão de imagens, métricas, alertas, backups, auditoria, usuários e RBAC. Trate como uma ferramenta séria para homelab e frotas pequenas, ainda não como substituto direto de um control plane gerenciado.

### Início rápido

#### Docker

```bash
docker run -d --name dockeeper-manager \
  -p 8090:8090 \
  -p 50051:50051 \
  -v dockeeper-data:/home/dockeeper/pb_data \
  -e DK_GRPC_HOSTS=dockeeper.example.com \
  ghcr.io/onflux-tech/dockeeper-manager:latest
```

Abra `http://localhost:8090`, crie a conta de admin e adicione um server no dashboard. O manager gera o comando de instalação do host; rode esse comando lá como root.

`DK_GRPC_HOSTS` é opcional e adiciona nomes DNS extras ao certificado do manager, necessário quando os agents chegam por um hostname diferente de `localhost`. A porta `50051` precisa estar acessível para os agents; a `8090` é o dashboard e a API.

Os dados do manager ficam em `/home/dockeeper/pb_data` e contêm o banco SQLite, a CA gerada e a chave de criptografia. Faça backup do volume, não apenas do arquivo do banco.

#### A partir do código

Requisitos: Go 1.25+, Node.js 22+, npm, Docker e `air`.

```bash
git clone https://github.com/onflux-tech/dockeeper.git
cd dockeeper

cd web && npm ci && cd ..
make hooks
make build-web
make dev-all
```

O `make dev-all` sobe o manager em `8090`/`50051`, o agent em `localhost:50051` e o Vite em `5173`. O manager usa `dk_data`; o agent usa `dk-agent-data`. Use `http://localhost:5173` para hot reload e `http://localhost:8090` para o build embutido.

### Desenvolvimento

```bash
make dev-all        # manager, agent e web
make verify-fast    # format, harness, changelog, build-web, testes Go e lint web
make verify         # gate completo: verify-fast mais lint Go, format web e testes web
make test           # testes Go
make test-web       # Vitest
make lint           # golangci-lint
make proto-gen      # regenera o código protobuf
```

Gates adicionais:

```bash
make harness-check  # portas do harness, índice de docs, migrations e segredos
make harness-test   # testes do gerador e do changelog
make changelog-check
```

Não rode `make verify` com o `make dev-all` no ar: o `air` segura `web/dist` no Windows e o `build-web` falha com `EPERM`. Pare o dev stack antes.

### Estrutura do projeto

A árvore de pastas é a mesma da seção [Project Structure](#project-structure).

### Documentação

A documentação interna é escrita em português do Brasil. Comece pelo [AGENTS.md](AGENTS.md) para invariantes e mapa de áreas, depois leia o documento da tarefa:

- [Arquitetura](docs/architecture.md): topologia, streams, auth, EventBus e collections
- [Gotchas](docs/gotchas.md): falhas silenciosas em migrations, proto, RBAC, embed, TLS e Docker socket
- [Migrations](docs/migrations.md): checklist de migration
- [Review](docs/review.md): contrato de review e evidência
- [Release](docs/release.md): versionamento, promoção, GoReleaser e GHCR
- [E2E](docs/e2e.md): runbook local e staging
- [Harness](docs/harness.md): fonte do harness, portas geradas e checks

### Desenvolvimento com agentes de IA

O repositório traz um harness compartilhado por Claude Code e OpenCode. A fonte fica em `harness/`; `make harness-sync` gera as portas em `.claude/` e `.opencode/`, e `make harness-check` falha em drift, referência quebrada, doc fora do índice, lacuna de migration ou segredo exposto.

O harness inclui skills por área, um agent revisor, comandos de verificação e review, e um hook `pre-push` que roda os gates. O `AGENTS.md` é a fonte única de instrução; o `CLAUDE.md` apenas importa esse arquivo.

### Contribuindo

1. Leia o [AGENTS.md](AGENTS.md) e as docs da área que vai mudar.
2. Crie uma branch a partir de `develop`: `feat/...`, `fix/...` ou `chore/...`.
3. Siga TDD onde existe suíte e mantenha os checks do harness verdes.
4. Rode `make verify-fast` antes de commitar e `make verify` antes do push; corrija os achados do `/review`.
5. Abra o PR contra `develop` com o template preenchido e o plano de teste. PRs de feature entram por squash; issues fecham com `Closes #` no PR de promoção para `production`.
6. Use conventional commits em inglês, imperativo, sem coautoria. Não use `--no-verify` nem force-push em branch compartilhada.

### Licença

A licença ainda não foi publicada. Até que uma seja adicionada, o código está disponível apenas para visualização.
