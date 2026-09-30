import { describe, expect, it } from "vitest";
import { isPortfolioRuntime } from "./runtime";

describe("isPortfolioRuntime", () => {
  it("matches only the AWS portfolio build", () => {
    expect(isPortfolioRuntime(undefined)).toBe(false);
    expect(isPortfolioRuntime("")).toBe(false);
    expect(isPortfolioRuntime("local-live")).toBe(false);
    expect(isPortfolioRuntime("aws-portfolio-demo")).toBe(true);
  });
});
