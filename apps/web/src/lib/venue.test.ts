import { describe, expect, it } from "vitest";
import { clusterModeLabel, venueLabel } from "./venue";

describe("venueLabel", () => {
  it("keeps the local cluster and the AWS dev cluster distinct", () => {
    expect(venueLabel(undefined)).toBe("LOCAL — LIVE");
    expect(venueLabel({ cluster: "opspilot-dev", mode: "local-kubernetes" })).toBe("LOCAL — LIVE");
    expect(venueLabel({ cluster: "opspilot-aws-dev", mode: "eks" })).toBe("AWS — PLAN ONLY");
    expect(venueLabel({ venueLabel: "AWS — PLAN ONLY" })).toBe("AWS — PLAN ONLY");
    expect(venueLabel({ mode: "aws-portfolio-demo" })).toBe("AWS — PORTFOLIO DEMO");
    expect(venueLabel({ cluster: "prod" })).toBe("UNKNOWN");
  });
});

describe("clusterModeLabel", () => {
  it("names the platform without treating an unknown mode as local", () => {
    expect(clusterModeLabel("local-kubernetes")).toBe("Local k3d");
    expect(clusterModeLabel("eks")).toBe("EKS plan only");
    expect(clusterModeLabel("aws-portfolio-demo")).toBe("AWS portfolio demo");
    expect(clusterModeLabel("unavailable")).toBe("Unavailable");
  });
});
