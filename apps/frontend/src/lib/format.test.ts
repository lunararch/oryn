import { describe, expect, it } from "vitest";
import { formatDate, formatNumber } from "./format";

describe("formatDate", () => {
  it("formats RFC3339 timestamps for display", () => {
    expect(formatDate("2025-01-02T03:04:05Z")).toBe("Jan 2, 2025");
  });

  it("returns a dash for missing timestamps", () => {
    expect(formatDate("")).toBe("—");
  });
});

describe("formatNumber", () => {
  it("groups thousands", () => {
    expect(formatNumber(1234)).toBe("1,234");
  });
});
