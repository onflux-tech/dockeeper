import { describe, expect, it } from "vitest";
import { isVersionNewer } from "../version";

describe("isVersionNewer", () => {
	it("treats non-release builds as older than any release", () => {
		expect(isVersionNewer("develop", "v0.9.0")).toBe(true);
		expect(isVersionNewer("dev", "v0.9.0")).toBe(true);
		expect(isVersionNewer("", "v0.9.0")).toBe(true);
	});

	it("compares semver with and without the v prefix", () => {
		expect(isVersionNewer("v0.9.0", "v0.9.0")).toBe(false);
		expect(isVersionNewer("0.9.0", "v0.9.0")).toBe(false);
		expect(isVersionNewer("v0.8.9", "v0.9.0")).toBe(true);
		expect(isVersionNewer("v0.10.0", "v0.9.0")).toBe(false);
	});

	it("reads the numeric prefix of build suffixes like the backend", () => {
		expect(isVersionNewer("v0.9.0-dirty", "v0.9.0")).toBe(false);
		expect(isVersionNewer("v0.8.0-dirty", "v0.9.0")).toBe(true);
		expect(isVersionNewer("v0.9.0-1-gabc123", "v0.9.0")).toBe(false);
	});

	it("does not offer a release when the latest version is empty", () => {
		expect(isVersionNewer("v0.9.0", "")).toBe(false);
	});
});
