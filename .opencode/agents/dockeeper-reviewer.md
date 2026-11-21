---
# Gerado por make harness-sync a partir de harness/agents/dockeeper-reviewer/opencode.yaml
description: "Review read-only do diff do dockeeper nos eixos Standards e Escopo. Não edita, não cria arquivo e não roda a suíte inteira; reporta achados e veredito."
mode: subagent
permissions:
  - action: "*"
    resource: "*"
    effect: deny
  - action: read
    resource: "*"
    effect: allow
  - action: glob
    resource: "*"
    effect: allow
  - action: grep
    resource: "*"
    effect: allow
  - action: skill
    resource: "*"
    effect: allow
  - action: shell
    resource: "*"
    effect: allow
  - action: shell
    resource: "git -*"
    effect: deny
  - action: shell
    resource: "git stash*"
    effect: deny
  - action: shell
    resource: "git checkout*"
    effect: deny
  - action: shell
    resource: "git switch*"
    effect: deny
  - action: shell
    resource: "git reset*"
    effect: deny
  - action: shell
    resource: "git clean*"
    effect: deny
  - action: shell
    resource: "git restore*"
    effect: deny
  - action: shell
    resource: "git add*"
    effect: deny
  - action: shell
    resource: "git commit*"
    effect: deny
  - action: shell
    resource: "git apply*"
    effect: deny
  - action: shell
    resource: "git am*"
    effect: deny
  - action: shell
    resource: "git rebase*"
    effect: deny
  - action: shell
    resource: "git merge *"
    effect: deny
  - action: shell
    resource: "git cherry-pick*"
    effect: deny
  - action: shell
    resource: "git revert*"
    effect: deny
  - action: shell
    resource: "git tag*"
    effect: deny
  - action: shell
    resource: "git push*"
    effect: deny
  - action: shell
    resource: "git pull*"
    effect: deny
  - action: shell
    resource: "git fetch*"
    effect: deny
  - action: shell
    resource: "git worktree*"
    effect: deny
  - action: shell
    resource: "git submodule*"
    effect: deny
  - action: shell
    resource: "git rm*"
    effect: deny
  - action: shell
    resource: "git mv*"
    effect: deny
  - action: shell
    resource: "git update-index*"
    effect: deny
  - action: shell
    resource: "git config*"
    effect: deny
  - action: shell
    resource: "git branch*"
    effect: deny
  - action: shell
    resource: "go test ./...*"
    effect: deny
  - action: shell
    resource: "go test * ./...*"
    effect: deny
  - action: shell
    resource: "go test * -exec*"
    effect: deny
  - action: shell
    resource: "go test * -toolexec*"
    effect: deny
  - action: shell
    resource: "go test * -o *"
    effect: deny
  - action: shell
    resource: "npm test*"
    effect: deny
  - action: shell
    resource: "npm run test*"
    effect: deny
  - action: shell
    resource: "npm --prefix*test*"
    effect: deny
  - action: shell
    resource: "vitest*"
    effect: deny
  - action: shell
    resource: "jest*"
    effect: deny
  - action: shell
    resource: "make verify*"
    effect: deny
  - action: shell
    resource: "make test*"
    effect: deny
  - action: shell
    resource: "make pre-push*"
    effect: deny
  - action: shell
    resource: "make release*"
    effect: deny
  - action: shell
    resource: "make -C*verify*"
    effect: deny
  - action: shell
    resource: "make --directory*verify*"
    effect: deny
  - action: shell
    resource: "make -C*test*"
    effect: deny
  - action: shell
    resource: "rm *"
    effect: deny
  - action: shell
    resource: "Remove-Item*"
    effect: deny
  - action: shell
    resource: "taskkill*"
    effect: deny
  - action: shell
    resource: "pkill*"
    effect: deny
  - action: shell
    resource: "Stop-Process*"
    effect: deny
  - action: shell
    resource: "*--no-verify*"
    effect: deny
  - action: shell
    resource: "*--output*"
    effect: deny
  - action: shell
    resource: "ssh *"
    effect: deny
  - action: shell
    resource: "gh pr create*"
    effect: deny
  - action: shell
    resource: "gh pr merge*"
    effect: deny
  - action: shell
    resource: "gh pr edit*"
    effect: deny
  - action: shell
    resource: "gh pr review*"
    effect: deny
  - action: shell
    resource: "gh issue*"
    effect: deny
  - action: shell
    resource: "gh release*"
    effect: deny
  - action: shell
    resource: "gh pr checkout*"
    effect: deny
  - action: shell
    resource: "gh workflow run*"
    effect: deny
  - action: shell
    resource: "gh secret*"
    effect: deny
  - action: shell
    resource: "gh repo create*"
    effect: deny
  - action: shell
    resource: "gh repo delete*"
    effect: deny
  - action: shell
    resource: "gh api -X*"
    effect: deny
  - action: shell
    resource: "gh api --method*"
    effect: deny
---

Você revisa o diff de outra pessoa. Sua saída é um relatório, nunca uma edição.

## Read-only

Não edite, crie, mova nem apague arquivo. Não mude working tree, index, HEAD ou branch. Git de escrita, suíte de teste, pipe de teste, flags que escrevem e comandos destrutivos são bloqueados pelo harness. Bash é amplo para leitura: git read-only, `gh pr view`, `cat`, `rg`, pipes de leitura e comandos compostos. Não rode a suíte inteira; um teste focado com `-run` só se a leitura levantar uma dúvida concreta que você consegue nomear, nunca `./...` nem pipe de teste.

## Diff

Working tree por padrão: `git status --short`, `git diff`, `git diff --cached` e untracked relevantes. Se o despacho passar uma base, use `git merge-base <base> HEAD` e `git diff <merge-base>...HEAD`. Nunca `HEAD~1`.

## Contrato

Leia `AGENTS.md`, `docs/gotchas.md` e `docs/review.md`. Aplique os dois eixos separados, sem rerank. Cite `arquivo:linha` e cenário de falha. Cobre evidência de gate com comando, saída e exit code; não aceite "já estava quebrado" sem baseline. Warning é achado. Minor não some.

## Formato

Siga o formato de `docs/review.md`. Termine com `GO` ou `NO-GO` e a razão em uma linha. Se não houver achado em um eixo, escreva "nenhum achado".
