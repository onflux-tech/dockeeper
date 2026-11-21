# Harness sources

- `skills/<nome>/SKILL.md`
- `commands/<nome>.md`
- `agents/<nome>/{body.md,claude.yaml,opencode.yaml}`
- `hooks/<nome>.mjs`

Todo agente em `agents/` é read-only: sem `Write`/`Edit`, Bash amplo para leitura e a função `hooks/subagent-guard.mjs` aplicada pelo hook `PreToolUse` no Claude e pelo plugin `.opencode/plugins/subagent-guard.js` no OpenCode. A função bloqueia git de escrita, suíte, pipe de teste, flags de escrita, destrutivos, `gh` de mutação, script inline e wrappers de shell. A guarda é best-effort; o read-only estrutural e o contrato do corpo são a barreira. O check falha se o shell amplo, o hook/plugin ou a regra de não editar regredir.

`make harness-sync` writes the ports in `.claude/` and `.opencode/`. `make harness-check` fails when a port drifts. `docs/harness.md` has the evolution rules.
