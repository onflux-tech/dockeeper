# Harness sources

- `skills/<nome>/SKILL.md`
- `commands/<nome>.md`
- `agents/<nome>/{body.md,claude.yaml,opencode.yaml}`

Todo agente em `agents/` é read-only: a porta OpenCode nega `*` e libera só `read`, `glob`, `grep`, `skill`, git read-only e `go test` focado. O check falha se a allowlist regredir para `*`, `edit` ou comando composto.

`make harness-sync` writes the ports in `.claude/` and `.opencode/`. `make harness-check` fails when a port drifts. `docs/harness.md` has the evolution rules.
