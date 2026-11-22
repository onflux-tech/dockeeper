---
description: QA de UI/UX do frontend web no diff atual, com browser e screenshots
argument-hint: [base|rota]
---

1. Determine a base do diff como no `/review` e confirme que ele toca `web/**`. Se não tocar, responda que não há frontend para validar e pare.
2. Despache o subagent `dockeeper-ui-qa` com a base, as rotas afetadas (ou `$ARGUMENTS` quando for rota) e o contrato de `docs/review.md`.
3. Apresente o relatório. `NO-GO` bloqueia o merge de frontend; `INCONCLUSIVO` volta para o usuário com o comando que faltou.
4. Não edite durante o QA.
