# Harness sources

- `skills/<nome>/SKILL.md`
- `commands/<nome>.md`
- `agents/<nome>/{body.md,claude.yaml,opencode.yaml}`
- `hooks/<nome>.mjs`

Todo agente em `agents/` é read-only: a porta OpenCode nega `*` e libera só `read`, `glob`, `grep`, `skill`, git read-only e `go test` focado; a porta Claude recebe o hook `PreToolUse` de `hooks/subagent-guard.mjs`, que nega git de escrita, comando composto, pipe e teste fora de `-run`. O check falha se a allowlist ou o hook regredir.

`make harness-sync` writes the ports in `.claude/` and `.opencode/`. `make harness-check` fails when a port drifts. `docs/harness.md` has the evolution rules.
