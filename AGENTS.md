# AGENTS.md — Dockeeper

Fonte única de instrução para Claude Code e OpenCode. O `CLAUDE.md` importa este arquivo; o detalhe fica em `docs/`.

## Interação

- **Toda pergunta ao usuário usa a tool de ask/question.** Nunca pergunte apenas em texto livre.
- A opção recomendada é sempre a primeira, marcada `(Recomendado)`.
- Uma pergunta por vez; espere a resposta antes de seguir.
- Comunicação em pt-BR. Código, commits, nomes de arquivo e mensagens de ferramenta ficam em inglês.
- Comece pelo resultado; não narre ação rotineira.

## Execução

- Antes de implementar uma mudança aprovada que envolva código, pergunte via ask/question como executar: **Nativo (Recomendado)** na sessão principal ou **Subagent-driven** (superpowers).
- O padrão é nativo. Subagents entram quando a mudança for paralelizável ou isolável por arquivo.
- A escolha não altera os gates: `make verify-fast` antes de commitar e `make verify` antes de push ou release.

## Autorizações

- **Commit local:** sempre autorizado ao fim de uma task, sem pedir.
- **Push remoto:** nunca automático. Peça com a tool de ask/question e espere um sim explícito. Autorização anterior não vale para o próximo push.
- **Merge e tag de release:** pedido explícito do usuário.
- O hook `.githooks/pre-push` roda `make pre-push`; nenhum push remoto passa sem format, lint e testes. Não use `--no-verify`.
- `--no-verify` e force-push em branch compartilhada são proibidos.
- **SSH de staging:** o host/alias fica em configuração local, não versionada. Só acesse quando o usuário pedir na mensagem atual, para depurar deploy ou aplicação. Comece por leitura; mutação só com confirmação. Não inicie SSH por conta própria.
- Arquivos locais untracked não fazem parte deste guia e não devem ser commitados.

## Mapa

- **Manager:** `cmd/manager/main.go`, `internal/manager/` (app, grpc, channel, routes, services, store, events, jobs, hooks, middleware, notify), `migrations/`, `proto/`, `web/`, `frontend.go`.
- **Agent:** `cmd/agent/main.go`, `internal/agent/` (agent, config, enrollment, detector, cert, grpc_client, collector, executor, tunnel, grpc_handlers).
- **Shared:** `internal/shared/` (constants, models, logging, crypto, security).
- **Topologia:** o agent abre a conexão para o manager; nunca o contrário. Três streams gRPC: CommandChannel (bidi persistente), DataStream (outputs contínuos) e ExecStream (terminal).
- **Auth:** mTLS com CN = `server.Id`; HTTP com superuser PocketBase, `X-API-Key` ou JWT de `dk_users` (15 min).
- **Portas e dados:** HTTP 8090, gRPC 50051; manager em `dk_data/`, agent em `dk-agent-data/`.
- **Frontend:** React 19 + Vite 8 + Tailwind 4 + TanStack Query 5; `web/dist` é embutido por `//go:embed all:web/dist`.

## Invariantes

- **HTTP e DB via PocketBase.** Proibido chi/gin/echo, Postgres, ORM e SQL raw (`app.DB().NewQuery(...)`).
- **Logger:** `log/slog` com `TextHandler`. Sem JSON, zerolog ou zap.
- **Erros:** `fmt.Errorf("contexto: %w", err)`; `errors.New` só fora de hot path.
- **Frontend:** `npm`, não pnpm.
- **Rotas customizadas:** `rbac.RequireCapability(constants.Cap*)` é obrigatório. Exceções públicas conhecidas: `/api/dk/health`, download/install público de agent e action token.
- **Migrations:** arquivo `migrations/NN_nome.go` com numeração sequencial; blank import em `cmd/manager/main.go:14`; constante em `internal/shared/constants/collections.go`; model em `internal/shared/models/models.go`.
- **Proto:** editar `proto/dockeeper/agent/v1/agent.proto` e rodar `make proto-gen`. Nunca editar `proto/gen/*.pb.go`.
- **Frontend embed:** `make build-web` antes de `make build-manager`. Em CI o GoReleaser já faz isso; no build manual, não esqueça.
- **EventBus:** `internal/manager/events/bus.go` é single-goroutine; não bloquear o handler.
- **Sem comentários no código.** Nenhum, inclusive doc comment.

