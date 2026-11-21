import test from "node:test";
import assert from "node:assert/strict";
import {
	makeRefErrors,
	parseDocsIndex,
	parseFrontmatter,
	parseMakeTargets,
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
