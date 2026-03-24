import { useCallback, useEffect, useRef, useState } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { SearchAddon } from "@xterm/addon-search";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { useAuthStore } from "@/stores/auth-store";

interface UseTerminalOptions {
	cmd?: string;
	tty?: boolean;
}

interface UseTerminalReturn {
	containerRef: (el: HTMLDivElement | null) => void;
	connected: boolean;
	searchVisible: boolean;
	searchQuery: string;
	setSearchQuery: (q: string) => void;
	toggleSearch: () => void;
	findNext: () => void;
	findPrevious: () => void;
}

const THEME = {
	background: "#0a0a0a",
	foreground: "#e4e4e7",
	cursor: "#a1a1aa",
	selectionBackground: "#3f3f46",
	black: "#09090b",
	red: "#ef4444",
	green: "#22c55e",
	yellow: "#eab308",
	blue: "#3b82f6",
	magenta: "#a855f7",
	cyan: "#06b6d4",
	white: "#e4e4e7",
	brightBlack: "#52525b",
	brightRed: "#f87171",
	brightGreen: "#4ade80",
	brightYellow: "#facc15",
	brightBlue: "#60a5fa",
	brightMagenta: "#c084fc",
	brightCyan: "#22d3ee",
	brightWhite: "#fafafa",
};

function buildWsUrl(containerId: string, token: string, cmd: string, tty: boolean): string {
	const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
	const host = window.location.host;
	const params = new URLSearchParams({
		token,
		cmd,
		tty: String(tty),
	});
	return `${proto}//${host}/api/dk/containers/${containerId}/terminal?${params}`;
}

export function useTerminal(containerId: string | undefined, options: UseTerminalOptions = {}): UseTerminalReturn {
	const { cmd = "/bin/sh", tty = true } = options;

	const [connected, setConnected] = useState(false);
	const [searchVisible, setSearchVisible] = useState(false);
	const [searchQuery, setSearchQuery] = useState("");

	const termRef = useRef<Terminal | null>(null);
	const fitRef = useRef<FitAddon | null>(null);
	const searchRef = useRef<SearchAddon | null>(null);
	const wsRef = useRef<WebSocket | null>(null);
	const containerElRef = useRef<HTMLDivElement | null>(null);
	const resizeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
	const observerRef = useRef<ResizeObserver | null>(null);

	// Cleanup everything.
	const cleanup = useCallback(() => {
		if (resizeTimerRef.current) clearTimeout(resizeTimerRef.current);
		if (observerRef.current) {
			observerRef.current.disconnect();
			observerRef.current = null;
		}
		if (wsRef.current) {
			wsRef.current.close();
			wsRef.current = null;
		}
		if (termRef.current) {
			termRef.current.dispose();
			termRef.current = null;
		}
		fitRef.current = null;
		searchRef.current = null;
		setConnected(false);
	}, []);

	// Initialize terminal and WebSocket when container element is available.
	const init = useCallback(() => {
		if (!containerId || !containerElRef.current) return;

		const token = useAuthStore.getState().token;
		if (!token) return;

		// Create terminal.
		const term = new Terminal({
			cursorBlink: true,
			fontSize: 14,
			fontFamily: "'JetBrains Mono', 'Cascadia Code', 'Fira Code', Menlo, monospace",
			theme: THEME,
			allowProposedApi: true,
		});
		termRef.current = term;

		// Addons.
		const fit = new FitAddon();
		fitRef.current = fit;
		term.loadAddon(fit);

		const search = new SearchAddon();
		searchRef.current = search;
		term.loadAddon(search);

		term.loadAddon(new WebLinksAddon());

		// Mount.
		term.open(containerElRef.current);
		fit.fit();

		// WebSocket.
		const url = buildWsUrl(containerId, token, cmd, tty);
		const ws = new WebSocket(url);
		ws.binaryType = "arraybuffer";
		wsRef.current = ws;

		ws.onopen = () => {
			setConnected(true);
			term.focus();
			// Send initial resize.
			ws.send(JSON.stringify({ type: "resize", rows: term.rows, cols: term.cols }));
		};

		ws.onmessage = (ev) => {
			if (ev.data instanceof ArrayBuffer) {
				term.write(new Uint8Array(ev.data));
			} else if (typeof ev.data === "string") {
				try {
					const ctrl = JSON.parse(ev.data) as { type: string; exit_code?: number };
					if (ctrl.type === "exit") {
						term.writeln(`\r\n\x1b[90m[Process exited with code ${ctrl.exit_code ?? "?"}]\x1b[0m`);
						setConnected(false);
					}
				} catch {
					// Not JSON, write as text.
					term.write(ev.data);
				}
			}
		};

		ws.onclose = () => setConnected(false);
		ws.onerror = () => setConnected(false);

		// Terminal data → WS stdin (binary).
		term.onData((data) => {
			if (ws.readyState === WebSocket.OPEN) {
				const encoder = new TextEncoder();
				ws.send(encoder.encode(data));
			}
		});

		// Terminal binary data → WS stdin (binary).
		term.onBinary((data) => {
			if (ws.readyState === WebSocket.OPEN) {
				const buf = new Uint8Array(data.length);
				for (let i = 0; i < data.length; i++) {
					buf[i] = data.charCodeAt(i);
				}
				ws.send(buf);
			}
		});

		// Debounced resize.
		const sendResize = () => {
			if (resizeTimerRef.current) clearTimeout(resizeTimerRef.current);
			resizeTimerRef.current = setTimeout(() => {
				fit.fit();
				if (ws.readyState === WebSocket.OPEN) {
					ws.send(JSON.stringify({ type: "resize", rows: term.rows, cols: term.cols }));
				}
			}, 100);
		};

		// ResizeObserver on the container div.
		const observer = new ResizeObserver(sendResize);
		observer.observe(containerElRef.current);
		observerRef.current = observer;
	}, [containerId, cmd, tty]); // cleanup is stable (no deps)

	// Ref callback: when the div mounts, start; when it unmounts, cleanup.
	const containerRef = useCallback(
		(el: HTMLDivElement | null) => {
			// Cleanup previous.
			cleanup();
			containerElRef.current = el;
			if (el) init();
		},
		[cleanup, init],
	);

	// Cleanup on unmount.
	useEffect(() => cleanup, [cleanup]);

	// Search.
	useEffect(() => {
		if (searchRef.current && searchQuery) {
			searchRef.current.findNext(searchQuery);
		}
	}, [searchQuery]);

	const toggleSearch = useCallback(() => {
		setSearchVisible((v) => !v);
		if (searchVisible) {
			setSearchQuery("");
			searchRef.current?.clearDecorations();
		}
	}, [searchVisible]);

	const findNext = useCallback(() => {
		if (searchRef.current && searchQuery) {
			searchRef.current.findNext(searchQuery);
		}
	}, [searchQuery]);

	const findPrevious = useCallback(() => {
		if (searchRef.current && searchQuery) {
			searchRef.current.findPrevious(searchQuery);
		}
	}, [searchQuery]);

	return {
		containerRef,
		connected,
		searchVisible,
		searchQuery,
		setSearchQuery,
		toggleSearch,
		findNext,
		findPrevious,
	};
}
