---
name: dockeeper-implement
description: "Use ao implementar issue, feature ou correção no dockeeper, com critérios de aceite, TDD, gates e review antes do PR."
---

# Implementação no dockeeper

1. Leia `AGENTS.md` (mapa, invariantes e comandos) e `docs/gotchas.md` da área.
2. Crie a branch (`feat/`, `fix/` ou `chore/`) a partir de `develop`.
3. Antes de editar, escreva 3 a 5 critérios de aceite no formato `comando -> saída esperada` e 1 a 3 implementações erradas plausíveis. Eles vão para o review.
4. Use `dockeeper-migration` se tocar collection, campo, índice ou relação. Rota customizada leva `rbac.RequireCapability`. Proto: edite o `.proto` e rode `make proto-gen`.
5. TDD onde existe suíte: Go em `internal/**`, web em `web/src/**/__tests__`.
6. Rode `make verify-fast` antes do commit. Antes do push, `make verify`.
7. Rode `/review` no diff do PR (`merge-base` com `develop`) e corrija os achados que sobreviverem à refutação. Recheck só para concorrência, autorização, migration ou proto.
8. Commit em inglês, conventional, imperativo. Peça autorização, faça push da branch e abra o PR para `develop` com body em pt-BR e `Closes #N`.
9. Espere o `quality`; merge com squash só com autorização.
