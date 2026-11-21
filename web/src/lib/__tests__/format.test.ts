import { describe, expect, it } from "vitest";
import { formatCpuMem } from "../format";

describe("formatCpuMem", () => {
	it("omits the separator when only one value is present", () => {
		expect(formatCpuMem(0, 1024)).toBe("1.0 KB");
		expect(formatCpuMem(2.5, 0)).toBe("2.5%");
		expect(formatCpuMem(2.5, 1024)).toBe("2.5% · 1.0 KB");
		expect(formatCpuMem(0, 0)).toBe("—");
	});
});
