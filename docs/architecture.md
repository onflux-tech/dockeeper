# Arquitetura — Dockeeper

Referência detalhada do código. As invariantes curtas estão em `AGENTS.md`; este arquivo explica como as peças se encaixam.

## Visão geral

Sistema Go de orquestração e observabilidade multi-host para containers Docker (Podman/Kubernetes via enum). O **manager** é o servidor central: HTTP API, frontend embutido, DB PocketBase, gRPC server e orquestração de comandos. O **agent** roda em cada host gerenciado, lê o Docker socket local e conecta ao manager por gRPC mTLS.

Ações cobrem stop/remove/update/recreate de containers, backup/restore de volumes, terminal interativo (xterm.js + WebSocket), self-update do binário do agent, notificações multicanal, alertas e RBAC. Não é monitoramento passivo: permite ações destrutivas.

## Stack

- Go 1.25.1, `CGO_ENABLED=0` em produção.
- HTTP e DB: PocketBase 0.36.7 (roteador, auth, SQLite e migrations).
- SQLite puro Go: `modernc.org/sqlite` 1.46.2.
- gRPC 1.79.3 + protobuf 1.36.11, mTLS obrigatório.
- Object storage de backups: `minio/minio-go/v7` 7.0.99.
- Cron: `robfig/cron/v3` 3.0.1 via `app.Cron()`.
- Métricas de host: `shirou/gopsutil/v4` 4.26.2.
- WebSocket: `coder/websocket` 1.8.14.
- Logging: `log/slog` com `TextHandler` em `internal/shared/logging/logging.go`.
- Frontend: React 19, TypeScript 5.9, Vite 8, Tailwind 4, TanStack Query 5, React Router 7, Radix UI, Recharts, xterm.js, Zustand 5, Zod 4, react-hook-form. Testes com Vitest 4 + Testing Library.

## Manager e agent

O agent abre a conexão para o manager; o manager nunca disca para o host gerenciado. Zero portas inbound no host gerenciado.

Três streams gRPC, todos agent → manager:

1. **CommandChannel** (`AgentCommandChannelService.OpenChannel`): bidi persistente. O agent envia `ChannelInit` (`server_id`), heartbeats e resultados. O manager envia `ChannelCommand` multiplexado por `request_id` e sinais para abrir DataStream/ExecStream. A correlação usa `pending[request_id]` e canais Go; há fila de prioridade (`internal/manager/channel/priority.go`) e rate limiters por categoria.
2. **DataStream** (`OpenDataStream`): unidirecional para outputs contínuos (logs, image pull, compose up/down/pull, backup/restore, update de container, update do binário do agent). O manager sinaliza pelo CommandChannel; o agent abre o stream com `stream_id`. O manager aguarda em `streamWaiter[...]`.
3. **ExecStream** (`OpenExecStream`): bidi para terminal (`docker exec`), mesmo padrão de sinalização.

### Auth e mTLS

A CA interna ECDSA fica em `dataDir/certs/`. No enrollment, o manager assina o certificado do agent com CN = `server.Id`. As conexões gRPC seguintes usam mTLS; `ServerIDFromContext` extrai o CN do peer. O certificado do agent vale 1 ano e é renovado por um loop de 1h.

`DK_GRPC_HOSTS` injeta SANs extras no certificado do server. Em produção, precisa incluir o hostname externo do manager; sem isso, a validação TLS do agent falha por CN mismatch.

### Auth HTTP e RBAC

Três camadas em `internal/manager/middleware/rbac.go`:

1. Superuser do PocketBase (bypass).
2. Header `X-API-Key`.
3. Token JWT do PocketBase para `dk_users` (15 min).

As capabilities granulares ficam em `internal/shared/constants/capabilities.go`. Groups guardam JSON de capabilities; API keys têm hash SHA256, prefixo de 8 caracteres e capabilities próprias.

Exceções públicas conhecidas: `GET /api/dk/health`, download/install público de agent e action token. O restante das rotas customizadas precisa de `rbac.RequireCapability`.

### EventBus

`internal/manager/events/bus.go` é in-process, com buffer de 1000 eventos e uma única goroutine de dispatch. Handlers: `DBHandler` (persiste em `events`), `AuditHandler` e `AlertHandler`. Eventos incluem ServerEnrolled, ServerOnline/Offline, ContainerStarted/Stopped, BackupCompleted/Failed e HealthCheckFailed.

### Frontend embutido

`frontend.go` na raiz tem `//go:embed all:web/dist`. `FrontendDist` é passada para `manager.Bootstrap()` → `registerFrontendHandler()`, que serve a SPA como fallback para tudo que não começa com `/api/` ou `/_/`. Em desenvolvimento, o Vite faz proxy de `/api` para `localhost:8090`.

## Estrutura de pastas

