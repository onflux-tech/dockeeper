import type { ReactNode } from "react";

// ── Token color classes ─────────────────────────────────────────────────────

const C_KEY = "text-blue-400";
const C_STRING = "text-green-400";
const C_NUMBER = "text-amber-400";
const C_BOOL = "text-purple-400";
const C_NULL = "text-red-400";
const C_BRACE = "text-muted-foreground";

// ── JSON highlighter ────────────────────────────────────────────────────────

const JSON_REGEX =
	/("(?:[^"\\]|\\.)*")(\s*:)?|(\b(?:true|false)\b)|\b(null)\b|(-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?)\b|([{}[\],])/g;

let keyCounter = 0;

function span(text: string, className: string): ReactNode {
	return (
		<span key={keyCounter++} className={className}>
			{text}
		</span>
	);
}

export function highlightJSON(code: string): ReactNode[] {
	keyCounter = 0;

	if (!code || !code.trim()) {
		return [<span key={0}>{code}</span>];
	}

	const nodes: ReactNode[] = [];
	let lastIndex = 0;

	for (const match of code.matchAll(JSON_REGEX)) {
		const idx = match.index;

		// Text between matches (whitespace, newlines)
		if (idx > lastIndex) {
			nodes.push(<span key={keyCounter++}>{code.slice(lastIndex, idx)}</span>);
		}

		const [full, str, colon, bool, nul, num, brace] = match;

		if (str) {
			if (colon) {
				// Key: "name":
				nodes.push(span(str, C_KEY));
				nodes.push(<span key={keyCounter++}>{colon}</span>);
			} else {
				// String value
				nodes.push(span(full, C_STRING));
			}
		} else if (bool) {
			nodes.push(span(full, C_BOOL));
		} else if (nul) {
			nodes.push(span(full, C_NULL));
		} else if (num) {
			nodes.push(span(full, C_NUMBER));
		} else if (brace) {
			nodes.push(span(full, C_BRACE));
		}

		lastIndex = idx + full.length;
	}

	// Trailing text
	if (lastIndex < code.length) {
		nodes.push(<span key={keyCounter++}>{code.slice(lastIndex)}</span>);
	}

	return nodes.length > 0 ? nodes : [<span key={0}>{code}</span>];
}
