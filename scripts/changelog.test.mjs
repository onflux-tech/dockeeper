import test from "node:test";
import assert from "node:assert/strict";
import { checkEntries, isDate, isSemver, parseChangelog } from "./changelog.mjs";

const sample = `export const changelog = [
	{ version: "Unreleased", date: "2026-11-20", changes: [] },
	{ version: "0.8.0", date: "2026-04-06", changes: [] },
];`;

test("parseChangelog extrai versão e data", () => {
	assert.deepEqual(parseChangelog(sample), [
		{ version: "Unreleased", date: "2026-11-20" },
		{ version: "0.8.0", date: "2026-04-06" },
	]);
});

test("checkEntries aceita versão com tag", () => {
	assert.deepEqual(checkEntries(parseChangelog(sample), new Set(["v0.8.0"])), []);
});

test("checkEntries aceita a próxima versão sem tag", () => {
	const entries = [...parseChangelog(sample), { version: "0.9.0", date: "2026-11-21" }];
	assert.deepEqual(checkEntries(entries, new Set(["v0.8.0"])), []);
});

test("checkEntries recusa versão fantasma abaixo da próxima", () => {
	const entries = [
		{ version: "0.9.0", date: "2026-11-21" },
		{ version: "0.7.0", date: "2026-10-01" },
		{ version: "0.8.0", date: "2026-04-06" },
	];
	const errors = checkEntries(entries, new Set(["v0.8.0"]));
	assert.equal(errors.length, 1);
	assert.match(errors[0], /0\.7\.0: sem tag/);
});

test("checkEntries recusa duas versões sem tag", () => {
	const entries = [
		{ version: "0.10.0", date: "2026-12-01" },
		{ version: "0.9.0", date: "2026-11-21" },
		{ version: "0.8.0", date: "2026-04-06" },
	];
	const errors = checkEntries(entries, new Set(["v0.8.0"]));
	assert.equal(errors.length, 1);
	assert.match(errors[0], /0\.9\.0: sem tag/);
});

test("isSemver e isDate", () => {
	assert.equal(isSemver("0.8.0"), true);
	assert.equal(isSemver("v0.8.0"), false);
	assert.equal(isDate("2026-11-20"), true);
	assert.equal(isDate("TBD"), false);
});
