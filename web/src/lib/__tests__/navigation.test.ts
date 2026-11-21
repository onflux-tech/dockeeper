import { describe, expect, it } from "vitest";
import { navigation } from "../navigation";

describe("navigation", () => {
	it("adds Apps under Overview after Servers", () => {
		expect(navigation[0].items.map((item) => item.href)).toEqual(["/", "/servers", "/apps"]);
	});
});
