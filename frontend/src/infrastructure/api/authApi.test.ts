import { describe, expect, it } from "vitest";
import { buildAuthorizeUrl, verifyCallbackState } from "./authApi";

describe("auth api", () => {
  it("builds keycloak authorize url with pkce params", () => {
    const url = buildAuthorizeUrl(
      {
        authMode: "keycloak",
        dataMode: "api",
        keycloakUrl: "http://localhost:8089",
        keycloakRealm: "k8s-ai",
        keycloakClientId: "k8s-ai-frontend",
        keycloakRedirectUri: "http://localhost:5173/auth/callback",
        demoUser: "demo",
        demoRole: "operator"
      },
      "challenge-001",
      "state-001"
    );

    expect(url).toContain("http://localhost:8089/realms/k8s-ai/protocol/openid-connect/auth");
    expect(url).toContain("client_id=k8s-ai-frontend");
    expect(url).toContain("code_challenge=challenge-001");
    expect(url).toContain("code_challenge_method=S256");
    expect(url).toContain("state=state-001");
  });

  it("rejects callback when state does not match", () => {
    expect(() => verifyCallbackState("actual", "expected")).toThrow("登录状态校验失败");
  });
});
