import { describe, expect, it } from "vitest";
import { completedSteps, formatAgo, formatDuration, formatLatency, formatPercent } from "./format";

describe("formatPercent", () => {
  it("keeps operational precision without trailing zeros", () => {
    expect(formatPercent(12.4)).toBe("12.4%");
    expect(formatPercent(98.71)).toBe("98.71%");
    expect(formatPercent(0.3)).toBe("0.3%");
    expect(formatPercent(100)).toBe("100%");
  });
});

describe("formatLatency", () => {
  it("renders milliseconds and seconds the way the incident view does", () => {
    expect(formatLatency(240)).toBe("240ms");
    expect(formatLatency(1800)).toBe("1.8s");
    expect(formatLatency(90)).toBe("90ms");
    expect(formatLatency(12000)).toBe("12s");
  });
});

describe("formatDuration", () => {
  it("formats short and long durations", () => {
    expect(formatDuration(5_000)).toBe("5s");
    expect(formatDuration(65_000)).toBe("1m 5s");
    expect(formatDuration(3_600_000)).toBe("1h");
    expect(formatDuration(5_400_000)).toBe("1h 30m");
  });
});

describe("formatAgo", () => {
  it("measures from a fixed clock", () => {
    const now = Date.parse("2026-09-28T12:08:00Z");
    expect(formatAgo("2026-09-28T12:00:00Z", now)).toBe("8m ago");
  });
});

describe("completedSteps", () => {
  it("counts only finished remediation steps", () => {
    expect(
      completedSteps([
        { status: "complete" },
        { status: "active" },
        { status: "pending" },
      ]),
    ).toBe(1);
    expect(completedSteps(undefined)).toBe(0);
  });
});
