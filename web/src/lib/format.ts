const MINUTE = 60;
const HOUR = 3600;
const DAY = 86_400;

/**
 * Format bytes into a human-readable string (e.g. "2.1 GB").
 */
export function formatBytes(bytes: number): string {
	if (bytes === 0) return "0 B";
	const units = ["B", "KB", "MB", "GB", "TB"];
	const i = Math.floor(Math.log(bytes) / Math.log(1024));
	const val = bytes / 1024 ** i;
	return `${val.toFixed(1)} ${units[i]}`;
}

/**
 * Format seconds into human-readable uptime (e.g. "108d 12h" or "3h 45m").
 */
export function formatUptime(seconds: number): string {
	if (!seconds || seconds <= 0) return "—";
	const d = Math.floor(seconds / DAY);
	const h = Math.floor((seconds % DAY) / HOUR);
	const m = Math.floor((seconds % HOUR) / MINUTE);
	if (d > 0) return `${d}d ${h}h`;
	if (h > 0) return `${h}h ${m}m`;
	return `${m}m`;
}

/**
 * Format uptime from a started_at date string.
 */
export function formatUptimeFromDate(dateStr: string | undefined): string {
	if (!dateStr) return "—";
	const diff = Math.floor((Date.now() - new Date(dateStr).getTime()) / 1000);
	if (diff <= 0) return "—";
	return formatUptime(diff);
}

/**
 * Returns a human-readable relative time string (e.g. "2m ago" or "in 364d").
 * Handles both past and future dates.
 */
export function relativeTime(dateStr: string | undefined): string {
	if (!dateStr) return "never";

	const diff = Math.floor((Date.now() - new Date(dateStr).getTime()) / 1000);

	if (diff < 0) return formatFuture(-diff);
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

function formatFuture(secs: number): string {
	if (secs < MINUTE) return "in <1m";
	if (secs < HOUR) return `in ${Math.floor(secs / MINUTE)}m`;
	if (secs < DAY) return `in ${Math.floor(secs / HOUR)}h`;
	return `in ${Math.floor(secs / DAY)}d`;
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
