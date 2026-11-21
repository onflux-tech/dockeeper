# Release e deploy — Dockeeper

Receita real do projeto. A receita histórica fica em configuração local.

## Fluxo atual

1. Trabalhe e commite em `develop`. Commit local é autorizado; push remoto exige ask/question e sim explícito.
2. Promova para `production`:
   ```bash
   git checkout production
   git merge develop
   git push origin production
   ```
   O histórico é linear; como `develop` e `production` nunca divergiram, o merge é fast-forward.
3. Crie a tag anotada:
   ```bash
   git tag -a vX.Y.Z -m "Dockeeper vX.Y.Z"
   git push origin vX.Y.Z
   ```
   O push da tag é remoto: peça permissão antes.
4. O push da tag dispara `.github/workflows/release.yml` e o GoReleaser.
5. Atualize `web/src/data/changelog.ts` no mesmo ciclo do release.

## O que o GoReleaser faz

- `go mod tidy` e `make build-web` antes dos builds.
- Binários manager e agent para linux/darwin em amd64/arm64.
- Archives `.tar.gz`, `checksums.txt` SHA-256.
- Changelog agrupado por conventional commits.
- Quatro imagens Docker (manager/agent × amd64/arm64) e manifests `:<version>` + `:latest` no GHCR.
- Release no GitHub com link de comparação e bloco de imagens.

## O que o CI faz e não faz

- `ci.yml`: em push para `develop`/`production`, apenas chama o build Docker e publica `:develop`/`:latest`.
- `release.yml`: em tag `v*.*.*`, roda GoReleaser.
- Nenhum dos dois roda `go test`, `golangci-lint`, `npm run lint`, `npm test`, `buf lint` ou smoke test. O CI verde não prova qualidade.
- Rode `make verify-fast` antes de commitar e `make verify` antes de release. Hoje o baseline completo é vermelho (49 lint issues e 4 testes de dashboard); a correção do baseline fica para uma mudança separada.

## Versão

Há três fontes concorrentes:

- Tag: `v0.8.0` (última).
- Changelog do produto: `web/src/data/changelog.ts` em `1.0.0`.
- Código/Makefile: `0.1.0`; `web/package.json` em `0.0.0`.

A tag é a fonte de verdade até decisão contrária. O release deve atualizar o changelog para a mesma versão; os defaults `0.1.0`/`0.0.0` são placeholders e não devem ser citados como versão do produto.

## Checklist de release

- [ ] `make verify` executado e saída registrada (ou baseline conhecido documentado).
- [ ] `web/src/data/changelog.ts` atualizado com a versão e a data.
- [ ] `develop` limpo e commits revisados.
- [ ] Promoção para `production` feita.
- [ ] Tag anotada `vX.Y.Z` criada.
- [ ] Push da tag autorizado pelo usuário.
- [ ] Workflow de release verde.
- [ ] Assets e imagens GHCR conferidos (`manager` e `agent`, amd64/arm64).
- [ ] Release notes revisadas no GitHub.

## Quando o release falha

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
