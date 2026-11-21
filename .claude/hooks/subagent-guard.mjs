// Gerado por make harness-sync a partir de harness/hooks/subagent-guard.mjs

#!/usr/bin/env node
import { readFileSync } from "node:fs";

const MUTATING_GIT =
	/\bgit\s+(?:stash|checkout|switch|reset|clean|restore|add|commit|apply|am|rebase|merge|cherry-pick|revert|tag|push|pull|fetch|worktree|submodule|rm|mv|update-index|config)(?:\s|$)/i;
const COMPOUND = /[;&|]|>>?/;
const READ_ONLY_GIT = /^git\s+(?:status|log|diff|show|merge-base|ls-files|blame|rev-parse)(?:\s|$)/i;
const FOCUSED_TEST = /^go\s+test\s+(?![^\n]*\.\.\.)(?=[^\n]*-run\s)/i;

export function evaluate(command) {
	const cmd = String(command || "").trim();
	if (!cmd) return "BLOQUEADO: comando vazio.";
	if (MUTATING_GIT.test(cmd)) {
		return "BLOQUEADO: git de escrita não é permitido no subagente read-only. Use status, log, diff, show, merge-base, ls-files, blame ou rev-parse.";
	}
	if (COMPOUND.test(cmd)) {
		return "BLOQUEADO: comando composto, pipe, redirecionamento ou utilitário de shell. Rode um comando git read-only por chamada; para ler use Read e para buscar use Grep.";
	}
	if (READ_ONLY_GIT.test(cmd) || FOCUSED_TEST.test(cmd)) return null;
	return "BLOQUEADO: o subagente só pode rodar git read-only ou go test focado com -run, sem ./... .";
}

if (process.argv[1] && String(process.argv[1]).replace(/\\/g, "/").endsWith("/subagent-guard.mjs")) {
	let raw = "";
	process.stdin.on("data", (chunk) => (raw += chunk));
	process.stdin.on("end", () => {
		let input;
		try {
			input = JSON.parse(raw);
		} catch {
			process.exit(0);
		}
		const message = evaluate((input.tool_input || {}).command || "");
		if (!message) process.exit(0);
		console.log(
			JSON.stringify({
				hookSpecificOutput: {
					hookEventName: "PreToolUse",
					permissionDecision: "deny",
					permissionDecisionReason: message,
				},
			}),
		);
		process.exit(0);
	});
}
