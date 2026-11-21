---
name: dockeeper-release
description: "Use ao preparar ou publicar um release do dockeeper: versão, changelog, tag, GoReleaser, GHCR e CI."
---

# Release

1. Leia `docs/release.md`.
2. Rode `make changelog-draft` e cure o `Unreleased` em `web/src/data/changelog.ts`.
3. Rode `make verify` e cole a saída real antes de promover.
4. Escolha a versão: antes de 1.0.0, breaking ou feature → minor, correção → patch.
5. Promova `Unreleased` para `x.y.z` com a data do release e commite.
6. Rode `make release-check VERSION=x.y.z`.
7. Promova `production`, crie a tag anotada e faça o push do branch e da tag juntos.
8. Push, merge e tag só com pedido explícito do usuário.
