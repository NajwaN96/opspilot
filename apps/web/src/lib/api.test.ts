import { describe, expect, it } from "vitest";
import { messageFromBody } from "./api";

describe("messageFromBody", () => {
  it("reads the API error envelope", () => {
    expect(messageFromBody({ error: { code: "invalid", message: "policy denied this action" } }, "fallback")).toBe(
      "policy denied this action",
    );
  });

  it("falls back when the body is not an error envelope", () => {
    expect(messageFromBody(null, "Request failed")).toBe("Request failed");
    expect(messageFromBody({ ok: true }, "Request failed")).toBe("Request failed");
  });
});
