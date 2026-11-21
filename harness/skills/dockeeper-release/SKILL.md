---
name: dockeeper-release
description: "Use ao preparar ou publicar um release do dockeeper: versão, changelog, PR de promoção, tag, GoReleaser, GHCR e CI."
---

# Release

1. Leia `docs/release.md`.
2. Rode `make changelog-draft` e cure o `Unreleased` em `web/src/data/changelog.ts`.
3. Rode `make verify` e cole a saída real antes de promover.
4. Escolha a versão: antes de 1.0.0, breaking ou feature -> minor, correção -> patch.
5. Promova `Unreleased` para `x.y.z` em `chore/release-x.y.z`, commite e abra PR para `develop`; squash merge com autorização.
6. Rode `make release-check VERSION=x.y.z`.
7. Abra o PR de promoção `develop -> production` com resumo, `Closes #...` e o checklist; espere o `quality`.
8. Faça merge com **merge commit** (o único do fluxo) com autorização.
9. Crie a tag anotada no merge commit e peça autorização para o push da tag; acompanhe o GoReleaser.
