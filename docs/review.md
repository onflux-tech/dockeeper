# Review — Dockeeper

Contrato de review pós-implementação. Vale para o agente principal, para o `dockeeper-reviewer`, para o `dockeeper-refuter` e para o `/review`.

## Quando

- Antes de abrir ou atualizar o PR, depois de `make verify`.
- Em mudança de código; docs, config e harness puros não pedem review completo.
- Em issue, feature ou refactor amplo, o review acontece antes de declarar pronto.

## Base do diff

- Working tree por padrão: `git status --short`, `git diff`, `git diff --cached` e untracked relevantes.
- PR de feature: `git merge-base origin/develop HEAD` e `git diff <merge-base>...HEAD`.
- PR de promoção: `git diff origin/production...origin/develop`.
- Range de commits: base explícita no despacho. Use `git merge-base <base> HEAD` e `git diff <merge-base>...HEAD`; nunca `HEAD~1`.
- O revisor busca o diff sozinho. Não cole o diff no prompt.

## Eixos separados, sem rerank

### Standards

O diff segue `AGENTS.md` e `docs/gotchas.md`? Confira:

- PocketBase para HTTP e DB; sem chi/gin/echo, Postgres, ORM ou SQL raw.
- `log/slog` com `TextHandler`; sem JSON, zerolog ou zap.
- Erros com `fmt.Errorf("contexto: %w", err)`; sem `errors.New` em hot path.
- Rota customizada com `rbac.RequireCapability(constants.Cap*)`.
- Migration com numeração sequencial, constante, blank import do pacote em `cmd/manager/main.go` e checklist em `docs/migrations.md`.
- Proto editado no `.proto` e regenerado com `make proto-gen`; nunca editar `proto/gen/*.pb.go`.
- Frontend com `npm`; `make build-web` antes de `make build-manager`.
- Sem comentários no código.
- Sem mudança fora do escopo, segredo, host pessoal, path local ou artefato de build.
- Commit em inglês, conventional, imperativo, sem co-autoria.
- Baseline de smells do Fowler como juízo, não violação dura: Mysterious Name, Duplicated Code, Feature Envy, Data Clumps, Primitive Obsession, Repeated Switches, Shotgun Surgery, Divergent Change, Speculative Generality, Message Chains, Middle Man, Refused Bequest.

### Escopo

- Se houver issue, feature ou plano aprovado: o diff implementa o que foi pedido? Aponte requisito faltando, pela metade, scope creep e implementação que parece certa mas está errada.
- Sem issue, feature ou plano disponível: pule o eixo e escreva `sem escopo declarado`; não invente requisito a partir do diff.
- Em correção de bug: a causa raiz está provada? Sem root cause, sem fix.

## Evidência

Toda afirmação sobre gate precisa de comando, saída e exit code. Sem isso, é alegação.

- `make verify` é o gate completo; `make verify-fast` é o de commit. No PR, o gate é o `quality` do CI.
- Não canalize teste para `tail`/`head`/`grep`; redirecione para arquivo e leia.
- Uma suíte por vez.
- Warning em saída de teste ou build é achado.
- Reviewer e refuter têm Bash amplo para leitura: git read-only, `gh`, `cat`, `rg`, pipes de leitura e comandos compostos. Mutações de git, suíte de teste, pipe de teste, flags que escrevem e comandos destrutivos são bloqueados pelo harness.
- No OpenCode, a deny-list de `permissions` faz o bloqueio; no Claude, o hook `PreToolUse` de `.claude/hooks/subagent-guard.mjs`, referenciado no frontmatter do agente, faz o mesmo.
- Falha pré-existente só com baseline: `git stash`, rodar, `git stash pop`, rodar, as duas saídas coladas.
- Saída sem alvo identificado é `INCONCLUSIVA`, não PASS.

## Formato do relatório

