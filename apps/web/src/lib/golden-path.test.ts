import { describe, expect, it } from "vitest";
import { previewError, previewFiles } from "./golden-path";

describe("golden path preview", () => {
  it("renders a non-root deployment skeleton", () => {
    const files = previewFiles({ name: "billing-api", runtime: "go", owner: "payments", slo: "99.9", strategy: "canary" });
    const deploy = files.find((file) => file.path === "deploy/deployment.yaml");
    expect(deploy?.body).toContain("runAsNonRoot: true");
    expect(deploy?.body).toContain("billing-api");
    expect(files.map((file) => file.path)).toContain("slo.yaml");
  });

  it("rejects a name that could overwrite a path", () => {
    expect(previewError({ name: "../payment-api", runtime: "go", owner: "payments", slo: "99.9", strategy: "rolling" })).toBeTruthy();
  });
});
