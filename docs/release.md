# Release e deploy — Dockeeper

Receita real do projeto. A receita histórica fica em configuração local.

## Versão

- A tag git é a fonte única: `vX.Y.Z`.
- O changelog tem `Unreleased` no topo e as versões lançadas abaixo.
- Antes de 1.0.0: breaking ou feature → minor; correção → patch. `1.0.0` só com decisão explícita de estabilidade.
- O rascunho do 1.0.0 fica em `docs/releases/1.0.0.md`; ele só entra em `web/src/data/changelog.ts` quando a tag `v1.0.0` for criada.
- O código usa `dev` como default; `make` injeta o `git describe` e o GoReleaser injeta a versão da tag via ldflags.

## Fluxo atual

1. Rode `make changelog-draft` e cure o `Unreleased` em `web/src/data/changelog.ts`; as entradas entram nos PRs de feature ao longo do ciclo.
2. Em `develop`, rode `make verify` e registre a saída.
3. Escolha `x.y.z` pela política acima.
4. Crie `chore/release-x.y.z` a partir de `develop`, promova `Unreleased` para `x.y.z` com a data e commite.
5. Abra PR para `develop`, espere o `quality` e faça squash merge.
6. Em `develop`, rode `make release-check VERSION=x.y.z`; ele exige árvore limpa, tag ainda inexistente e entrada para a versão.
7. Abra o PR de promoção `develop -> production`, título `chore(release): x.y.z`, body com resumo, `Closes #N` das issues entregues e o checklist abaixo.
8. Espere o `quality` do PR e faça merge com **merge commit**; squash e rebase não valem para a promoção.
9. Atualize `production`, crie a tag anotada no merge commit e peça autorização para o push:
   ```bash
   git checkout production
   git pull --ff-only
   git tag -a vX.Y.Z -m "Dockeeper vX.Y.Z"
   git push origin vX.Y.Z
   ```
   O hook `pre-push` roda `make verify` com a tag já presente, então o `changelog-check` passa.
10. Acompanhe o workflow de release e confira assets e imagens GHCR.

## O que o GoReleaser faz

- `go mod tidy` e `make build-web` antes dos builds.
- Binários manager e agent para linux/darwin em amd64/arm64.
- Archives `.tar.gz`, `checksums.txt` SHA-256.
- Changelog agrupado por conventional commits.
- Quatro imagens Docker (manager/agent × amd64/arm64) e manifests `:<version>` + `:latest` no GHCR.
- Release no GitHub com link de comparação e bloco de imagens.

## O que o CI faz e não faz

- `.github/workflows/ci.yml`: `quality` em PR e push para `develop`/`production`; `docker` só em push e depende de `quality`.
- O PR de feature roda `quality` sem Docker; o PR de promoção também.
- No push em `production` pós-merge, `quality` e `docker` publicam `:latest`; a tag dispara `release.yml`.
- O release não roda `make verify` sozinho. O gate é o CI/pre-push; não crie tag com o CI vermelho.
- O hook `pre-push` executa `make pre-push` (`verify`).
- `make changelog-check` valida a estrutura do changelog contra as tags; ele roda no `make verify` e no CI.

## Checklist de release

- [ ] `make changelog-draft` executado e `Unreleased` curado.
- [ ] `make verify` verde em `develop` e saída registrada.
- [ ] `Unreleased` promovido para `x.y.z` com a data via PR squash em `develop`.
- [ ] `make release-check VERSION=x.y.z` verde.
- [ ] PR de promoção `develop -> production` aberto com `Closes #...` e checklist.
- [ ] `quality` do PR de promoção verde.
- [ ] Merge commit do PR de promoção feito.
- [ ] Tag anotada `vX.Y.Z` criada no merge commit.
- [ ] Push da tag autorizado pelo usuário.
- [ ] Workflow de release verde.
- [ ] Assets e imagens GHCR conferidos (`manager` e `agent`, amd64/arm64).
- [ ] Release notes revisadas no GitHub.

## Quando o release falha

- Se o PR de promoção falhar, corrija em `develop` com outro PR e atualize o PR de promoção.
- Não reutilize uma tag publicada. Crie outra versão ou peça decisão ao usuário.
- Não force push em tag ou branch compartilhada.
- Se a falha for no GoReleaser, confira o log do workflow e rode o release local com `GITHUB_TOKEN` e login no GHCR apenas se o usuário pedir.
- Se a falha for de versão/changelog, corrija e refaça o ciclo.

## Deploy

O deploy anterior foi removido em 20/11/2026. Não há deploy ativo.

O compose de referência fica em configuração local, fora do repositório: manager único, `DK_GRPC_HOSTS` com hostname externo, Traefik/TLS, porta 50051, volume `manager-data` em `/home/dockeeper/pb_data`, healthcheck em `/api/dk/health`.

Para o futuro staging em um host definido em configuração local:

- SSH é pré-autorizado na configuração local, mas só quando o usuário pedir na mensagem atual; comece por leitura.
- Use stack, volume, host e credenciais separados; nunca reutilize o volume de produção.
- Pinar versão ou digest em vez de `:develop`/`:latest`.
- Valide local primeiro; depois staging; produção só com decisão explícita.
- O volume contém SQLite e `.encryption_key`; perder o volume invalida as registry credentials.
