import { evaluate } from "../../harness/hooks/subagent-guard.mjs";

const AGENTS = new Set(["dockeeper-reviewer", "dockeeper-refuter", "dockeeper-spec-reviewer", "dockeeper-ui-qa"]);

export default {
	id: "subagent-guard",
	async setup(ctx) {
		await ctx.tool.hook("execute.before", async (event) => {
			if (event.tool !== "shell") return;
			const sessionID = event.sessionID || event.sessionId || event.session?.id;
			if (!sessionID) return;
			let session;
			try {
				session = await ctx.session.get({ sessionID });
			} catch (error) {
				console.error("subagent-guard: session lookup failed", error);
				return;
			}
			const agent = event.agent || session?.agent || session?.metadata?.agent;
			if (!AGENTS.has(agent)) return;
			const input = event.input || {};
			const message = evaluate(input.command || input.cmd || "");
			if (message) throw new Error(message);
		});
	},
};
