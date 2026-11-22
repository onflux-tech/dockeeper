---
# Gerado por make harness-sync a partir de harness/commands/review.md
description: Review do diff atual com reviewer e refutação adversarial
argument-hint: [base]
---

1. Despache o subagent `dockeeper-reviewer` para revisar o diff atual.
   - Sem argumento: working tree (`git status --short`, `git diff`, `git diff --cached` e untracked relevantes).
   - PR de feature: `git merge-base origin/develop HEAD` e `git diff <merge-base>...HEAD`.
   - PR de promoção: `git diff origin/production...origin/develop`.
   - Inclua no prompt: a base usada, os critérios de aceite e as mutações plausíveis (se existirem) e o contrato `docs/review.md`.
   - Não cole o diff no prompt; o agent busca.
   - Se o diff toca `web/**`, despache também o `dockeeper-ui-qa` na mesma base, com as rotas afetadas, os viewports e o contrato de `docs/review.md`. Ele roda em paralelo ao reviewer.

2. Com o relatório do reviewer em mãos, despache o subagent `dockeeper-refuter` na mesma base.
   - Cole no prompt apenas o relatório do reviewer, a base, os critérios de aceite e as mutações plausíveis.
   - Peça `UPHELD`, `REFUTED` ou `INCONCLUSIVO` por achado, a lente operacional e, se o reviewer deu `GO`, `GO CONFIRMADO` ou `GO REFUTADO`.
   - Não cole o diff no prompt; o agent busca.

3. Apresente os dois relatórios separados e aplique o contrato de `docs/review.md`:
   - `UPHELD` e `INCONCLUSIVO` continuam valendo; confirme cada Important no código antes de corrigir.
   - `REFUTED` não é corrigido; fica registrado com a prova do refuter.
   - `GO REFUTADO` e achado Critical/Important da lente operacional bloqueiam.
   - Minor refutado não some: vai para o relatório e o usuário tria.
   - Relatório do `dockeeper-ui-qa` entra separado. `NO-GO` de UI bloqueia o merge de frontend; `INCONCLUSIVO` volta para o usuário com o comando que faltou.

4. Não edite durante o review. Reporte os achados e espere a decisão.
