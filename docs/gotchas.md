# Gotchas — Dockeeper

Falhas que não aparecem no CI atual. Leia antes de mexer na área correspondente.

## Migration que não roda

**Sintoma:** a coleção ou o campo não existe em runtime, mas o código compila.
**Causa:** arquivo em `migrations/` sem o blank import em `cmd/manager/main.go:14`, ou constante/model ausentes.
**Correção:** numeração sequencial, `init()` com `migrations.Register`, blank import em `cmd/manager/main.go`, constante em `internal/shared/constants/collections.go` e struct em `internal/shared/models/models.go`. `migrations/migrations.go` é só o pacote; não coloque imports lá.

## Proto sem regenerar

**Sintoma:** campo novo ignorado pelo outro lado, comportamento estranho ou pânico.
**Causa:** editou `proto/dockeeper/agent/v1/agent.proto` e não rodou `make proto-gen`.
**Correção:** rode `make proto-gen` antes de buildar. Nunca edite `proto/gen/*.pb.go`.

## Rota sem RBAC

**Sintoma:** endpoint responde para qualquer usuário autenticado, ou sem auth dependendo do middleware.
**Causa:** rota customizada sem `rbac.RequireCapability(constants.Cap*)`.
**Correção:** adicione a capability. Exceções públicas: `/api/dk/health`, download/install público de agent e action token. O `categoryLimiter` existe em `internal/manager/app.go:38`, mas nenhuma rota usa `ForCategory`; se for aplicar rate limit por categoria, use o limiter existente.

## Frontend embutido desatualizado

**Sintoma:** o binário serve assets antigos, ou loga “frontend embed not available, skipping SPA handler”.
**Causa:** `make build-manager` sem `make build-web` antes.
**Correção:** use `make build-all` ou rode `make build-web` primeiro. Em CI o GoReleaser já faz `make build-web`.

## `make build-web` com EPERM no Windows

**Sintoma:** o Vite falha em `web/dist/assets` com `EPERM, Permission denied`.
**Causa:** build anterior, dev server ou antivírus ainda segurando a pasta.
**Correção:** pare o processo que usa `web/dist`, remova o artefato (`rm -rf web/dist`) e rode `make build-web` de novo. `web/dist` é saída de build e não é versionado.

## Perda do `.encryption_key`

**Sintoma:** registry credentials viram lixo; erros de decrypt.
**Causa:** o arquivo `dk_data/.encryption_key` não está no backup.
**Correção:** inclua `dk_data/` inteiro no backup. Sem a chave, as credenciais salvas são inacessíveis.

## TLS do agent falhando em produção

**Sintoma:** validação TLS falha com CN mismatch.
**Causa:** `DK_GRPC_HOSTS` sem o hostname externo do manager.
**Correção:** inclua o hostname externo nas SANs do server cert. Em dev, `manager:50051` funciona pelo DNS interno.

## Self-update e download de agent em repo privado

**Sintoma:** download do agent ou self-update retorna 404; checksum falha.
**Causa:** `internal/manager/routes/servers.go` e `internal/manager/services/self_update.go` chamam a API do GitHub sem token; o repo é privado. O checksum também retorna `nil` em erro de download/parse, ou seja, falha aberto.
**Correção:** usar token/proxy autenticado ou tornar os assets públicos; corrigir o checksum para falhar fechado antes de confiar no self-update.

## Imagem `latest` reportando `latest`

**Sintoma:** o self-update sempre acha que há versão nova.
**Causa:** o CI publica `manager:latest` com `VERSION=latest`; `isNewerVersion` trata não-semver como dev.
**Correção:** use tag de versão/digest em deploy; não use `:latest` como fonte de verdade. O release do GoReleaser publica a versão correta, mas o CI também reescreve `latest`.

## Custo do Docker socket no agent

**Sintoma:** host com muitos containers fica lento; `ContainerStats` demora minutos; discovery faz inspect demais.
**Causa:** `collectContainers` mede todos os containers antes de abrir o stream; discovery faz `ContainerInspect` de todos a cada 30 s; Traefik inspeciona todos a cada 5 min.
**Correção:** meça antes de aumentar a frequência; não há memória de área versionada.

## EventBus bloqueado

**Sintoma:** eventos atrasam ou o manager trava.
**Causa:** handler síncrono demorado no `internal/manager/events/bus.go`, que é single-goroutine.
**Correção:** não bloqueie o handler; dispare trabalho pesado fora dele.

## Dois sistemas de auth

**Sintoma:** `e.Auth.Collection().Name` dá resultado inesperado; permissões divergem.
**Causa:** PocketBase tem `_superusers` e o projeto tem `dk_users` (JWT 15 min). O middleware trata os dois.
**Correção:** verifique `IsSuperuser()` antes de assumir a coleção; não presuma que todo auth é `dk_users`.

## Versão tripla

**Sintoma:** usuário, changelog e binário discordam da versão.
**Causa:** tag `v0.8.0`, `web/src/data/changelog.ts` em `1.0.0`, `Makefile`/`cmd/*` em `0.1.0`.
**Correção:** a tag é a fonte de verdade; atualize o changelog no release e não confie nos defaults `0.1.0`/`0.0.0`.

## `go test ./...` pegando `web/node_modules`

**Sintoma:** o teste ou o lint inclui `web/node_modules/flatted/golang`.
**Causa:** há um pacote Go dentro de `node_modules` e o `exclude-dirs: web` do golangci-lint não o cobre em todos os casos.
**Correção:** rode `go test ./internal/... ./cmd/...` quando quiser o escopo do projeto; ou ajuste o exclude para `web/node_modules`.

## Gates atuais

`make verify` roda `format-check`, `build-web`, `lint`, `test`, `lint-web`, `format-web` e `test-web`. O hook `pre-push` chama `make pre-push`, que roda `verify`; não há push remoto sem esses gates. O CI roda o job `quality` em push e pull request.

## Arquivos locais que não são versionados

Arquivos locais untracked não fazem parte da documentação versionada; não commitar. Se uma investigação local virar decisão, promova para issue. Configurações locais ignoradas não entram no repositório: a documentação versionada é `AGENTS.md`, `CLAUDE.md`, `docs/` e `.opencode/`.
