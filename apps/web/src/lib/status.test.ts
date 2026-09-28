import { describe, expect, it } from "vitest";
import { statusLabel, statusTone } from "./status";

describe("statusTone", () => {
  it("maps operational states to a tone", () => {
    expect(statusTone("connected")).toBe("good");
    expect(statusTone("disconnected")).toBe("bad");
    expect(statusTone("healthy")).toBe("good");
    expect(statusTone("Resolved")).toBe("good");
    expect(statusTone("critical")).toBe("bad");
    expect(statusTone("SEV-2")).toBe("bad");
    expect(statusTone("investigating")).toBe("warn");
    expect(statusTone("SEV-3")).toBe("info");
    expect(statusTone("rolled-back")).toBe("neutral");
  });
});

describe("statusLabel", () => {
  it("keeps severities and title-cases workflow states", () => {
    expect(statusLabel("sev-2")).toBe("SEV-2");
    expect(statusLabel("in-progress")).toBe("In Progress");
    expect(statusLabel("investigating")).toBe("Investigating");
  });
});
