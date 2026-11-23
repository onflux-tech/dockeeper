# Processo — Dockeeper

Contrato do processo gateado. Vale para o agente principal, para os subagentes e para Claude Code e OpenCode.

## Quando

- Qualquer pedido que mude código, schema, config, infra ou harness.
- Pergunta pura, análise read-only e diagnóstico não disparam o Gate 0; a primeira edição dispara, mesmo depois do diagnóstico.

## Gate 0

Antes de qualquer edição, uma pergunta com a ask do harness, nesta ordem:

1. **Processo completo (Recomendado)** — brainstorming, spec, revisão da spec, grill condicional, plano, revisão do plano, execução, review e close.
2. **Caminho rápido** — diagnóstico, critérios de aceite, TDD, gates e review leve.
3. **Só diagnóstico** — análise read-only, sem edição.
4. **Já tenho spec/plano** — começar na revisão ou na execução, com os paths.

Sem resposta, sem edição. Se a ask não estiver disponível ou falhar (run não interativo), pare e reporte; não assuma a escolha nem edite. A escolha fica no header do plano (`Process: full`) ou em `tmp/<YYYY-MM-DD>-<slug>/process.md` (`Process: fast`), com o escopo e os critérios de aceite. O despacho de subagente leva `process`, `spec`, `plan` e `task`; no caminho rápido, `spec` e `plan` vão como `none`, e no completo um arquivo ausente faz o worker parar. O close lê o process e `tmp/<entrega>/review.md`.

## Processo completo

| Fase | O que roda | Artefato | Gate |
|---|---|---|---|
| 0 Gate | ask | escolha | resposta |
| 1 Brainstorm | `brainstorming` | spec em `docs/superpowers/specs/` | aprovação do design; a spec sempre é escrita |
| 2 Spec review | `dockeeper-spec-reviewer` + `/spec` | relatório; spec revisada | `PRONTO`; Critical/Important resolvidos |
| 2b Grill | `grilling` + `domain-modeling` | Q/R na spec; `CONTEXT.md`/ADR | entendimento compartilhado; recheck |
| 3 Plan | `writing-plans` | plano em `docs/superpowers/plans/` | self-review; Execution Handoff suprimido |
| 3b Plan review | `/spec <spec> <plano>` | relatório; plano revisado | `PRONTO`; recheck; aprovação do plano |
| 4 Execution | `dockeeper-implement` | commits | ask nativo/subagent; `verify-fast` por commit; para nos commits |
| 5 Review | `/review` | `tmp/<entrega>/review.md` | evolução/changelog antes; `make verify`; `GO` |
| 6 Close | `dockeeper-close` | PR | ask para push/PR; apaga o tmp após o merge |

A spec é sempre escrita, inclusive quando o `brainstorming` classificar a mudança como pequena; o caminho Bounded de versões antigas do skill é sobrescrito.

O `writing-plans` termina com um Execution Handoff que pede o modo de execução. No processo completo, ele é suprimido: rode a revisão do plano, corrija, e só então peça a aprovação e o modo.

O `dockeeper-implement` para nos commits. `/review`, a evolução do harness, o `make verify` completo, o push e o PR pertencem ao close.

## Spec review

`dockeeper-spec-reviewer` é read-only e fresh-context. Ele lê a spec, o plano, `AGENTS.md`, `docs/gotchas.md`, `docs/harness.md` e o código, e aplica:

1. Escopo e YAGNI.
2. Contratos externos: enum real do proto, PocketBase, S3, saída de comando, gRPC, caminho legado e versão do agent.
3. Falhas e bordas: restore parcial, traversal por entrada do tar, segredo, config corrompida, concorrência, backfill de migration, rollback, cleanup e permissão.
4. Prontidão: interface real contra o package, ordem, migration, constante, grant, RBAC, proto, teste e aceite verificável.
5. Consistência: spec/plano, requisito/task, aceite/teste, UI/API.
6. Processo do repo: índice de docs, harness check, changelog e gates.

