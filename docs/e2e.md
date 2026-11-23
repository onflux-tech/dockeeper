# E2E — Dockeeper

Validação real de UI e fluxo, além dos testes unitários. Não substitui `make test`/`npm test`; prova o que o usuário vê.

## Browser

- Use a ferramenta de browser disponível na sessão. `browse` (OpenCode) é o padrão para a UI local e para o Chrome já logado; `browser` é o browser in-app quando a sessão oferece.
- `browser-harness` (CDP) fica para execução scriptada, headless, paralela ou cloud.
- Não use dois browsers na mesma aba; escolha um e mantenha o mesmo até o fim do fluxo.
- Console, rede e screenshots são evidência. Não declare “funciona” sem eles.

## Local

### Pré-requisitos

- Go 1.25.1, Node 22+ (CI) ou 24 (local), npm, Docker rodando.
- `golangci-lint` v2, `buf`, `protoc-gen-go`, `protoc-gen-go-grpc` para o build completo.
- Air para `make dev-manager`/`make dev-agent`; ou use `make dev` para rodar o manager direto.

### Subir

```bash
make build-web      # garante o dist embutido atualizado
make dev-manager    # manager em 8090/50051, dados em ./dk_data
make dev-agent      # agent em localhost:50051, dados em ./dk-agent-data
# opcional, para UI com hot reload:
make dev-web        # Vite em 5173 com proxy para 8090
```

Para isolar o e2e do ambiente de desenvolvimento, use um data dir separado:

```bash
./bin/dockeeper-manager serve --http=0.0.0.0:8090 --dir=./dk_data-e2e
```

### Primeiro admin

`GET /api/dk/auth/status` informa se o setup é necessário. Se sim:

```bash
curl -sS -X POST http://localhost:8090/api/dk/auth/setup \
  -H 'Content-Type: application/json' \
  -d '{"email":"e2e@local.test","password":"e2e-only-password"}'
```

Use credenciais de teste locais. Nunca use uma senha real de produção neste fluxo.

### Health

```bash
curl -sS http://localhost:8090/api/dk/health
```

### Fluxo com browser

1. Abra `http://localhost:8090` (frontend embutido) ou `http://localhost:5173` (Vite).
2. Faça snapshot da página; preencha email/senha; clique em login.
3. Espere o dashboard carregar; leia console e rede.
4. Navegue pela matriz abaixo; tire screenshot dos estados relevantes.
5. Feche a aba ao terminar.

## Matriz mínima

| Área | O que validar | Cuidado |
|---|---|---|
| Auth | Setup, login, logout, sessão persistida, 401 sem token | usar usuário de teste local |
| Dashboard | Total de servers, online/offline e containers; comparar com a API/coleção | a contagem de containers usa `totalItems`, não só a página |
| Servers | Lista, detalhe, status, containers do server, ordenação | a ordenação da página do server é um bug conhecido (issue #2) |
| Containers | Lista, filtros, start/stop/restart, terminal | ações destrutivas só em container de teste local |
| Images | Lista, pull, remove, prune | após prune, a lista pode não atualizar (issue #1) |
| Alerts | Criar regra, listar, estado | não disparar notificação real sem pedir |
| API playground | Login, envio de request, resposta e erro | não usar API key real de produção |
| Configurações | Settings, RBAC, usuários | validar 401/403 quando aplicável |

## Evidência

Salve em `tmp/e2e/YYYY-MM-DD-<slug>.md` (ignorado pelo git) ou anexe na PR. O arquivo local é scratch; a evidência durável é o PR. `make tmp-prune` mantém relatórios e screenshots, mas remove binários, logs e estado de teste dentro de `tmp/e2e`.

```markdown
# E2E — <slug>

- Data:
- Commit:
- URL:
- Ambiente: local | staging
- Browser:

## Passos
1. ...

## Esperado x observado
- ...

## Console e rede
- Erros de console:
- Requests 5xx:

## Screenshots
- caminho/arquivo.png

## Veredito
- passou | falhou | bloqueado
```

## Staging futuro

- Host: definido em configuração local, não versionada. SSH pré-autorizado só quando o usuário pedir na mensagem atual; comece por leitura.
- Use stack, volume, host e credenciais separados do ambiente de produção.
- Pinar versão/digest da imagem; não usar `:latest`/`:develop` como verdade.
- Tenha um host de teste com Docker para o agent, ou um agent local apontando para o manager de staging.
- Nunca rode ação destrutiva em produção; staging também exige cuidado com volumes.
- Credenciais de staging documentadas neste guia podem ser usadas; credenciais reais são digitadas pelo usuário no browser.

## Limpeza

- Pare manager/agent/web ao terminar.
- Remova containers/imagens de teste que você criou.
- Não remova volumes do Docker nem faça prune global; a proteção local bloqueia.
- Não deixe abas de browser abertas sem necessidade.
