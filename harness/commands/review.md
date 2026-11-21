---
description: Review read-only do diff atual contra AGENTS.md e docs/review.md
argument-hint: [base]
---

Despache o subagent `dockeeper-reviewer` para revisar o diff atual.

- Sem argumento: working tree (`git status --short`, `git diff`, `git diff --cached` e untracked relevantes).
- Com base: `git merge-base $ARGUMENTS HEAD` e `git diff <merge-base>...HEAD`.
- Inclua no prompt: a base usada, os critérios de aceite e as mutações plausíveis (se existirem) e o contrato `docs/review.md`.
- Não cole o diff no prompt; o agent busca.
- Não edite durante o review. Reporte os achados e espere a decisão.