```text
## Standards
[Critical | Important | Minor] arquivo:linha
Uma frase com o defeito.
Cenário de falha: entrada ou estado concreto -> resultado errado.

## Escopo
(mesmo formato; ou "sem escopo declarado")

## Gates
Comando por comando: evidência colada, ausente ou baseline não provado.

## Veredito
GO ou NO-GO, com a razão em uma linha.
```

- O agente principal confirma cada achado Important lendo o código antes de corrigir.
- Minor não some: vai para o relatório e o usuário tria.
- Defeito que o plano mandou fazer é decisão do usuário, não do revisor.
- O revisor não edita, não cria arquivo, não muda branch e não roda a suíte inteira.

## Refutação adversarial

O `/review` despacha o `dockeeper-refuter` logo depois do reviewer. Ele recebe o relatório, busca o mesmo diff pela base e tenta derrubar cada achado. A refutação existe para tirar falso positivo e para dar segunda opinião ao `GO`; ela não substitui o reviewer. O refuter também cobre uma lente operacional que o reviewer não usa, e um achado novo dessa lente é dele.

### Regras

- O ônus da prova é de quem refuta. `REFUTED` exige `arquivo:linha`, teste com assert citado, trecho da spec/issue/plano ou comando com saída.
- `UPHELD` quando o cenário de falha é alcançável. `INCONCLUSIVO` quando a decisão exige execução; o parecer traz o comando exato que falta.
- Severidade não muda na refutação. `REFUTED` em Minor não apaga o achado do relatório; o usuário tria.
- Reviewer com `GO`: o refuter ataca as 2 ou 3 afirmações mais frágeis do relatório. Sem afirmações, ataca 2 ou 3 hipóteses de falha nas partes mais arriscadas do diff.
- O refuter não cria achado novo no diff do reviewer. A exceção é o defeito que derruba um `GO` e o achado da lente operacional, ambos com severidade e cenário.
- O refuter não edita, não cria arquivo, não muda branch e não roda a suíte inteira.

### Lente operacional

O que quebra em produção e não aparece na leitura estática:

- concorrência: EventBus single-goroutine, mutex, ordem de escrita, cache e dirty;
- goroutine e stream: watcher, canal sem dreno, reconexão, cancelamento de ctx;
- recurso: cache sem limite, conexão por ciclo, goroutine por reconexão;
- modo degradado: gate fechado, Docker ou manager fora, backoff;
- silêncio: erro que vira log e some, inspect que falha e não volta.

### Formato

```text
## Refutação
[Critical | Important | Minor] arquivo:linha
UPHELD | REFUTED | INCONCLUSIVO — prova em uma ou duas frases; em REFUTED, o arquivo:linha, teste ou trecho que derruba.

## GO
(apenas se o reviewer deu GO) afirmações atacadas e GO CONFIRMADO ou GO REFUTADO.

## Lente operacional
[Critical | Important | Minor] arquivo:linha e o cenário de falha; ou "nenhum achado".

## Veredito
NO-GO MANTIDO | NO-GO REFUTADO | GO CONFIRMADO | GO REFUTADO
```

### Como o principal aplica

- `UPHELD` e `INCONCLUSIVO` continuam valendo; Important só é corrigido depois de confirmação no código.
- `REFUTED` não é corrigido; fica registrado com a prova.
- `NO-GO REFUTADO` exige todos os Critical e Important refutados e nenhum achado da lente; vira `GO`.
- `GO REFUTADO` e achado Critical/Important da lente bloqueiam e vão para a triagem do usuário.

## Depois da correção

- Correção com prova própria (teste vermelho antes e verde depois) e gates verdes não pede review completo novo.
- Recheck só quando a correção toca concorrência, autorização, migration, proto ou contrato que a prova não alcança.
- Recheck é por achado: `ADDRESSED` ou `NOT ADDRESSED`, com a prova de cada um.
- Relatório e commit trazem o delta do que foi corrigido e a prova; não reaproveite o veredito anterior sem ele.
- Atualize o PR com commits novos; force-push em branch de PR é proibido.
