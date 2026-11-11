import { describe, expect, it } from "vitest";
import { formatDuration } from "./duration";

describe("formatDuration", () => {
  it("formats hours and minutes", () => {
    expect(formatDuration(3600)).toBe("1h");
    expect(formatDuration(5400)).toBe("1h 30m");
    expect(formatDuration(2700)).toBe("45m");
    expect(formatDuration(30)).toBe("<1m");
    expect(formatDuration(0)).toBe("0m");
  });
});
