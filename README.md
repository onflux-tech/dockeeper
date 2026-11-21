<p align="center">
  <img alt="Dockeeper — every Docker host, one dashboard" src="docs/assets/banner.png" width="100%">
</p>

<h1 align="center">Dockeeper</h1>

<p align="center">
  <strong>Self-hosted Docker management across servers: containers, compose stacks, images, networks, volumes, metrics and alerts from one dashboard, with agents that connect out to the manager over mTLS.</strong>
</p>

<p align="center">
  <a href="https://github.com/onflux-tech/dockeeper/actions/workflows/ci.yml"><picture><source media="(prefers-color-scheme: dark)" srcset="https://www.shieldcn.dev/github/ci/onflux-tech/dockeeper.svg?workflow=ci.yml&amp;branch=production&amp;variant=secondary&amp;size=sm&amp;mode=dark"><img alt="CI" src="https://www.shieldcn.dev/github/ci/onflux-tech/dockeeper.svg?workflow=ci.yml&amp;branch=production&amp;variant=secondary&amp;size=sm&amp;mode=light"></picture></a>
  <a href="LICENSE"><picture><source media="(prefers-color-scheme: dark)" srcset="https://www.shieldcn.dev/github/license/onflux-tech/dockeeper.svg?variant=ghost&amp;size=sm&amp;mode=dark"><img alt="License" src="https://www.shieldcn.dev/github/license/onflux-tech/dockeeper.svg?variant=ghost&amp;size=sm&amp;mode=light"></picture></a>
  <a href="https://github.com/onflux-tech/dockeeper/commits/production"><picture><source media="(prefers-color-scheme: dark)" srcset="https://www.shieldcn.dev/github/last-commit/onflux-tech/dockeeper.svg?variant=secondary&amp;size=sm&amp;mode=dark"><img alt="Last commit" src="https://www.shieldcn.dev/github/last-commit/onflux-tech/dockeeper.svg?variant=secondary&amp;size=sm&amp;mode=light"></picture></a>
  <a href="https://github.com/onflux-tech/dockeeper/stargazers"><picture><source media="(prefers-color-scheme: dark)" srcset="https://www.shieldcn.dev/github/stars/onflux-tech/dockeeper.svg?variant=secondary&amp;size=sm&amp;mode=dark"><img alt="GitHub Stars" src="https://www.shieldcn.dev/github/stars/onflux-tech/dockeeper.svg?variant=secondary&amp;size=sm&amp;mode=light"></picture></a>
</p>

