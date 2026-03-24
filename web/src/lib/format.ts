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

/** Simple human-readable cron description. */
export function describeCron(expr: string): string {
	const parts = expr.trim().split(/\s+/);
	if (parts.length !== 5) return expr;

	const cronMap: Record<string, string> = {
		"* * * * *": "Every minute",
		"*/5 * * * *": "Every 5 minutes",
		"*/15 * * * *": "Every 15 minutes",
		"*/30 * * * *": "Every 30 minutes",
		"0 * * * *": "Every hour",
		"0 */2 * * *": "Every 2 hours",
		"0 */6 * * *": "Every 6 hours",
		"0 */12 * * *": "Every 12 hours",
		"0 0 * * *": "Daily at midnight",
		"0 2 * * *": "Daily at 02:00",
		"0 4 * * *": "Daily at 04:00",
		"0 0 * * 0": "Weekly on Sunday",
		"0 0 1 * *": "Monthly on the 1st",
	};

	const normalized = parts.join(" ");
	if (cronMap[normalized]) return cronMap[normalized];

	const [min, hour, dom, , dow] = parts;
	if (dom === "*" && dow === "*" && !min.includes("*") && !hour.includes("*")) {
		return `Daily at ${hour.padStart(2, "0")}:${min.padStart(2, "0")}`;
	}
	return expr;
}
