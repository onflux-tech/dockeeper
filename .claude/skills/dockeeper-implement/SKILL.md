---
# Gerado por make harness-sync a partir de harness/skills/dockeeper-implement/SKILL.md
name: dockeeper-implement
description: "Use ao implementar issue, feature ou correção no dockeeper, com critérios de aceite, TDD, gates e review antes do push."
---

# Implementação no dockeeper

1. Leia `AGENTS.md` (mapa, invariantes e comandos) e `docs/gotchas.md` da área.
2. Antes de editar, escreva 3 a 5 critérios de aceite no formato `comando -> saída esperada` e 1 a 3 implementações erradas plausíveis. Eles vão para o review.
3. Use `dockeeper-migration` se tocar collection, campo, índice ou relação. Rota customizada leva `rbac.RequireCapability`. Proto: edite o `.proto` e rode `make proto-gen`.
4. TDD onde existe suíte: Go em `internal/**`, web em `web/src/**/__tests__`.
5. Rode `make verify-fast` antes do commit. Antes do push, `make verify`.
6. Despache `dockeeper-reviewer` no diff atual e corrija os achados confirmados. Recheck só para concorrência, autorização, migration ou proto.
7. Commit em inglês, conventional, imperativo. Push só com autorização explícita.