```text
cmd/
  manager/main.go          entrypoint do manager
  agent/main.go            entrypoint do agent

internal/
  manager/
    app.go                 Bootstrap(): wiring de stores, services, rotas, jobs, EventBus, CA e gRPC
    grpc/                  handlers gRPC (enrollment, metrics, discovery, traefik, command channel, data/exec stream)
    channel/               AgentChannel, Registry (server_id → AgentChannel), PriorityQueue
    routes/                um arquivo por domínio (containers, images, compose, backups, alerts, ...)
    services/              AlertEngine, BackupService, CascadeService, UpdateOrchestrator, SelfUpdateService, CertAuthority, ...
    store/                 wrappers PocketBase por coleção (RBACStore, MetricsStore, BackupStore, AuditStore, ...)
    events/                Bus, tipos e handlers (DB, Audit, Alert)
    jobs/                  schedulers cron (agregação de métricas, alert evaluator, backup scheduler, cleanup)
    hooks/                 hooks do PocketBase (AdminSetup, AuditHooks, AgentOfflineChecker, AuditCleanup)
    middleware/            RBAC, RateLimiter (auth + por categoria), Auth
    notify/                Channel interface + Telegram, Slack, Discord, Webhook, SMTP, WhatsApp, ntfy

  agent/
    agent.go               Agent, Run(), ensureEnrolled(), certRenewalLoop()
    config.go              Config em YAML (dk-agent-data/config.yaml)
    enrollment.go          Enroll() HTTP para o endpoint público
    detector.go            DetectOrchestrator() → OrchestratorClient
    cert/manager.go        SaveCerts, TLSConfig, NeedsRenewal
    grpc_client.go         GRPCClient (Discovery, Metrics, Traefik, CommandChannel, DataStream, ExecStream, Conn)
    collector/             DiscoveryCollector, MetricsCollector, TraefikCollector, DockerClient
    executor/              DockerExecutor + ops (container, image, compose, network, volume, backup, update, exec, uninstall)
    grpc_handlers/         handlers RPC legacy usados pelo tunnel.Client
    tunnel/                CommandChannel, DataStream e ExecStream clients

  shared/
    constants/             collections.go, capabilities.go
    models/                models.go (structs Go que mapeiam o JSON das coleções), container_spec.go
    logging/               ConfigureDefault()
    crypto/                AES-GCM para registry credentials
    security/              sanitize.go, ssrf.go

migrations/                migrations numeradas (01_collections.go ... 14_alert_optional_fields.go)
proto/
  dockeeper/agent/v1/agent.proto   fonte única de verdade
  buf.yaml + buf.gen.yaml          geração via buf
  gen/dockeeper/agent/v1/          gerado; não editar
web/                       React + Vite; dist embutido no binário
frontend.go                //go:embed all:web/dist
.air.manager.toml / .air.agent.toml
Makefile
.goreleaser.yml
docker-compose.yml
.golangci-lint.yml
```

Números medidos em 20/11/2026: ~31K LOC Go, 178 arquivos `.go` sem `proto/gen`, ~31,8K LOC TS/TSX, 35 collections, 17 services gRPC, 93 messages, 14 migrations, 6 arquivos de teste Go e 3 de frontend.

## Coleções críticas

- `servers`: entidade central (status, `enrollment_token`, `cert_expires_at`, labels, orchestrator).
- `containers`, `images`, `networks`, `volumes`, `compose_stacks`: FK para `servers` com cascade.
- Métricas em três granularidades: `*_raw` (1 min), `*_hourly`, `*_daily`.
- `update_policies` / `update_history`, `backup_policies` / `backup_records`, `alert_rules` / `notification_channels` / `notification_logs`.
- `groups` / `group_members` / `api_keys`, `dk_users` (auth, JWT 15 min).
- `audit_logs`, `events`, `health_checks` / `health_check_logs`, `batch_jobs`, `tags`, `action_tokens`, `blackout_windows`, `traefik_routes`, `registry_credentials`, `terminal_sessions`, `settings`.

## Migrations

O arquivo `migrations/migrations.go` é apenas o pacote. O blank import que registra as migrations fica em `cmd/manager/main.go:14`. Cada migration usa `init()` e `pocketbase/migrations.Register`; a numeração é sequencial e não pode ser reutilizada.

Ao adicionar uma coleção ou campo, propague em: migration, constante em `internal/shared/constants/collections.go`, struct em `internal/shared/models/models.go` e tipos do frontend quando o campo aparecer na UI.

## Pontos de atenção estruturais

- O `categoryLimiter` é criado em `internal/manager/app.go:38`, mas nenhuma rota usa `ForCategory` hoje. Se for adicionar rate limit por categoria, use o limiter existente em vez de criar outro.
- A verificação de versão no enrollment não tem gate de compatibilidade de proto em runtime. Campo novo sem `make proto-gen` vira bug silencioso.
- O `DockerClient` faz chamadas concorrentes ao socket (discovery + metrics). Em hosts com muitos containers, isso satura o daemon; veja `docs/gotchas.md`.