O relatório usa `Critical | Important | Minor`, `spec:linha` ou `plano:linha`, cenário concreto, o que falta decidir ou provar, `Review Focus: coberto | novo` e `PRONTO`/`NÃO PRONTO`. Uma mudança depois de `PRONTO` reabre o gate; o recheck é focado e devolve `ADDRESSED`/`NOT ADDRESSED` por achado. No modo plano, o cabeçalho é `## Plano`.

O `/spec` não edita a spec. O agente principal apresenta o relatório, espera a triagem do usuário e só então aplica as correções aprovadas e o recheck. A spec só é `PRONTO` na versão final.

## Grill

Condicional, depois da revisão da spec. Roda quando o review devolve uma decisão que só o usuário pode tomar ou a spec traz um termo, contrato ou decisão difícil de reverter.

Invoque `grilling` + `domain-modeling` direto; nunca `grill-with-docs`, cuja flag `disable-model-invocation` o OpenCode ignora. A árvore e as rodadas valem; cada pergunta usa a ask, uma por chamada, com resposta recomendada. Fatos o agente busca; decisões são do usuário. As respostas viram Q/R na spec. `CONTEXT.md` é criado sob demanda; ADR só quando a decisão é difícil de reverter, surpreendente sem contexto e resultado de um trade-off real. Mudou a spec, recheck focado.

## Caminho rápido

Só quando o usuário escolhe e a mudança é docs/config/harness puro sem alterar comportamento, permissões ou comandos, correção de um arquivo ou UI dentro do padrão existente.

- Código: diagnóstico (em bug), critérios de aceite, TDD, `verify-fast` durante, `make verify` antes de `/review`, e `/review`. Sem spec ou plano.
- Docs/config/harness: light review com `harness-check` e `harness-test` e leitura focada; a saída verde libera o commit, sem `/review`, refuter ou UI QA.

Grave `tmp/<entrega>/process.md` com `Process: fast`, o escopo e os gates.

Escale para o completo, com ask, se tocar migration/collection, proto/RPC, capability/RBAC, backup/restore, stream, auth ou contrato não provado, ou passar de 8 arquivos/5 tasks em código. Docs/config/harness puro é isento do limite; mudança de comportamento, permissões ou comandos escala.

## Bug

O processo completo começa com `systematic-debugging`. A spec é curta: sintoma, comando de reprodução, causa raiz provada, correção, riscos e testes. O caminho rápido é só para bug trivial com causa raiz provada. Sem causa raiz, sem fix.

## Perguntas

Toda pergunta ao usuário usa a ask do harness: `AskUserQuestion` no Claude, `question` no OpenCode. Uma por chamada, opção recomendada primeiro. Isso vale para o Gate 0, para o `brainstorming`, para o `writing-plans` e para o grill; não pergunte em texto livre.

## Registros

- `tmp/<YYYY-MM-DD>-<slug>/process.md`: `Process: full|fast`, escopo, spec, plano, critérios de aceite e fase.
- `tmp/<YYYY-MM-DD>-<slug>/review.md`: relatório do reviewer, do refuter e do UI QA quando houver, e o veredito.
- O PR é a evidência durável. O diretório local é scratch e é apagado depois do merge.

## Relação com o review

`docs/review.md` é o contrato do diff pós-implementação. A revisão de spec e de plano vem antes do código; o `/review` do diff vem depois da evolução do harness e do `make verify`, com `git diff <merge-base>` mais untracked quando a entrega ainda não foi commitada. Só `GO` ou `GO CONFIRMADO` libera commit, push e PR no processo completo; no caminho rápido de docs/config/harness, a saída verde de `harness-check` e `harness-test` libera o commit.

## Não é

- Substituto do `/review`, do refuter ou do UI QA.
- O plugin EGM.
- Processo automático sem a escolha do usuário.
- Review de diff; isso é `docs/review.md`.