## Comandos

| Comando | O que faz |
|---|---|
| `make dev-all` / `make run` | manager + agent + web em paralelo |
| `make dev-manager` | air do manager em 8090/50051 com `dk_data` |
| `make dev-agent` | air do agent em `localhost:50051` com `dk-agent-data` |
| `make dev-web` | Vite em 5173 com proxy para 8090 |
| `make build-all` | `build-web` → `build-manager` → `build-agent` |
| `make proto-gen` | `buf generate` em `proto/` |
| `make test` | `go test` em `cmd`, `internal` e `migrations` |
| `make lint` | `golangci-lint` em `cmd`, `internal` e `migrations` |
| `make test-web` | `vitest run` em `web/` |
| `make lint-web` | `eslint .` em `web/` |
| `make format` | `gofmt` + `goimports` no Go |
| `make format-check` | falha se `gofmt` precisar mudar algo |
| `make format-web` | `prettier --check` em `web/` |
| `make tools` | instala `goimports` |
| `make hooks` | instala o hook `pre-push` |
| `make pre-push` | `verify`, chamado pelo hook |
| `make verify` | format-check, build-web, lint, test, lint-web, format-web, test-web |
| `make verify-fast` | format-check, test, lint-web |
| `make release` | `goreleaser release --clean` |

## Baseline

Medido em 20/11/2026:

- `go test` passa.
- `golangci-lint` em `cmd`, `internal` e `migrations`: **0 issues**.
- `npm run lint` passa.
- `npm run format:check` passa.
- `npm test` passa: 18 testes.
- O CI roda `quality` em push e pull request (format, lint, build, test) e o build Docker apenas em push.

Não declare “pronto” sem colar a saída real. Uma falha só é pré-existente com prova de baseline (`git stash`, rodar, restaurar).

## Git e release

- Branches `develop` e `production`; histórico linear, sem merge commit e sem PR no histórico. O fluxo é commit em `develop`, promoção para `production` e tag anotada `vX.Y.Z`.
- Commits em inglês, conventional commits, tipo minúsculo, escopo por área e imperativo. Sem co-autoria.
- Release: push da tag dispara GoReleaser, que builda linux/darwin amd64/arm64, archives, checksums, changelog e imagens GHCR multi-arch.
- O changelog do produto fica em `web/src/data/changelog.ts`.
- Versão tripla: tag `v0.8.0`, changelog `1.0.0`, código `0.1.0`. A tag é a fonte de verdade até decisão contrária.
- Receita completa, o que o CI não faz e o deploy futuro: `docs/release.md`.

## E2E

- Validação local com `browse` (OpenCode) em `http://localhost:8090`: setup do admin, login, dashboard, servers, containers, images/prune, alerts e API playground.
- `browser-harness` fica para execução scriptada/headless.
- Staging futuro em um host definido em configuração local; SSH só quando o usuário pedir.
- Runbook e formato de evidência: `docs/e2e.md`.

## Notas internas

- Notas locais ficam em `tmp/` (ignorado) e não são versionadas. O código vence qualquer nota.

## Ponteiros

- `docs/architecture.md` — visão detalhada de manager/agent, streams, RBAC, EventBus, pastas e collections.
- `docs/gotchas.md` — falhas silenciosas e como evitá-las.
- `docs/release.md` — release, CI, GHCR e deploy futuro.
- `docs/e2e.md` — validação local e staging.
- `CLAUDE.md` — específico do Claude Code; importa este arquivo.

## Manutenção

- Números e caminhos deste arquivo envelhecem. Confira no código antes de repetir: collections, services, messages, migrations, testes.
- Não duplique o detalhe que já está em `docs/`; use ponteiro.
- Arquivos locais untracked não fazem parte deste guia; não commitar.
