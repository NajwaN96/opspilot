import { describe, expect, it } from "vitest";
import { clusterModeLabel, venueLabel } from "./venue";

describe("venueLabel", () => {
  it("keeps the local cluster and the AWS dev cluster distinct", () => {
    expect(venueLabel(undefined)).toBe("LOCAL");
    expect(venueLabel({ cluster: "opspilot-dev", mode: "local-kubernetes" })).toBe("LOCAL");
    expect(venueLabel({ cluster: "opspilot-aws-dev", mode: "eks" })).toBe("AWS DEV");
    expect(venueLabel({ venueLabel: "AWS DEV" })).toBe("AWS DEV");
    expect(venueLabel({ cluster: "prod" })).toBe("UNKNOWN");
  });
});

describe("clusterModeLabel", () => {
  it("names the platform without treating an unknown mode as local", () => {
    expect(clusterModeLabel("local-kubernetes")).toBe("Local Kubernetes");
    expect(clusterModeLabel("eks")).toBe("Amazon EKS");
    expect(clusterModeLabel("unavailable")).toBe("Unavailable");
  });
});
