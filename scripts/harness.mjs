#!/usr/bin/env node
import { execFileSync } from "node:child_process";
import {
	existsSync,
	mkdirSync,
	readFileSync,
	readdirSync,
	rmSync,
	statSync,
	writeFileSync,
} from "node:fs";
import { basename, dirname, join, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const HARNESS = join(ROOT, "harness");
const MARKER = "# Gerado por make harness-sync a partir de ";
const MARKER_JS = "// Gerado por make harness-sync a partir de ";
const MARKERS = [MARKER.trim(), MARKER_JS.trim()];
const FRONTMATTER = /^---\r?\n([\s\S]*?)\r?\n---\r?\n/;
const DOC_ROW = /^\| `([^`]+)` \|/gm;
const MAKE_TARGET = /^([A-Za-z0-9_.-]+):/gm;
const MAKE_REF = /\bmake ([a-z][a-z0-9-]*)\b/g;
const BACKTICK = /`([^`\n]+)`/g;
const FILE_EXT = /\.(?:go|md|json|jsonc|ya?ml|ts|tsx|js|jsx|mjs|cjs|proto|sh|css|html|mod|sum|toml)$/;
const REF_PREFIXES = [
	"docs/",
	"internal/",
	"cmd/",
	"migrations/",
	"proto/",
	"web/",
	".claude/",
	".opencode/",
	".githooks/",
	".github/",
	"harness/",
	"scripts/",
];
const REF_ROOTS = new Set(["AGENTS.md", "CLAUDE.md", "Makefile", ".mcp.json", ".gitignore", "go.mod", "frontend.go"]);
const REF_ALLOW = new Set([
	"web/dist",
	"web/node_modules",
	"bin",
	"dk_data",
	"dk-agent-data",
	"tmp",
	"dist",
	".claude/settings.local.json",
]);
const REF_ALLOW_PREFIXES = ["web/dist", "web/node_modules", "bin/", "dk_data/", "dk-agent-data/", "tmp/", "dist/"];
const PORT_DIRS = [
	".claude/skills",
	".claude/commands",
	".claude/agents",
	".claude/hooks",
	".opencode/skills",
	".opencode/commands",
	".opencode/agents",
];
const SECRETS = [
	{ re: /C:\\Users\\/i, name: "path pessoal do Windows" },
	{ re: /\/home\/(?!dockeeper\/)[A-Za-z0-9._-]+\//, name: "path pessoal do Linux" },
	{ re: /-----BEGIN (?:RSA |OPENSSH |EC )?PRIVATE KEY-----/, name: "chave privada" },
	{ re: /\bghp_[A-Za-z0-9]{20,}\b/, name: "token do GitHub" },
	{ re: /\bgithub_pat_[A-Za-z0-9_]{20,}\b/, name: "token do GitHub" },
	{ re: /\bAKIA[0-9A-Z]{16}\b/, name: "chave da AWS" },
];

const normalize = (text) => text.replace(/\r\n/g, "\n");
const read = (path) => normalize(readFileSync(path, "utf8"));
const rel = (path) => relative(ROOT, path).split(sep).join("/");

function walk(dir) {
	if (!existsSync(dir)) return [];
	const files = [];
	for (const entry of readdirSync(dir, { withFileTypes: true })) {
		const full = join(dir, entry.name);
		if (entry.isDirectory()) files.push(...walk(full));
		else if (entry.isFile()) files.push(full);
	}
	return files;
}

export function parseFrontmatter(text) {
	const normalized = normalize(text);
	const match = normalized.match(FRONTMATTER);
	if (!match) throw new Error("frontmatter ausente");
	return { fm: match[1], body: normalized.slice(match[0].length) };
}

export function field(fm, key) {
	const match = fm.match(new RegExp(`^${key}:\\s*(.*)$`, "m"));
	return match ? match[1].trim() : "";
}

export function validateDescription(fm, file) {
	const raw = field(fm, "description");
	if (!raw) return `${file}: description é obrigatória`;
	if (raw.includes(": ") && !/^["']/.test(raw)) return `${file}: description com ": " precisa de aspas`;
	return null;
}

const READ_ONLY_SHELL = [
	"go test *",
	"git status *",
	"git log *",
	"git diff *",
	"git show *",
	"git merge-base *",
	"git ls-files *",
	"git blame *",
	"git rev-parse *",
];

const READ_ONLY_TOOLS = ["Read", "Grep", "Glob", "Bash", "Skill"];

export function readOnlyAgentErrors(name, yaml) {
	const text = normalize(yaml).trim();
	const errors = [];
	if (!/^  - action: "\*"\n    resource: "\*"\n    effect: deny$/m.test(text)) {
		errors.push(`harness/agents/${name}/opencode.yaml: falta a regra inicial action "*" resource "*" effect deny`);
	}
	for (const action of ["read", "glob", "grep", "skill"]) {
		if (!new RegExp(`^  - action: ${action}$`, "m").test(text)) {
			errors.push(`harness/agents/${name}/opencode.yaml: falta allow para ${action}`);
		}
	}
	const shells = [...text.matchAll(/^  - action: shell\n    resource: "([^"]+)"\n    effect: allow$/gm)].map(
		(match) => match[1],
	);
	for (const resource of shells) {
		if (!READ_ONLY_SHELL.includes(resource)) {
			errors.push(`harness/agents/${name}/opencode.yaml: shell allow fora da allowlist: ${resource}`);
		}
	}
	for (const resource of READ_ONLY_SHELL) {
		if (!shells.includes(resource)) {
			errors.push(`harness/agents/${name}/opencode.yaml: falta shell allow: ${resource}`);
		}
	}
	if (/^  - action: (edit|write|subagent|webfetch|websearch)$/m.test(text)) {
		errors.push(`harness/agents/${name}/opencode.yaml: agente read-only não pode liberar edit/write/subagent/webfetch/websearch`);
	}
	return errors;
}

export function readOnlyClaudeErrors(name, yaml) {
	const tools = field(yaml, "tools")
		.split(",")
		.map((tool) => tool.trim())
		.filter(Boolean);
	const errors = [];
	for (const tool of READ_ONLY_TOOLS) {
		if (!tools.includes(tool)) errors.push(`harness/agents/${name}/claude.yaml: falta tool ${tool}`);
	}
	for (const tool of ["Write", "Edit", "Patch"]) {
		if (tools.includes(tool)) errors.push(`harness/agents/${name}/claude.yaml: agente read-only não pode ter ${tool}`);
	}
	if (!/^hooks:\n  PreToolUse:/m.test(yaml)) {
		errors.push(`harness/agents/${name}/claude.yaml: falta o hook PreToolUse escopado ao subagente`);
	}
	if (!yaml.includes(".claude/hooks/subagent-guard.mjs")) {
		errors.push(`harness/agents/${name}/claude.yaml: falta referência a .claude/hooks/subagent-guard.mjs`);
	}
	return errors;
}

export function renderPort(text, sourceRel) {
	const { fm, body } = parseFrontmatter(text);
	const cleanBody = body.replace(/^\n+/, "").replace(/\s+$/, "");
	return `---\n${MARKER}${sourceRel}\n${fm}\n---\n\n${cleanBody}\n`;
}

export function renderAgent(name, body, yaml, harness) {
	const head = harness === "claude" ? `name: ${name}\n` : "";
	const sourceRel = `harness/agents/${name}/${harness}.yaml`;
	return `---\n${MARKER}${sourceRel}\n${head}${yaml.trim()}\n---\n\n${body.trim()}\n`;
}

export function renderHook(text, sourceRel) {
	return `${MARKER_JS}${sourceRel}\n\n${normalize(text).trim()}\n`;
}

export function parseDocsIndex(text) {
	const start = text.indexOf("## Ponteiros");
	if (start < 0) return [];
	const end = text.indexOf("\n## ", start + 1);
	const section = text.slice(start, end < 0 ? undefined : end);
	return [...section.matchAll(DOC_ROW)].map((match) => match[1]);
}

export function parseMakeTargets(text) {
	const targets = new Set();
	for (const match of text.matchAll(MAKE_TARGET)) {
		if (!match[1].startsWith(".")) targets.add(match[1]);
	}
	const phony = text.match(/^\.PHONY:\s*(.*)$/m);
	if (phony) {
		for (const name of phony[1].split(/\s+/)) if (name) targets.add(name);
	}
	return targets;
}

export function makeRefErrors(text, targets, file) {
	const errors = [];
	for (const match of text.matchAll(MAKE_REF)) {
		if (!targets.has(match[1])) errors.push(`${file}: make ${match[1]} não existe`);
	}
	return errors;
}

export function sequenceErrors(numbers) {
	const sorted = [...numbers].sort((a, b) => a - b);
	for (let i = 0; i < sorted.length; i++) {
		if (sorted[i] !== i + 1) {
			return [`migrations: esperado ${String(i + 1).padStart(2, "0")}, encontrado ${String(sorted[i]).padStart(2, "0")}`];
		}
	}
	return [];
}

function readSources() {
	const sources = [];

	for (const entry of readdirSync(join(HARNESS, "skills"), { withFileTypes: true })) {
		if (!entry.isDirectory()) continue;
		const name = entry.name;
		const file = join(HARNESS, "skills", name, "SKILL.md");
		if (!existsSync(file)) throw new Error(`harness/skills/${name}: SKILL.md ausente`);
		const text = read(file);
		const { fm } = parseFrontmatter(text);
		const error = validateDescription(fm, `harness/skills/${name}/SKILL.md`);
		if (error) throw new Error(error);
		if (field(fm, "name") !== name) throw new Error(`harness/skills/${name}: name deve ser ${name}`);
		sources.push({ kind: "skill", name, text });
	}

	for (const entry of readdirSync(join(HARNESS, "commands"), { withFileTypes: true })) {
		if (!entry.isFile() || !entry.name.endsWith(".md")) continue;
		const name = basename(entry.name, ".md");
		const text = read(join(HARNESS, "commands", entry.name));
		const { fm } = parseFrontmatter(text);
		const error = validateDescription(fm, `harness/commands/${entry.name}`);
		if (error) throw new Error(error);
		sources.push({ kind: "command", name, text });
	}

	for (const entry of readdirSync(join(HARNESS, "agents"), { withFileTypes: true })) {
		if (!entry.isDirectory()) continue;
		const name = entry.name;
		const base = join(HARNESS, "agents", name);
		const bodyFile = join(base, "body.md");
		const claudeFile = join(base, "claude.yaml");
		const opencodeFile = join(base, "opencode.yaml");
		for (const file of [bodyFile, claudeFile, opencodeFile]) {
			if (!existsSync(file)) throw new Error(`${rel(file)} ausente`);
		}
		const body = read(bodyFile).trim();
		if (!body) throw new Error(`${rel(bodyFile)} vazio`);
		const claude = read(claudeFile).trim();
		const opencode = read(opencodeFile).trim();
		const claudeError = validateDescription(claude, rel(claudeFile));
		const opencodeError = validateDescription(opencode, rel(opencodeFile));
		if (claudeError) throw new Error(claudeError);
		if (opencodeError) throw new Error(opencodeError);
		if (field(opencode, "mode") !== "subagent") throw new Error(`${rel(opencodeFile)}: mode deve ser subagent`);
		const agentErrors = [...readOnlyAgentErrors(name, opencode), ...readOnlyClaudeErrors(name, claude)];
		if (agentErrors.length) throw new Error(agentErrors.join("; "));
		sources.push({ kind: "agent", name, body, claude, opencode });
	}

	const hooksDir = join(HARNESS, "hooks");
	if (existsSync(hooksDir)) {
		for (const entry of readdirSync(hooksDir, { withFileTypes: true })) {
			if (!entry.isFile() || !entry.name.endsWith(".mjs")) continue;
			const name = basename(entry.name, ".mjs");
			const text = read(join(hooksDir, entry.name));
			if (!text.trim()) throw new Error(`harness/hooks/${entry.name}: vazio`);
			sources.push({ kind: "hook", name, text });
		}
	}

	return sources;
}

function expectedPorts(sources) {
	const ports = new Map();
	for (const source of sources) {
		if (source.kind === "skill") {
			const content = renderPort(source.text, `harness/skills/${source.name}/SKILL.md`);
			ports.set(`.claude/skills/${source.name}/SKILL.md`, content);
			ports.set(`.opencode/skills/${source.name}/SKILL.md`, content);
		} else if (source.kind === "command") {
			const content = renderPort(source.text, `harness/commands/${source.name}.md`);
			ports.set(`.claude/commands/${source.name}.md`, content);
			ports.set(`.opencode/commands/${source.name}.md`, content);
		} else if (source.kind === "hook") {
			ports.set(`.claude/hooks/${source.name}.mjs`, renderHook(source.text, `harness/hooks/${source.name}.mjs`));
		} else {
			ports.set(`.claude/agents/${source.name}.md`, renderAgent(source.name, source.body, source.claude, "claude"));
			ports.set(`.opencode/agents/${source.name}.md`, renderAgent(source.name, source.body, source.opencode, "opencode"));
		}
	}
	return ports;
}

function syncPorts(ports) {
	for (const [path, content] of ports) {
		const full = join(ROOT, path);
		mkdirSync(dirname(full), { recursive: true });
		if (!existsSync(full) || read(full) !== content) writeFileSync(full, content);
	}
	for (const dir of PORT_DIRS) {
		for (const file of walk(join(ROOT, dir))) {
			if (!file.endsWith(".md") && !file.endsWith(".mjs")) continue;
			if (ports.has(rel(file))) continue;
			if (read(file).split("\n").slice(0, 10).some((line) => MARKERS.some((marker) => line.startsWith(marker)))) {
				rmSync(file);
			}
		}
	}
	return ports.size;
}

function checkPorts(ports) {
	const errors = [];
	for (const [path, content] of ports) {
		const full = join(ROOT, path);
		if (!existsSync(full)) errors.push(`${path}: porta ausente; rode make harness-sync`);
		else if (read(full) !== content) errors.push(`${path}: porta divergente; rode make harness-sync`);
	}
	for (const dir of PORT_DIRS) {
		for (const file of walk(join(ROOT, dir))) {
			if (!file.endsWith(".md") && !file.endsWith(".mjs")) continue;
			if (ports.has(rel(file))) continue;
			if (read(file).split("\n").slice(0, 10).some((line) => MARKERS.some((marker) => line.startsWith(marker)))) {
				errors.push(`${rel(file)}: porta gerada órfã; rode make harness-sync`);
			}
		}
	}
	return errors;
}

function checkDocsIndex() {
	const errors = [];
	const listed = parseDocsIndex(read(join(ROOT, "AGENTS.md")));
	const docs = existsSync(join(ROOT, "docs"))
		? readdirSync(join(ROOT, "docs")).filter((file) => file.endsWith(".md")).map((file) => `docs/${file}`)
		: [];
	for (const doc of docs) if (!listed.includes(doc)) errors.push(`${doc}: fora do índice do AGENTS.md`);
	for (const path of listed) if (!existsSync(join(ROOT, path))) errors.push(`AGENTS.md: ${path} não existe`);
	return errors;
}

function checkMakeRefs() {
	const targets = parseMakeTargets(read(join(ROOT, "Makefile")));
	const files = [
		join(ROOT, "AGENTS.md"),
		join(ROOT, "CLAUDE.md"),
		...walk(join(ROOT, "docs")).filter((file) => file.endsWith(".md")),
		...walk(join(ROOT, "harness")).filter((file) => file.endsWith(".md")),
	];
	return files.flatMap((file) => makeRefErrors(read(file), targets, rel(file)));
}

function fileRefErrors(text, file) {
	const errors = [];
	for (const match of text.matchAll(BACKTICK)) {
		const token = match[1].trim();
		if (!token || /[\s*?<>{}()|$~#]/.test(token) || token.includes("[") || token.includes("]") || token.includes("...") || token.includes("NN")) continue;
		const [path, line] = token.split(":");
		if (REF_ALLOW.has(token) || REF_ALLOW.has(path)) continue;
		if (REF_ALLOW_PREFIXES.some((prefix) => token.startsWith(prefix))) continue;
		const hasPrefix = REF_PREFIXES.some((prefix) => path.startsWith(prefix));
		if (!hasPrefix && !REF_ROOTS.has(path)) continue;
		if (!path.endsWith("/") && !FILE_EXT.test(path) && !REF_ROOTS.has(path)) continue;
		const full = join(ROOT, path);
		if (!existsSync(full)) {
			errors.push(`${file}: ${token} não existe`);
			continue;
		}
		if (line && /^\d+(?:-\d+)?(?:,\d+)*$/.test(line) && statSync(full).isFile()) {
			const first = Number(line.split(/[-,]/)[0]);
			if (first > read(full).split("\n").length) errors.push(`${file}: ${token} aponta para linha inexistente`);
		}
	}
	return errors;
}

function checkFileRefs() {
	const files = [
		join(ROOT, "AGENTS.md"),
		join(ROOT, "CLAUDE.md"),
		...walk(join(ROOT, "docs")).filter((file) => file.endsWith(".md")),
		...walk(join(ROOT, "harness")).filter((file) => file.endsWith(".md")),
	];
	return files.flatMap((file) => fileRefErrors(read(file), rel(file)));
}

function checkMigrations() {
	const errors = [];
	const files = readdirSync(join(ROOT, "migrations")).filter((file) => /^\d{2}_.*\.go$/.test(file));
	errors.push(...sequenceErrors(files.map((file) => Number(file.slice(0, 2)))));
	if (!read(join(ROOT, "cmd/manager/main.go")).includes('_ "github.com/onflux-tech/dockeeper/migrations"')) {
		errors.push("cmd/manager/main.go: blank import de migrations ausente");
	}
	const constants = read(join(ROOT, "internal/shared/constants/collections.go"));
	const names = new Set([...constants.matchAll(/\bColl\w+\s*=\s*"([^"]+)"/g)].map((match) => match[1]));
	for (const file of files) {
		for (const match of read(join(ROOT, "migrations", file)).matchAll(/New(?:Base|Auth)Collection\("([^"]+)"\)/g)) {
			if (!names.has(match[1])) errors.push(`migrations/${file}: collection ${match[1]} sem constante Coll*`);
		}
	}
	return errors;
}

function checkSensitive() {
	const files = [
		join(ROOT, "AGENTS.md"),
		join(ROOT, "CLAUDE.md"),
		join(ROOT, ".mcp.json"),
		join(ROOT, ".claude/settings.json"),
		join(ROOT, ".opencode/opencode.json"),
		...walk(join(ROOT, "docs")),
		...walk(join(ROOT, "harness")),
	].filter((file) => file.endsWith(".md") || file.endsWith(".json"));
	const errors = [];
	for (const file of files) {
		const text = read(file);
		for (const secret of SECRETS) if (secret.re.test(text)) errors.push(`${rel(file)}: ${secret.name}`);
	}
	return errors;
}

function checkGitignore(ports) {
	const errors = [];
	for (const path of ports.keys()) {
		try {
			execFileSync("git", ["check-ignore", "-q", path], { cwd: ROOT, stdio: "ignore" });
			errors.push(`${path}: ignorado pelo .gitignore; ajuste a exceção`);
		} catch (error) {
			if (error.status !== 1) errors.push(`${path}: git check-ignore falhou`);
		}
	}
	return errors;
}

export function main(argv = process.argv.slice(2)) {
	try {
		const sources = readSources();
		const ports = expectedPorts(sources);
		if (argv[0] === "sync") {
			console.log(`harness: ${syncPorts(ports)} portas sincronizadas`);
			return 0;
		}
		if (argv[0] === "check") {
			const errors = [
				...checkPorts(ports),
				...checkDocsIndex(),
				...checkMakeRefs(),
				...checkFileRefs(),
				...checkMigrations(),
				...checkSensitive(),
				...checkGitignore(ports),
			];
			if (errors.length) {
				for (const error of errors) console.error(`harness: ${error}`);
				return 1;
			}
			console.log("harness: ok");
			return 0;
		}
		console.error("uso: node scripts/harness.mjs <sync|check>");
		return 2;
	} catch (error) {
		console.error(`harness: ${error.message}`);
		return 1;
	}
}

if (process.argv[1] && process.argv[1].replace(/\\/g, "/").endsWith("/scripts/harness.mjs")) {
	process.exitCode = main();
}
