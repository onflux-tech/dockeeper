---
name: dockeeper-close
description: "Use ao fechar uma entrega do dockeeper: evolução do harness, changelog, verify, review com veredito durável, commit e PR."
---

# Fechamento de entrega

1. Leia o process (`Process: full|fast`), a spec, o plano e os critérios de aceite. No fast path, leia `tmp/<entrega>/process.md`; no full, o header do plano e a spec.
2. Liste o que a entrega ensinou e aplique a tabela de `docs/harness.md`: gotcha, convenção, lente de review ou procedimento repetido.
3. Edite a fonte em `harness/`, nunca a porta, e rode `make harness-sync`, `make harness-check` e `make harness-test`.
4. Atualize o Registro de evolução em `docs/harness.md`. Atualize o changelog do produto (`web/src/data/changelog.ts`) só se a entrega mudar o produto.
5. Rode `make verify`. Para docs, config ou harness sem mudança de comportamento, permissões ou comandos, o light review de `docs/review.md` vale; mudança de comportamento, permissões ou comandos exige o review completo.
6. Rode `/review` na base `git merge-base origin/develop HEAD`, com `git diff <merge-base>` mais untracked, porque a evolução ainda está no working tree. Grave reviewer, refuter e UI QA (quando `web/**`) e o veredito em `tmp/<entrega>/review.md`.
7. Triague: `UPHELD` e `INCONCLUSIVO` continuam; `REFUTED` não é corrigido. Corrija, rode `make verify` de novo e recheque os achados afetados. Só `GO` ou `GO CONFIRMADO` libera o commit.
8. Commite fonte e portas juntas, em inglês, conventional, imperativo. Peça autorização, faça push da branch e abra o PR para `develop`. Squash merge só com autorização; promoção é o fluxo do `dockeeper-release`.
9. Depois do merge, apague `tmp/<entrega>/` ou rode `make tmp-prune`.
