---
# Gerado por make harness-sync a partir de harness/commands/verify.md
description: Roda make verify (ou verify-fast) e reporta a saída real
argument-hint: [fast]
---

Rode o gate do dockeeper e reporte o resultado real.

1. Se `$ARGUMENTS` contiver `fast`, rode `make verify-fast`; caso contrário, `make verify`.
2. Redirecione para arquivo: `mkdir -p tmp && make verify > tmp/verify.log 2>&1; echo "exit=$?"`.
3. Leia `tmp/verify.log` e cole as linhas relevantes: exit code, `golangci-lint`, testes Go, ESLint, Prettier e Vitest.
4. Só declare "passou" com exit code 0 e a saída. Se falhar, mostre o erro e pare.
