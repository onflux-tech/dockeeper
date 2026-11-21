const MUTATING_GIT =
	/\bgit\s+(?:stash|checkout|switch|reset|clean|restore|add|commit|apply|am|rebase|merge|cherry-pick|revert|tag|push|pull|fetch|worktree|submodule|rm|mv|update-index|config|branch)(?:\s|$)/i;
const DESTRUCTIVE = [
	/\brm\s+/i,
	/\bRemove-Item\b/i,
	/\btaskkill\b/i,
	/\bpkill\b/i,
	/\bStop-Process\b/i,
	/--no-verify\b/i,
	/\bgit\s+push\b[^\n]*(?:--force(?:-with-lease)?|\s-f\b)/i,
];
const TEST_SUITE = [
	/\bgo\s+test\b(?![^\n]*-run\s)[^\n]*/i,
	/\bgo\s+test\b[^\n]*\.\.\./i,
	/\bnpm\s+(?:run\s+)?test\b/i,
	/\b(?:vitest|jest)\b/i,
	/\bmake\s+(?:verify(?:-fast)?|test|pre-push|release)\b/i,
];
const TEST_PIPE = /\b(?:go\s+test|npm\s+(?:run\s+)?test|vitest|jest)\b[^\n|]*\|\s*(?:tail|head|grep|less|more)\b/i;
const WRITE_FLAGS = [
	/\bgo\s+test\b[^\n]*\s-(?:o|exec|toolexec)\b/i,
	/\bgit\s+(?:diff|log|show)\b[^\n]*--output\b/i,
	/\bssh\b/i,
];

export function evaluate(command) {
	const cmd = String(command || "").trim();
	if (!cmd) return "BLOQUEADO: comando vazio.";
	if (MUTATING_GIT.test(cmd)) {
		return "BLOQUEADO: git de escrita não é permitido no subagente read-only. Use status, log, diff, show, merge-base, ls-files, blame ou rev-parse.";
	}
	for (const rule of DESTRUCTIVE) {
		if (rule.test(cmd)) {
			return "BLOQUEADO: comando destrutivo ou que pula hook. O subagente é read-only.";
		}
	}
	for (const rule of TEST_SUITE) {
		if (rule.test(cmd)) {
			return "BLOQUEADO: suíte de teste no subagente. Rode no máximo um teste focado com -run e sem ./....";
		}
	}
	if (TEST_PIPE.test(cmd)) {
		return "BLOQUEADO: nunca rode teste com pipe para tail/head/grep/less. Redirecione para arquivo e leia com Read.";
	}
	for (const rule of WRITE_FLAGS) {
		if (rule.test(cmd)) {
			return "BLOQUEADO: flag de escrita ou acesso remoto. O subagente busca dados por leitura local.";
		}
	}
	return null;
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
