import { describe, expect, it, vi } from "vitest";
import { ApiError, apiRequest, buildAuthHeaders } from "./client";

describe("api client", () => {
  it("adds authorization header in keycloak mode", () => {
    expect(buildAuthHeaders({ mode: "keycloak", accessToken: "token-001" })).toEqual({
      Authorization: "Bearer token-001"
    });
  });

  it("adds demo headers in dev mode", () => {
    expect(buildAuthHeaders({ mode: "dev", demoUser: "admin", demoRole: "admin" })).toEqual({
      "X-Demo-User": "admin",
      "X-Demo-Role": "admin"
    });
  });

  it("throws backend error payload", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => ({
        ok: false,
        status: 400,
        json: async () => ({
          error: { code: "INVALID_REQUEST", message: "Invalid request body.", requestId: "req-001" }
        })
      }))
    );

    await expect(apiRequest("/api/demo", { auth: { mode: "dev" } })).rejects.toMatchObject({
      code: "INVALID_REQUEST",
      message: "Invalid request body.",
      requestId: "req-001"
    } satisfies Partial<ApiError>);
  });
});
