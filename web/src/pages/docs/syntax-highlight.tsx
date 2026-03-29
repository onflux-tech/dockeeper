import type { ReactNode } from "react";

// ── Token color classes ─────────────────────────────────────────────────────

const C_KEYWORD = "text-blue-400";
const C_STRING = "text-green-400";
const C_FLAG = "text-amber-400";
const C_MUTED = "text-muted-foreground";
const C_DEFAULT = "text-foreground/80";

// ── Helpers ─────────────────────────────────────────────────────────────────

let keyCounter = 0;

function span(text: string, className: string): ReactNode {
  return (
    <span key={keyCounter++} className={className}>
      {text}
    </span>
  );
}

function plain(text: string): ReactNode {
  return <span key={keyCounter++}>{text}</span>;
}

// ── cURL highlighter ────────────────────────────────────────────────────────

const CURL_REGEX =
  /(\/\/[^\n]*)|("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*')|(\\$)|(--?[a-zA-Z][\w-]*)|\b(GET|POST|PUT|PATCH|DELETE|curl)\b|(https?:\/\/\S+)/g;

export const highlightCurl = (code: string): ReactNode[] => {
  keyCounter = 0;
  const nodes: ReactNode[] = [];
  let lastIndex = 0;

  for (const match of code.matchAll(CURL_REGEX)) {
    const idx = match.index;

    // Text before match
    if (idx > lastIndex) {
      nodes.push(span(code.slice(lastIndex, idx), C_DEFAULT));
    }

    const [full, comment, str, continuation, flag, verb, url] = match;

    if (comment) {
      nodes.push(span(full, C_MUTED));
    } else if (str) {
      nodes.push(span(full, C_STRING));
    } else if (continuation) {
      nodes.push(span(full, C_MUTED));
    } else if (flag) {
      nodes.push(span(full, C_FLAG));
    } else if (verb) {
      nodes.push(span(full, C_KEYWORD));
    } else if (url) {
      nodes.push(span(full, C_MUTED));
    }

    lastIndex = idx + full.length;
  }

  // Trailing text
  if (lastIndex < code.length) {
    nodes.push(span(code.slice(lastIndex), C_DEFAULT));
  }

  return nodes.length > 0 ? nodes : [plain(code)];
};

// ── JavaScript highlighter ──────────────────────────────────────────────────

const JS_KEYWORDS =
  /\b(const|let|var|await|async|function|return|throw|new|if|else|try|catch|typeof|import|from|export)\b/;

const JS_REGEX = new RegExp(
  [
    /(\/\/[^\n]*)/.source, // comment
    /(`(?:[^`\\]|\\.)*`)/.source, // template literal
    /("(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*')/.source, // string
    JS_KEYWORDS.source, // keyword
    /(\.\w+(?=\s*\())/.source, // method call
    /(\b\w+(?=\s*:))/.source, // object key
  ].join("|"),
  "g",
);

export const highlightJS = (code: string): ReactNode[] => {
  keyCounter = 0;
  const nodes: ReactNode[] = [];
  let lastIndex = 0;

  for (const match of code.matchAll(JS_REGEX)) {
    const idx = match.index;

    // Text before match
    if (idx > lastIndex) {
      nodes.push(span(code.slice(lastIndex, idx), C_DEFAULT));
    }

    const [full, comment, template, str, keyword, method, objKey] = match;

    if (comment) {
      nodes.push(span(full, C_MUTED));
    } else if (template) {
      nodes.push(span(full, C_STRING));
    } else if (str) {
      nodes.push(span(full, C_STRING));
    } else if (keyword) {
      nodes.push(span(full, C_KEYWORD));
    } else if (method) {
      nodes.push(span(full, C_FLAG));
    } else if (objKey) {
      nodes.push(span(full, C_FLAG));
    }

    lastIndex = idx + full.length;
  }

  // Trailing text
  if (lastIndex < code.length) {
    nodes.push(span(code.slice(lastIndex), C_DEFAULT));
  }

  return nodes.length > 0 ? nodes : [plain(code)];
};
