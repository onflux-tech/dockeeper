export function isVersionNewer(current: string, latest: string): boolean {
	const c = current.replace(/^v/, "");
	const l = latest.replace(/^v/, "");

	if (l === "") return false;

	if (c === "" || c === "dev" || c === "develop" || !isSemver(c)) {
		return isSemver(l);
	}

	const cParts = c.split(".");
	const lParts = l.split(".");

	for (let i = 0; i < 3; i++) {
		const cv = Number.parseInt(cParts[i] ?? "0", 10) || 0;
		const lv = Number.parseInt(lParts[i] ?? "0", 10) || 0;
		if (lv > cv) return true;
		if (lv < cv) return false;
	}
	return false;
}

function isSemver(value: string): boolean {
	const parts = value.split(".");
	if (parts.length < 2 || parts.length > 3) return false;
	return parts.every((part) => part !== "" && /^\d+$/.test(part));
}
