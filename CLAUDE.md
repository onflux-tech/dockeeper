@AGENTS.md

# Claude Code specifics

- Permissões compartilhadas em `.claude/settings.json`; o host de SSH staging fica só na configuração local, não versionada; `git push` exige confirmação.
- Harness: edite `harness/` e rode `make harness-sync`; não edite `.claude/` nem `.opencode/` à mão.
- Antes de qualquer push remoto, use a tool de ask (`AskUserQuestion`) e espere o sim explícito.
- Toda pergunta ao usuário usa `AskUserQuestion`, com a opção recomendada como primeira e marcada `(Recomendado)`.
- Detalhe de arquitetura, gotchas, release e e2e em `docs/`.
