const MINUTE = 60;
const HOUR = 3600;
const DAY = 86_400;

/**
 * Returns a human-readable relative time string (e.g. "2 minutes ago").
 * Falls back to "just now" for times less than 30 seconds ago.
 */
export function relativeTime(dateStr: string | undefined): string {
	if (!dateStr) return "never";

	const diff = Math.floor(
		(Date.now() - new Date(dateStr).getTime()) / 1000,
	);

	if (diff < 0) return "just now";
	if (diff < 30) return "just now";
	if (diff < MINUTE) return `${diff}s ago`;
	if (diff < HOUR) {
		const m = Math.floor(diff / MINUTE);
		return `${m}m ago`;
	}
	if (diff < DAY) {
		const h = Math.floor(diff / HOUR);
		return `${h}h ago`;
	}
	const d = Math.floor(diff / DAY);
	return `${d}d ago`;
}
