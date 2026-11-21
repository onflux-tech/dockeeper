import test from "node:test";
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { evaluate as evaluateSubagentShell } from "../harness/hooks/subagent-guard.mjs";
import {
	makeRefErrors,
	parseDocsIndex,
	parseFrontmatter,
	parseMakeTargets,
	readOnlyAgentErrors,
	readOnlyClaudeErrors,
	renderAgent,
	renderPort,
	sequenceErrors,
	validateDescription,
} from "./harness.mjs";

test("parseFrontmatter aceita CRLF", () => {
	const parsed = parseFrontmatter('---\r\nname: x\r\ndescription: "a: b"\r\n---\r\n\r\nbody\r\n');
	assert.equal(parsed.fm.includes("name: x"), true);
	assert.equal(parsed.body.trim(), "body");
});

test("validateDescription exige aspas quando há dois-pontos", () => {
	assert.match(validateDescription("description: Use: x", "x.md"), /aspas/);
	assert.equal(validateDescription('description: "Use: x"', "x.md"), null);
});

test("parseDocsIndex extrai só a tabela de Ponteiros", () => {
	const text =
		"## Ponteiros\n\n| Arquivo | O que tem |\n|---|---|\n| `docs/a.md` | a |\n| `CLAUDE.md` | b |\n\n## Outro\n| `docs/b.md` | c |";
	assert.deepEqual(parseDocsIndex(text), ["docs/a.md", "CLAUDE.md"]);
});

test("makeRefErrors aponta target inexistente", () => {
	const targets = parseMakeTargets("verify:\n\t@echo ok\n.PHONY: verify");
	assert.deepEqual(makeRefErrors("rode `make verify`", targets, "AGENTS.md"), []);
	assert.deepEqual(makeRefErrors("rode `make missing`", targets, "AGENTS.md"), ["AGENTS.md: make missing não existe"]);
});

test("sequenceErrors aponta lacuna", () => {
	assert.deepEqual(sequenceErrors([1, 2, 3]), []);
	assert.equal(sequenceErrors([1, 2, 4]).length, 1);
});

test("renderPort insere o marcador", () => {
	const out = renderPort("---\nname: x\ndescription: y\n---\n\nbody\n", "harness/skills/x/SKILL.md");
	assert.match(out, /# Gerado por make harness-sync/);
	assert.match(out, /name: x/);
	assert.match(out, /body/);
});

test("renderAgent separa claude e opencode", () => {
	const claude = renderAgent("r", "body", 'description: "d"\ntools: Read', "claude");
	assert.match(claude, /name: r/);
	assert.match(claude, /tools: Read/);
	const opencode = renderAgent("r", "body", 'description: "d"\nmode: subagent', "opencode");
	assert.doesNotMatch(opencode, /name: r/);
	assert.match(opencode, /mode: subagent/);
});

test("agentes reais passam no contrato read-only", () => {
	for (const name of ["dockeeper-reviewer", "dockeeper-refuter"]) {
		const opencode = readFileSync(new URL(`../harness/agents/${name}/opencode.yaml`, import.meta.url), "utf8");
		const claude = readFileSync(new URL(`../harness/agents/${name}/claude.yaml`, import.meta.url), "utf8");
		assert.deepEqual(readOnlyAgentErrors(name, opencode), []);
		assert.deepEqual(readOnlyClaudeErrors(name, claude), []);
	}
});

test("readOnlyAgentErrors exige shell amplo, deny-list e nenhuma edição", () => {
	const opencode = readFileSync(new URL("../harness/agents/dockeeper-reviewer/opencode.yaml", import.meta.url), "utf8");
	const noStash = opencode.replace('  - action: shell\n    resource: "git stash*"\n    effect: deny\n', "");
	assert.match(readOnlyAgentErrors("x", noStash).join("\n"), /falta deny: git stash/);
	const noBroad = opencode.replace('  - action: shell\n    resource: "*"\n    effect: allow\n', "");
	assert.match(readOnlyAgentErrors("x", noBroad).join("\n"), /falta shell amplo/);
	const edit = `${opencode}\n  - action: edit\n    resource: "*"\n    effect: allow\n`;
	assert.match(readOnlyAgentErrors("x", edit).join("\n"), /não pode liberar edit/);
});

test("readOnlyClaudeErrors exige Skill e rejeita Write", () => {
	const claude = readFileSync(new URL("../harness/agents/dockeeper-reviewer/claude.yaml", import.meta.url), "utf8");
	const withoutSkill = claude.replace(", Skill", "");
	assert.match(readOnlyClaudeErrors("x", withoutSkill).join("\n"), /falta tool Skill/);
	const write = claude.replace("Bash, Skill", "Bash, Skill, Write");
	assert.match(readOnlyClaudeErrors("x", write).join("\n"), /não pode ter Write/);
});

test("subagent-guard libera leitura ampla e teste focado", () => {
	assert.equal(evaluateSubagentShell("git status --short"), null);
	assert.equal(evaluateSubagentShell("git -C . status --short"), null);
	assert.equal(evaluateSubagentShell("git diff $(git merge-base develop HEAD)...HEAD | head"), null);
	assert.equal(evaluateSubagentShell("cat AGENTS.md"), null);
	assert.equal(evaluateSubagentShell("gh pr view 12 --json body"), null);
	assert.equal(evaluateSubagentShell("git status && git diff --stat"), null);
	assert.equal(evaluateSubagentShell("go test ./internal/agent/collector/ -run 'TestDiscovery' -count=1"), null);
});

test("subagent-guard nega mutação, suíte, pipe de teste e destrutivo", () => {
	assert.match(evaluateSubagentShell("git stash"), /git de escrita/);
	assert.match(evaluateSubagentShell("git status\ngit branch -D main"), /git de escrita/);
	assert.match(evaluateSubagentShell("git status\nrm -rf dk_data"), /destrutivo/);
	assert.match(evaluateSubagentShell("go test ./pkg -run X\nrm -rf"), /destrutivo/);
	assert.match(evaluateSubagentShell("go test ./..."), /suíte/);
	assert.match(evaluateSubagentShell("go test ./pkg"), /suíte/);
	assert.match(evaluateSubagentShell("go test ./pkg -run X | head"), /pipe/);
	assert.match(evaluateSubagentShell("npm test | tail"), /suíte/);
	assert.match(evaluateSubagentShell("go test ./pkg -exec 'rm' -run X"), /flag de escrita/);
	assert.match(evaluateSubagentShell("git diff --output=out"), /flag de escrita/);
	assert.match(evaluateSubagentShell("ssh host"), /flag de escrita|remoto/);
	assert.match(evaluateSubagentShell("git push"), /git de escrita/);
	assert.match(evaluateSubagentShell("--no-verify"), /destrutivo/);
});

test("porta gerada do subagent-guard é JS válido", () => {
	const port = fileURLToPath(new URL("../.claude/hooks/subagent-guard.mjs", import.meta.url));
	execFileSync(process.execPath, ["--check", port], { stdio: "ignore" });
});
