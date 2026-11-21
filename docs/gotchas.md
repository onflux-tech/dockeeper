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
**Causa:** o discovery faz `ContainerInspect` de todos só no full sync de 10 min; fora dele, inspeciona apenas os containers avisados por evento. `health_status` ainda gera um inspect sem mudar o payload, porque `ContainerInfo` não carrega saúde. O fallback `fetchFromLabels` do Traefik ainda faz `ListContainers` (inspect de todos) quando a API não responde. A detecção do Traefik já usa o resumo do `ContainerList` e inspeciona no máximo o container encontrado; `collectContainers` abre o stream antes de medir. `ListContainerStats` usa `ContainerStatsOneShot` com até 8 chamadas simultâneas e calcula o delta de CPU no agent com a amostra anterior por container, então o primeiro ciclo após o start tem CPU zero; depois de um intervalo sem coleta, o próximo valor é a média do período. O zero do primeiro ciclo entra no histórico e pode cair numa janela de alerta, e `cpu_percent` passa a ser a média entre coletas (~10 s), não a amostra curta do daemon. Containers e sistema continuam a cada 10 s: janelas de alerta de container podem ter 10 s e o avaliador busca amostras só nessa janela, então uma cadência maior deixaria a janela vazia e a regra sem avaliar.
**Correção:** meça antes de aumentar a frequência; não há memória de área versionada.

## EventBus bloqueado

**Sintoma:** eventos atrasam ou o manager trava.
**Causa:** handler síncrono demorado no `internal/manager/events/bus.go`, que é single-goroutine.
**Correção:** não bloqueie o handler; dispare trabalho pesado fora dele.

## Dois sistemas de auth

**Sintoma:** `e.Auth.Collection().Name` dá resultado inesperado; permissões divergem.
**Causa:** PocketBase tem `_superusers` e o projeto tem `dk_users` (JWT 15 min). O middleware trata os dois.
**Correção:** verifique `IsSuperuser()` antes de assumir a coleção; não presuma que todo auth é `dk_users`.

## Discovery para após falha de report

**Sintoma:** containers e imagens ficam com dados antigos no manager, enquanto redes e volumes continuam atualizando.
**Causa:** `changed()` grava o hash do payload antes do RPC; se o primeiro report falha (manager fora do ar, por exemplo), o payload estável nunca é reenviado.
**Correção:** `handleRPCError` apaga o hash do recurso para forçar o retry no próximo ciclo.

## PocketBase 0.39 limita texto a 5000 por padrão

**Sintoma:** import de compose falha com `Must be no more than 5000 character(s)` depois do upgrade do PocketBase.
**Causa:** a partir da 0.39, `TextField` com `Max: 0` passa a valer 5000.
**Correção:** a migration 15 sobe os campos existentes para 1.000.000; campo novo de texto longo precisa de `Max` explícito.

## `Closes` em PR para develop não fecha a issue

**Sintoma:** PR com `Closes #3` é mergeado em `develop` e a issue continua aberta.
**Causa:** o auto-close do GitHub só dispara no merge para o default branch, que é `production`.
**Correção:** mantenha o `Closes #N` no body do PR de promoção; o PR de feature só referencia a issue.

## Versão divergente

**Sintoma:** usuário, changelog e binário discordam da versão.
**Causa:** já houve tag `v0.8.0`, changelog `1.0.0` e `Makefile`/`cmd/*` em `0.1.0`.
**Correção:** a tag é a fonte única; o changelog só lista versões tagueadas e o `Unreleased`. O código usa `dev` e recebe a versão por ldflags. Rode `make changelog-check` antes do commit.

## `go test ./...` pegando `web/node_modules`

**Sintoma:** o teste ou o lint inclui `web/node_modules/flatted/golang`.
**Causa:** há um pacote Go dentro de `node_modules` e o `exclude-dirs: web` do golangci-lint não o cobre em todos os casos.
**Correção:** rode `go test ./internal/... ./cmd/...` quando quiser o escopo do projeto; ou ajuste o exclude para `web/node_modules`.

## Gates atuais

`make verify` roda `format-check`, `harness-check`, `harness-test`, `changelog-check`, `build-web`, `lint`, `test`, `lint-web`, `format-web` e `test-web`. O hook `pre-push` chama `make pre-push`, que roda `verify`; não há push remoto sem esses gates. O CI roda o job `quality` em push e pull request; `docker` só em push e depende de `quality`.

## Arquivos locais que não são versionados

Arquivos locais untracked não fazem parte da documentação versionada; não commitar. Se uma investigação local virar decisão, promova para issue. Configurações locais ignoradas não entram no repositório: a documentação versionada é `AGENTS.md`, `CLAUDE.md`, `docs/` e `.opencode/`.
