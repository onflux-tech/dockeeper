import { useCallback, useState } from "react";
import type { HistoryEntry } from "./playground-sheet";

const STORAGE_KEY = "dk-playground-history";
const MAX_ENTRIES = 50;

function loadHistory(): HistoryEntry[] {
	try {
		const raw = localStorage.getItem(STORAGE_KEY);
		return raw ? JSON.parse(raw) : [];
	} catch {
		return [];
	}
}

function saveHistory(entries: HistoryEntry[]) {
	localStorage.setItem(STORAGE_KEY, JSON.stringify(entries.slice(0, MAX_ENTRIES)));
}

export function usePlaygroundHistory() {
	const [entries, setEntries] = useState<HistoryEntry[]>(() => loadHistory());

	const addEntry = useCallback((entry: HistoryEntry) => {
		setEntries((prev) => {
			const next = [entry, ...prev].slice(0, MAX_ENTRIES);
			saveHistory(next);
			return next;
		});
	}, []);

	const clearHistory = useCallback(() => {
		setEntries([]);
		localStorage.removeItem(STORAGE_KEY);
	}, []);

	return { entries, addEntry, clearHistory };
}
