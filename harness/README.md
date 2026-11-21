# Harness sources

- `skills/<nome>/SKILL.md`
- `commands/<nome>.md`
- `agents/<nome>/{body.md,claude.yaml,opencode.yaml}`
- `hooks/<nome>.mjs`

Todo agente em `agents/` é read-only: sem `Write`/`Edit`, Bash amplo para leitura e uma deny-list de mutação de git, suíte de teste, flags de escrita e comandos destrutivos. No OpenCode, as `permissions` aplicam a deny-list; no Claude, o hook `PreToolUse` de `hooks/subagent-guard.mjs` faz o mesmo. O check falha se a deny-list, o shell amplo ou o hook regredir.

`make harness-sync` writes the ports in `.claude/` and `.opencode/`. `make harness-check` fails when a port drifts. `docs/harness.md` has the evolution rules.