<p align="center">
  <a href="https://go.dev/"><picture><source media="(prefers-color-scheme: dark)" srcset="https://www.shieldcn.dev/badge/Language-Go-00ADD8.svg?logo=go&amp;variant=branded&amp;size=sm&amp;mode=dark"><img alt="Language · Go" src="https://www.shieldcn.dev/badge/Language-Go-00ADD8.svg?logo=go&amp;variant=branded&amp;size=sm&amp;mode=light"></picture></a>
  <a href="https://www.docker.com/"><picture><source media="(prefers-color-scheme: dark)" srcset="https://www.shieldcn.dev/badge/Runtime-Docker-2496ED.svg?logo=docker&amp;variant=branded&amp;size=sm&amp;mode=dark"><img alt="Runtime · Docker" src="https://www.shieldcn.dev/badge/Runtime-Docker-2496ED.svg?logo=docker&amp;variant=branded&amp;size=sm&amp;mode=light"></picture></a>
  <a href="https://react.dev/"><picture><source media="(prefers-color-scheme: dark)" srcset="https://www.shieldcn.dev/badge/Framework-React-61DAFB.svg?logo=react&amp;variant=branded&amp;size=sm&amp;mode=dark"><img alt="Framework · React" src="https://www.shieldcn.dev/badge/Framework-React-61DAFB.svg?logo=react&amp;variant=branded&amp;size=sm&amp;mode=light"></picture></a>
  <a href="https://pocketbase.io/"><picture><source media="(prefers-color-scheme: dark)" srcset="https://www.shieldcn.dev/badge/Database-PocketBase-1F2937.svg?variant=branded&amp;size=sm&amp;mode=dark"><img alt="Database · PocketBase" src="https://www.shieldcn.dev/badge/Database-PocketBase-1F2937.svg?variant=branded&amp;size=sm&amp;mode=light"></picture></a>
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
- [Configuration](#configuration)
- [Security](#security)
- [Requirements and Ports](#requirements-and-ports)
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

### Configuration

The manager reads three environment variables and keeps the rest in the dashboard.

| Variable | Default | Purpose |
|---|---|---|
| `DK_GRPC_HOSTS` | `localhost` | Comma-separated DNS names and IPs added as SANs to the manager certificate. Required when agents connect through a hostname other than `localhost`. |
| `DOCKEEPER_ENCRYPTION_KEY` | key generated in `pb_data/.encryption_key` | Encrypts registry credentials at rest. |
| `DOCKEEPER_BASE_URL` | request origin | Public URL used in notifications and generated links. |

The ports are fixed: `8090` serves the dashboard, the HTTP API and the agent install endpoints; `50051` serves gRPC for the agents. The manager data lives in `/home/dockeeper/pb_data` and holds the SQLite database, the internal CA and the encryption key. That volume is the only state to back up.

The agent takes three flags: `--manager-url` (`host:50051`), `--token` (enrollment token generated in the dashboard) and `--data-dir` (`/var/lib/dockeeper-agent` in the systemd install, `/data` in the container). A per-server token only matches that pending server and is cleared on use; the global quick-install token stays valid until you regenerate it. After enrollment the agent authenticates with its client certificate and renews it before expiry.

Metrics and log retention is set through `GET` and `PUT /api/dk/settings/retention`:

| Key | Default |
|---|---|
| `retention_raw_hours` | `168` (7 days) |
| `retention_hourly_days` | `30` |
| `retention_daily_days` | `365` |
| `retention_events_days` | `30` |
| `retention_audit_logs_days` | `90` |
| `retention_notification_logs_days` | `30` |

Backup policies keep their own `retention_days`.

### Security

- **Mutual TLS** -- the manager generates an internal CA on first start under `pb_data/certs/`: `ca.pem` and `ca-key.pem` (10 years), `server.pem` and `server-key.pem` (5 years, covering `localhost`, `dockeeper-manager`, the loopback IPs and every entry in `DK_GRPC_HOSTS`, and regenerated when those SANs change).
- **Agent certificates** -- issued with CN set to the server ID, valid for one year and renewed by the agent at 80% of the lifetime. The manager identifies every RPC by that CN.
- **Outbound-only topology** -- agents dial the manager. Managed hosts expose no inbound ports, and the manager never opens a connection to an agent.
- **HTTP authentication** -- the PocketBase superuser, an `X-API-Key` header or a 15-minute JWT issued to `dk_users`. Every custom route is bound to an RBAC capability such as `containers.manage` or `settings.view`; the public exceptions are health, agent install and action tokens.
- **Least privilege** -- users receive capabilities directly or through groups, and API keys act as their owner.
- **Audit log** -- mutations record the actor, the target and the payload.
- **Secrets at rest** -- registry credentials are encrypted with `DOCKEEPER_ENCRYPTION_KEY` or the generated key in `pb_data/.encryption_key`.

The data volume holds the CA, the server key and the encryption key. Back it up as a unit: restoring the database without these files breaks agent trust and stored credentials.

### Requirements and Ports

**Manager**: any Linux host with Docker, or the static binary. `amd64` and `arm64` images are published. The database is embedded (SQLite via PocketBase), so there is no external service to provision.

**Managed host**: Linux with Docker Engine reachable through `/var/run/docker.sock`, plus root or sudo for the systemd install. The container install mounts the socket instead. No inbound port is needed.

| From | To | Port | Notes |
|---|---|---|---|
| Browser | manager | `8090` | dashboard, HTTP API and agent install |
| Agent | manager | `50051` | gRPC over mTLS, outbound from the host |
| Agent | Docker Engine | `/var/run/docker.sock` | local socket |
| Agent | container registries | `443` | image pulls |

**From source**: Go 1.25+, Node.js 22+, npm, Docker and `air`.

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

Released under the [MIT License](LICENSE).

---

<a id="portugues"></a>

## Português

### Índice

- [Visão geral](#visão-geral)
- [Principais recursos](#principais-recursos)
- [Estado do projeto](#estado-do-projeto)
- [Início rápido](#início-rápido)
- [Configuração](#configuração)
- [Segurança](#segurança)
- [Requisitos e portas](#requisitos-e-portas)
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

### Configuração

O manager lê três variáveis de ambiente e mantém o restante no dashboard.

| Variável | Padrão | Função |
|---|---|---|
| `DK_GRPC_HOSTS` | `localhost` | Nomes DNS e IPs, separados por vírgula, adicionados como SANs ao certificado do manager. Necessário quando os agents chegam por um hostname diferente de `localhost`. |
| `DOCKEEPER_ENCRYPTION_KEY` | chave gerada em `pb_data/.encryption_key` | Criptografa as credenciais de registry em repouso. |
| `DOCKEEPER_BASE_URL` | origem da requisição | URL pública usada em notificações e links gerados. |

As portas são fixas: `8090` serve o dashboard, a API HTTP e os endpoints de instalação do agent; `50051` serve o gRPC para os agents. Os dados do manager ficam em `/home/dockeeper/pb_data` e contêm o banco SQLite, a CA interna e a chave de criptografia. Esse volume é o único estado a salvar.

O agent recebe três flags: `--manager-url` (`host:50051`), `--token` (token de enrollment gerado no dashboard) e `--data-dir` (`/var/lib/dockeeper-agent` na instalação systemd, `/data` no container). O token por server só vale para aquele server pendente e é limpo após o uso; o token global do quick install continua válido até você regenerá-lo. Depois do enrollment o agent autentica com o certificado de cliente e o renova antes do vencimento.

A retenção de métricas e logs é configurada por `GET` e `PUT /api/dk/settings/retention`:

| Chave | Padrão |
|---|---|
| `retention_raw_hours` | `168` (7 dias) |
| `retention_hourly_days` | `30` |
| `retention_daily_days` | `365` |
| `retention_events_days` | `30` |
| `retention_audit_logs_days` | `90` |
| `retention_notification_logs_days` | `30` |

Políticas de backup mantêm o próprio `retention_days`.

### Segurança

- **mTLS** -- o manager gera uma CA interna no primeiro start em `pb_data/certs/`: `ca.pem` e `ca-key.pem` (10 anos), `server.pem` e `server-key.pem` (5 anos, cobrindo `localhost`, `dockeeper-manager`, os IPs de loopback e cada entrada de `DK_GRPC_HOSTS`, e regenerados quando esses SANs mudam).
- **Certificados dos agents** -- emitidos com CN igual ao ID do server, válidos por um ano e renovados pelo agent em 80% do tempo de vida. O manager identifica cada RPC por esse CN.
- **Topologia somente de saída** -- os agents discam para o manager. Hosts gerenciados não expõem portas de entrada, e o manager nunca abre conexão para um agent.
- **Autenticação HTTP** -- superuser do PocketBase, header `X-API-Key` ou JWT de 15 minutos emitido para `dk_users`. Toda rota customizada está amarrada a uma capability de RBAC, como `containers.manage` ou `settings.view`; as exceções públicas são health, instalação de agent e action tokens.
- **Menor privilégio** -- usuários recebem capabilities direto ou por grupo, e API keys agem como o dono.
- **Audit log** -- mutações registram ator, alvo e payload.
- **Segredos em repouso** -- credenciais de registry são criptografadas com `DOCKEEPER_ENCRYPTION_KEY` ou com a chave gerada em `pb_data/.encryption_key`.

O volume de dados guarda a CA, a chave do server e a chave de criptografia. Faça backup como um conjunto: restaurar o banco sem esses arquivos quebra a confiança dos agents e as credenciais salvas.

### Requisitos e portas

**Manager**: qualquer host Linux com Docker, ou o binário estático. As imagens `amd64` e `arm64` são publicadas. O banco é embutido (SQLite via PocketBase), então não há serviço externo para provisionar.

**Host gerenciado**: Linux com Docker Engine acessível por `/var/run/docker.sock`, mais root ou sudo para a instalação systemd. A instalação em container monta o socket. Nenhuma porta de entrada é necessária.

| De | Para | Porta | Observação |
|---|---|---|---|
| Navegador | manager | `8090` | dashboard, API HTTP e instalação do agent |
| Agent | manager | `50051` | gRPC sobre mTLS, de saída do host |
| Agent | Docker Engine | `/var/run/docker.sock` | socket local |
| Agent | registries de container | `443` | pull de imagens |

**A partir do código**: Go 1.25+, Node.js 22+, npm, Docker e `air`.

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

Distribuído sob a [licença MIT](LICENSE).
