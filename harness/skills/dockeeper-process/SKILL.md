---
name: dockeeper-process
description: "Use ao iniciar pedido que muda código, schema, config, infra ou harness: gate de entrada, processo completo com spec e revisão, caminho rápido explícito e gates por fase."
---

# Processo gateado

Leia `docs/process.md` para o contrato. Este skill é o roteiro operacional.

## Gate 0

Antes de qualquer edição, chame a tool de ask do harness com uma pergunta e quatro opções, nesta ordem:

1. Processo completo (Recomendado)
2. Caminho rápido
3. Só diagnóstico
4. Já tenho spec/plano

Sem resposta, não edite. Leia só o necessário para classificar. Pergunta pura ou análise read-only não disparam o gate; a primeira edição sempre dispara, mesmo depois do diagnóstico. Registre a escolha no header do plano (`Process: full`) ou em `tmp/<YYYY-MM-DD>-<slug>/process.md` (`Process: fast`), com o escopo e os critérios de aceite.

- **Só diagnóstico:** analise e reporte, sem editar. Se o usuário pedir a correção depois, o gate volta.
- **Já tenho spec/plano:** peça os paths e comece na revisão da spec/plano ou na execução, sem refazer o brainstorming.

A entrada `/feature` já é a escolha do processo completo: não repita o Gate 0; comece no brainstorming.

## Processo completo

1. `brainstorming`: explore, escolha a abordagem e escreva a spec em `docs/superpowers/specs/`. A spec é sempre escrita; o caminho Bounded de versões antigas do skill não vale.
2. `/spec <spec>`: despache `dockeeper-spec-reviewer`, apresente o relatório e espere a triagem do usuário. Não edite a spec antes do aceite; depois, corrija Critical e Important e recheque. `PRONTO` só na versão final; uma mudança depois reabre o gate.
2b. Grill condicional: rode quando o review devolver decisão do usuário ou a spec trouxer termo/contrato novo. Invoque `grilling` + `domain-modeling` direto, nunca `grill-with-docs`. Uma pergunta por ask, com resposta recomendada; fatos você busca, decisões são do usuário. Registre Q/R na spec; `CONTEXT.md` e ADR só quando justificados. Mudou a spec, recheck focado.
3. `writing-plans`: escreva o plano em `docs/superpowers/plans/`. Não use o Execution Handoff do skill. Rode `/spec <spec> <plano>`; corrija, recheck e só então peça aprovação e o modo de execução.
4. `dockeeper-implement`: execute com TDD e `make verify-fast` antes de cada commit. O implement para nos commits; não rode `/review`, push ou PR. O despacho de subagente leva `process`, `spec`, `plan` e `task`; no caminho rápido, `spec` e `plan` vão como `none`; no completo, arquivo ausente para o worker.
5. Review: primeiro a evolução do harness e o changelog; depois `make harness-sync`, `make harness-check`, `make harness-test` e `make verify`. Rode `/review` em `git diff <merge-base>` + untracked e grave `tmp/<entrega>/review.md`. Triagem `UPHELD`/`INCONCLUSIVO`, correção, re-gate; só `GO` ou `GO CONFIRMADO` libera.
6. Close: commit do que restou, ask para push e PR, e apague `tmp/<entrega>/` depois do merge.

## Caminho rápido

Só para docs/config/harness puro que não muda comportamento, permissões ou comandos, correção de um arquivo ou UI dentro do padrão. Código: diagnóstico, critérios de aceite, TDD, `verify-fast` durante, `make verify` antes de `/review`, `/review`. Docs/config/harness: light review com `harness-check`/`harness-test` (ou `verify-fast`) e leitura focada, sem `/review`, refuter ou UI QA. Grave `tmp/<entrega>/process.md` com `Process: fast`.

Escale para o completo, com ask, se tocar migration/collection, proto/RPC, capability/RBAC, backup/restore, stream, auth ou contrato não provado, ou passar de 8 arquivos/5 tasks em código. Docs/config/harness puro é isento do limite; mudança de comportamento, permissões ou comandos escala.

## Bug

O processo completo começa com `systematic-debugging`; a spec é curta: sintoma, reprodução, causa raiz provada, correção, riscos e testes. O caminho rápido é só para bug trivial com causa raiz provada. Sem causa raiz, sem fix.

## Perguntas

Toda pergunta ao usuário usa a ask do harness: `AskUserQuestion` no Claude, `question` no OpenCode. Uma por chamada, opção recomendada primeiro. Isso vale também para as perguntas do `brainstorming` e do `writing-plans`; não pergunte em texto livre.
