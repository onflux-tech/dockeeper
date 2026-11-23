---
# Gerado por make harness-sync a partir de harness/skills/dockeeper-implement/SKILL.md
name: dockeeper-implement
description: "Use para executar a implementação no dockeeper: processo escolhido, critérios de aceite, TDD, verify-fast e commits; para nos commits."
---

# Execução no dockeeper

1. Leia `AGENTS.md`, `docs/process.md` e `docs/gotchas.md` da área.
2. Confirme a escolha do processo. O despacho traz `process`, `spec`, `plan` e `task`; no caminho rápido, `spec` e `plan` vêm como `none`. No processo completo, os quatro precisam existir e a spec e o plano precisam estar aprovados; arquivo ausente, pare e peça ao pai.
3. Crie a branch (`feat/`, `fix/` ou `chore/`) a partir de `develop`.
4. Antes de editar, escreva 3 a 5 critérios de aceite no formato `comando -> saída esperada` e 1 a 3 implementações erradas plausíveis. Grave os critérios em `tmp/<entrega>/process.md` ou no plano.
5. Use `dockeeper-migration` se tocar collection, campo, índice ou relação. Rota customizada leva `rbac.RequireCapability`. Proto: edite o `.proto` e rode `make proto-gen`.
6. TDD onde existe suíte: Go em `internal/**`, web em `web/src/**/__tests__`.
7. Rode `make verify-fast` antes de cada commit e cole a saída.
8. Commit em inglês, conventional, imperativo.
9. Pare nos commits. `/review`, a evolução do harness, o `make verify` completo, o push e o PR pertencem ao `dockeeper-close`.
