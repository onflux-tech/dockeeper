---
# Gerado por make harness-sync a partir de harness/commands/spec.md
description: Revisa uma spec e o plano com o dockeeper-spec-reviewer
argument-hint: <spec> [plano]
---

Despache o subagent `dockeeper-spec-reviewer` com a spec e o plano opcional.

1. Passe no prompt os paths da spec e do plano, o contrato do corpo do agente e "não edite".
2. Não cole o documento; o agente lê.
3. Apresente o relatório e espere a decisão do usuário; não edite a spec antes disso. Depois do aceite, aplique as correções aprovadas, rode o recheck focado por achado e só considere `PRONTO` na versão final.
4. Se a spec mudar depois de `PRONTO`, reabra o review. No plano, rode a mesma revisão antes de pedir aprovação ou modo de execução.
