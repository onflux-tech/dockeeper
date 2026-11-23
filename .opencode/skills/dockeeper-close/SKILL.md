---
# Gerado por make harness-sync a partir de harness/skills/dockeeper-close/SKILL.md
name: dockeeper-close
description: "Use ao fechar uma entrega do dockeeper: verificação, review, evolução do harness, commit e PR."
---

# Fechamento de entrega

1. Rode `/verify` e cole a saída real.
2. Rode `/review` no diff do PR (`merge-base` com `origin/develop`) e corrija os achados que sobreviverem à refutação.
3. Liste o que a entrega ensinou e aplique os critérios de `docs/harness.md`: gotcha, convenção, lente de review ou procedimento repetido.
4. Edite a fonte em `harness/`, nunca a porta em `.claude/` ou `.opencode/`.
5. Rode `make harness-sync`, `make harness-check` e `make harness-test`.
6. Atualize o registro de evolução em `docs/harness.md`.
7. Commite fonte e portas juntas, em inglês, conventional, imperativo. Peça autorização, faça push da branch e abra o PR para `develop`. Squash merge só com autorização; promoção é o fluxo do `dockeeper-release`.
8. Depois do merge, apague `tmp/<entrega>/` ou rode `make tmp-prune`; a evidência durável já está no PR.
