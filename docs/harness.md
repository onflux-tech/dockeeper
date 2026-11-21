# Harness — Dockeeper

Fonte única, portas geradas e checagem de drift para Claude Code e OpenCode.

| Campo | Valor |
|---|---|
| Fontes | `harness/skills/`, `harness/commands/`, `harness/agents/` |
| Portas | `.claude/{skills,commands,agents}/` e `.opencode/{skills,commands,agents}/` |
| Gerador | `make harness-sync` (`scripts/harness.mjs sync`) |
| Checagem | `make harness-check` (`scripts/harness.mjs check`) |
| Testes do gerador e do changelog | `make harness-test` (`node --test scripts/harness.test.mjs scripts/changelog.test.mjs`) |

## Regra de ouro

Edite `harness/` e rode `make harness-sync`. Não edite as portas à mão; o header `Gerado por make harness-sync` marca o que é gerado. `make harness-check` falha em drift, referência quebrada, doc fora do índice, target inexistente, migration fora de sequência, collection sem constante, frontmatter inválido, path pessoal ou segredo.

## Fluxo de evolução

Ao fim de uma entrega, depois de `/verify` e `/review`:

1. Liste o que a entrega ensinou.
2. Aplique a tabela abaixo.
3. Rode `make harness-sync`, `make harness-check` e `make harness-test`.
4. Registre a mudança em `## Registro de evolução`.
5. Commite fonte e portas juntas.

| O que a entrega ensinou | Destino | Quem decide |
|---|---|---|
| Armadilha nova | `docs/gotchas.md` | agente |
| Convenção que se repete | `AGENTS.md` ou `docs/review.md` | agente |
| Lente de review recorrente | corpo de `harness/agents/dockeeper-reviewer/body.md` | agente |
| Procedimento repetido em duas entregas | skill em `harness/skills/` | usuário |
| Comando, path ou collection novo | `AGENTS.md`, `docs/` e o check | agente |
| Ferramenta ou MCP novo | `.mcp.json` e `.opencode/opencode.json` | usuário |
| Papel novo | `harness/agents/` | usuário |

## Antes de mexer em X, leia Y

| Vou mexer em | Leia |
|---|---|
| manager/agent, streams, auth, EventBus | `docs/architecture.md` |
| migration, collection, campo, índice | `docs/migrations.md` e `docs/gotchas.md` |
| proto | `docs/gotchas.md` e `proto/dockeeper/agent/v1/agent.proto` |
| rota customizada | `AGENTS.md` e `internal/manager/middleware/rbac.go` |
| frontend embutido | `docs/gotchas.md` e `Makefile` |
| review, push ou achado | `docs/review.md` |
| release, tag ou GHCR | `docs/release.md` |
| skill, command ou agent | este arquivo e `harness/README.md` |

## Registro de evolução

| Data | Mudança |
|---|---|
| 2026-11-20 | Fonte única em `harness/`, `make harness-sync`/`harness-check` e skill `dockeeper-close`. |
| 2026-11-20 | Changelog com `Unreleased` e versões tagueadas, `changelog-draft`/`changelog-check`/`release-check` e gate de changelog no `verify` e no CI. |
| 2026-11-21 | Fluxo com PR: feature em `develop` por squash, promoção em `production` por merge commit, template de PR e base de review por PR. |

## Cadência

- `make harness-check` roda em `make verify`, no pre-push e no CI.
- O `/review` roda no diff do PR antes do merge; a promoção revisa contra `origin/production`.
- No release, rode `make harness-check` e confira o registro de evolução.
- Quando o check falhar, corrija a fonte e regenere; não edite a porta.
- Prosa desatualizada não é detectável por check; o fechamento de cada entrega é o ponto de revisão.
