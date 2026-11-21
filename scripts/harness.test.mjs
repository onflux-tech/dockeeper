import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
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

test("readOnlyAgentErrors rejeita shell amplo e regra de edição", () => {
	const opencode = readFileSync(new URL("../harness/agents/dockeeper-reviewer/opencode.yaml", import.meta.url), "utf8");
	const broad = opencode.replace('resource: "go test *"', 'resource: "*"');
	assert.match(readOnlyAgentErrors("x", broad).join("\n"), /fora da allowlist/);
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

test("subagent-guard libera git read-only e go test focado", () => {
	assert.equal(evaluateSubagentShell("git status --short"), null);
	assert.equal(evaluateSubagentShell("git log --oneline -5"), null);
	assert.equal(evaluateSubagentShell("git diff 4cdcbc3...HEAD"), null);
	assert.equal(evaluateSubagentShell("git diff $(git merge-base develop HEAD)...HEAD"), null);
	assert.equal(evaluateSubagentShell("git diff --stat HEAD"), null);
	assert.equal(evaluateSubagentShell("go test ./internal/agent/collector/ -run 'TestDiscovery' -count=1"), null);
});

test("subagent-guard nega escrita, composto e teste amplo", () => {
	assert.match(evaluateSubagentShell("git stash"), /git de escrita/);
	assert.match(evaluateSubagentShell("git checkout develop"), /git de escrita/);
	assert.match(evaluateSubagentShell("git status && git stash"), /BLOQUEADO/);
	assert.match(evaluateSubagentShell("git diff > out.log"), /BLOQUEADO/);
	assert.match(evaluateSubagentShell("go test ./..."), /BLOQUEADO/);
	assert.match(evaluateSubagentShell("go test ./pkg"), /BLOQUEADO/);
	assert.match(evaluateSubagentShell("cat file"), /BLOQUEADO/);
});
