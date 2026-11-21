import { describe, expect, it } from "vitest";
import { buildDiskOptions, diskUsage, latestPartitions } from "../disk-view";
import type { DiskPartition } from "@/types/pocketbase";

const partitions: DiskPartition[] = [
	{
		mountpoint: "/",
		device: "/dev/sda1",
		fstype: "ext4",
		total: 1000,
		used: 400,
		available: 600,
		percent: 40,
	},
	{
		mountpoint: "/data",
		device: "/dev/sdb1",
		fstype: "ext4",
		total: 2000,
		used: 500,
		available: 1500,
		percent: 25,
	},
];

describe("diskUsage", () => {
	it("sums every partition", () => {
		expect(diskUsage({ disk_used: 1, disk_total: 2, partitions }, "all")).toEqual({
			used: 900,
			total: 3000,
		});
	});

	it("includes the root disk when the partitions omit the root mount", () => {
		const boot: DiskPartition[] = [
			{
				mountpoint: "/boot",
				device: "/dev/sda2",
				fstype: "ext4",
				total: 1000,
				used: 200,
				available: 800,
				percent: 20,
			},
		];
		expect(diskUsage({ disk_used: 500, disk_total: 10000, partitions: boot }, "all")).toEqual({
			used: 700,
			total: 11000,
		});
	});

	it("falls back to the top-level values when there are no partitions", () => {
		expect(diskUsage({ disk_used: 10, disk_total: 20, partitions: [] }, "all")).toEqual({
			used: 10,
			total: 20,
		});
	});

	it("returns the chosen mountpoint", () => {
		expect(diskUsage({ disk_used: 1, disk_total: 2, partitions }, "/data")).toEqual({
			used: 500,
			total: 2000,
		});
	});

	it("returns null for an unknown mountpoint or a missing sample", () => {
		expect(diskUsage({ disk_used: 1, disk_total: 2, partitions }, "/missing")).toBeNull();
		expect(diskUsage(undefined, "all")).toBeNull();
	});
});

describe("buildDiskOptions", () => {
	it("lists all disks first and deduplicates mountpoints", () => {
		expect(buildDiskOptions(partitions)).toEqual([
			{ value: "all", label: "All disks (sum)" },
			{ value: "/", label: "/" },
			{ value: "/data", label: "/data" },
		]);
	});

	it("keeps the sum option when there are no partitions", () => {
		expect(buildDiskOptions([])).toEqual([{ value: "all", label: "All disks (sum)" }]);
	});
});

describe("latestPartitions", () => {
	it("picks the newest sample that has partitions", () => {
		expect(latestPartitions([{ partitions: [] }, { partitions }, { partitions: [] }])).toBe(
			partitions,
		);
	});

	it("returns an empty list when no sample has partitions", () => {
		expect(latestPartitions([])).toEqual([]);
		expect(latestPartitions([{ partitions: [] }])).toEqual([]);
	});
});
