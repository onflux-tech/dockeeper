import { evaluate } from "../../harness/hooks/subagent-guard.mjs";

const AGENTS = new Set(["dockeeper-reviewer", "dockeeper-refuter", "dockeeper-spec-reviewer", "dockeeper-ui-qa"]);

export default {
	id: "subagent-guard",
	async setup(ctx) {
		await ctx.tool.hook("execute.before", async (event) => {
			if (event.tool !== "shell") return;
			const input = event.input || {};
			let agent = event.agent;
			if (!agent) {
				const sessionID = event.sessionID || event.sessionId || event.session?.id;
				if (!sessionID) {
					throw new Error("BLOQUEADO: sessão ausente; o guard falha fechado.");
				}
				try {
					const session = await ctx.session.get({ sessionID });
					agent = session?.agent || session?.metadata?.agent;
				} catch (error) {
					console.error("subagent-guard: session lookup failed", error);
					throw new Error("BLOQUEADO: não foi possível confirmar o agente da sessão; o guard falha fechado.");
				}
			}
			if (!AGENTS.has(agent)) return;
			const message = evaluate(input.command || input.cmd || "");
			if (message) throw new Error(message);
		});
	},
};
