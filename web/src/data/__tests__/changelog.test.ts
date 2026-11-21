import { describe, expect, it } from "vitest";
import { changelog, latestRelease, sortChangelog } from "../changelog";

describe("changelog", () => {
	it("keeps Unreleased first", () => {
		expect(sortChangelog(changelog)[0].version).toBe("Unreleased");
	});

	it("exposes the latest released version", () => {
		expect(latestRelease?.version).toBe("0.8.0");
	});

	it("has unique versions", () => {
		expect(new Set(changelog.map((entry) => entry.version)).size).toBe(changelog.length);
	});
});
