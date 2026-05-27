import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useAuth } from "./useAuth";

vi.mock("../config", () => ({
  appConfig: {
    authMode: "dev",
    keycloakUrl: "http://localhost:8089",
    keycloakRealm: "k8s-ai",
    keycloakClientId: "k8s-ai-frontend",
    keycloakRedirectUri: "http://localhost:5173/auth/callback",
    demoUser: "demo",
    demoRole: "operator"
  }
}));

vi.mock("../infrastructure/api/userApi", () => ({
  getCurrentUser: vi.fn(async () => ({
    id: "user-demo",
    username: "demo",
    displayName: "Demo",
    role: "operator",
    status: "active"
  }))
}));

describe("useAuth", () => {
  it("waits for explicit dev login before loading user", async () => {
    const { result } = renderHook(() => useAuth());

    expect(result.current.user).toBeNull();
    await act(async () => {
      await result.current.login();
    });

    await waitFor(() => expect(result.current.user?.username).toBe("demo"));
    expect(result.current.auth.mode).toBe("dev");
  });
});
