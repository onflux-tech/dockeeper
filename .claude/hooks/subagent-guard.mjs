// Gerado por make harness-sync a partir de harness/hooks/subagent-guard.mjs

const MUTATING_SUBCOMMANDS = new Set([
	"stash",
	"checkout",
	"switch",
	"reset",
	"clean",
	"restore",
	"add",
	"commit",
	"apply",
	"am",
	"rebase",
	"merge",
	"cherry-pick",
	"revert",
	"tag",
	"push",
	"pull",
	"fetch",
	"worktree",
	"submodule",
	"rm",
	"mv",
	"update-index",
	"config",
	"branch",
]);

const GIT_COMMAND = /(?:^|[;&|(\n]|`)\s*(?:[A-Za-z_][A-Za-z0-9_]*=\S+\s+)*git\b([^\n;&|`]*)/g;

const DESTRUCTIVE = [
	/(?:^|[;&|(\n]|`)\s*rm\s+/i,
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
	/\bnpm\b[^\n]*(?:run\s+)?test\b/i,
	/\b(?:vitest|jest)\b/i,
	/\bmake\b[^\n]*(?:verify(?:-fast)?|test|pre-push|release)\b/i,
];

const TEST_PIPE = /\b(?:go\s+test|npm|vitest|jest|make)\b[^\n|]*\|\s*(?:tail|head|grep|less|more)\b/i;

const GH_MUTATION = [
	/\bgh\s+pr\s+(?:create|merge|edit|review|checkout|close|reopen)\b/i,
	/\bgh\s+issue\s+(?:create|close|reopen|comment|edit|delete)\b/i,
	/\bgh\s+release\s+(?:create|delete|edit|upload)\b/i,
	/\bgh\s+repo\s+(?:create|delete|edit|archive|rename|transfer)\b/i,
	/\bgh\s+workflow\s+run\b/i,
	/\bgh\s+secret\s+(?:set|delete)\b/i,
	/\bgh\s+api\b[^\n]*(?:-X\s*|--method[=\s]+)(?:POST|PUT|PATCH|DELETE)\b/i,
];

const WRITE_FLAGS = [
	/\bgo\s+test\b[^\n]*\s-(?:o|exec|toolexec)\b/i,
	/\bgit\s+(?:diff|log|show)\b[^\n]*--output\b/i,
	/\bssh\b/i,
];

function tokenize(args) {
	const tokens = [];
	let current = "";
	let quote = null;
	for (const char of args) {
		if (quote) {
			if (char === quote) quote = null;
			else current += char;
			continue;
		}
		if (char === '"' || char === "'") {
			quote = char;
			continue;
		}
		if (/\s/.test(char)) {
			if (current) {
				tokens.push(current);
				current = "";
			}
			continue;
		}
		current += char;
	}
	if (current) tokens.push(current);
	return tokens;
}

function hasMutatingGit(command) {
	for (const match of command.matchAll(GIT_COMMAND)) {
		const tokens = tokenize(match[1]);
		let index = 0;
		while (index < tokens.length && tokens[index].startsWith("-")) {
			const token = tokens[index];
			if (token === "-C" || token === "-c") index += 2;
			else if (/^--(?:git-dir|work-tree|namespace|exec-path|config-env)(?:=|$)/.test(token)) {
				index += token.includes("=") ? 1 : 2;
			} else index += 1;
		}
		if (MUTATING_SUBCOMMANDS.has((tokens[index] || "").toLowerCase())) return true;
	}
	return false;
}

export function evaluate(command) {
	const cmd = String(command || "").replace(/\\\r?\n/g, " ").trim();
	if (!cmd) return "BLOQUEADO: comando vazio.";
	if (hasMutatingGit(cmd)) {
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
	for (const rule of GH_MUTATION) {
		if (rule.test(cmd)) {
			return "BLOQUEADO: gh de mutação no subagente read-only. Use gh pr view ou consulte o principal.";
		}
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
