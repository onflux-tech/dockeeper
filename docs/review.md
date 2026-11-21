# Review — Dockeeper

Contrato de review pós-implementação. Vale para o agente principal, para o `dockeeper-reviewer` e para o `/review`.

## Quando

- Antes de push ou release, depois de `make verify`.
- Em mudança de código; docs, config e harness puros não pedem review completo.
- Em issue, feature ou refactor amplo, o review acontece antes de declarar pronto.

## Base do diff

- Working tree por padrão: `git status --short`, `git diff`, `git diff --cached` e untracked relevantes.
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

- `make verify` é o gate completo; `make verify-fast` é o de commit.
- Não canalize teste para `tail`/`head`/`grep`; redirecione para arquivo e leia.
- Uma suíte por vez.
- Warning em saída de teste ou build é achado.
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

## Depois da correção

- Correção com prova própria (teste vermelho antes e verde depois) e gates verdes não pede review completo novo.
- Recheck só quando a correção toca concorrência, autorização, migration, proto ou contrato que a prova não alcança.
- Recheck é por achado: `ADDRESSED` ou `NOT ADDRESSED`, com a prova de cada um.
- Relatório e commit trazem o delta do que foi corrigido e a prova; não reaproveite o veredito anterior sem ele.
