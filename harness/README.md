# Harness sources

- `skills/<nome>/SKILL.md`
- `commands/<nome>.md`
- `agents/<nome>/{body.md,claude.yaml,opencode.yaml}`

`make harness-sync` writes the ports in `.claude/` and `.opencode/`. `make harness-check` fails when a port drifts. `docs/harness.md` has the evolution rules.
