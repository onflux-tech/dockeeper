#!/usr/bin/env node
import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const CHANGELOG = join(ROOT, "web/src/data/changelog.ts");
const ENTRY = /\{\s*version:\s*"([^"]+)",\s*date:\s*"([^"]+)"/g;
const SEMVER = /^\d+\.\d+\.\d+$/;
const DATE = /^\d{4}-\d{2}-\d{2}$/;

export function parseChangelog(text) {
	return [...text.matchAll(ENTRY)].map((match) => ({ version: match[1], date: match[2] }));
}

export function isSemver(version) {
	return SEMVER.test(version);
}

export function isDate(date) {
	return DATE.test(date);
}

export function checkEntries(entries, tags) {
	const errors = [];
	const seen = new Set();
	const pending = entries
		.filter((entry) => entry.version !== "Unreleased" && isSemver(entry.version) && !tags.has(`v${entry.version}`))
		.reduce((latest, entry) => (latest && latest.localeCompare(entry.version, undefined, { numeric: true }) >= 0 ? latest : entry.version), "");
	for (const entry of entries) {
		if (seen.has(entry.version)) errors.push(`versão duplicada: ${entry.version}`);
		seen.add(entry.version);
		if (!isDate(entry.date)) errors.push(`${entry.version}: data inválida (${entry.date})`);
		if (entry.version === "Unreleased") continue;
		if (!isSemver(entry.version)) {
			errors.push(`versão inválida: ${entry.version}`);
			continue;
		}
		if (!tags.has(`v${entry.version}`) && entry.version !== pending) {
			errors.push(`${entry.version}: sem tag v${entry.version}`);
		}
	}
	return errors;
}

function git(args) {
	return execFileSync("git", args, { cwd: ROOT, encoding: "utf8" }).trim();
}

function tags() {
	return new Set(git(["tag", "--list", "v*"]).split("\n").filter(Boolean));
}

function lastTag() {
	try {
		return git(["describe", "--tags", "--abbrev=0"]);
	} catch {
		return "";
	}
}

function draft() {
	const tag = lastTag();
	const range = tag ? `${tag}..HEAD` : "HEAD";
	console.log(`# Rascunho desde ${tag || "o início"}\n`);
	console.log(git(["log", "--oneline", range]));
}

function check(version) {
	const entries = parseChangelog(readFileSync(CHANGELOG, "utf8"));
	const tagSet = tags();
	if (!version) {
		const errors = checkEntries(entries, tagSet);
		if (errors.length) {
			for (const error of errors) console.error(`changelog: ${error}`);
			process.exitCode = 1;
			return;
		}
		console.log(`changelog: ok (${entries.length} entradas)`);
		return;
	}
	if (!isSemver(version)) {
		console.error("changelog: use VERSION=x.y.z");
		process.exitCode = 1;
		return;
	}
	const errors = [];
	if (tagSet.has(`v${version}`)) errors.push(`tag v${version} já existe`);
	const entry = entries.find((item) => item.version === version);
	if (!entry) errors.push(`changelog sem entrada para ${version}`);
	if (entry && !isDate(entry.date)) errors.push(`${version}: data inválida (${entry.date})`);
	if (git(["status", "--porcelain", "--untracked-files=no"])) {
		errors.push("working tree suja; commite o changelog antes da tag");
	}
	if (errors.length) {
		for (const error of errors) console.error(`changelog: ${error}`);
		process.exitCode = 1;
		return;
	}
	console.log(`changelog: ${version} pronto para tag`);
}

const [command, ...args] = process.argv.slice(2);
if (process.argv[1] && process.argv[1].replace(/\\/g, "/").endsWith("/scripts/changelog.mjs")) {
	if (command === "draft") {
		draft();
	} else if (command === "check") {
		const index = args.indexOf("--version");
		check(index >= 0 ? args[index + 1] : "");
	} else {
		console.error("uso: node scripts/changelog.mjs <draft|check> [--version x.y.z]");
		process.exitCode = 2;
	}
}
