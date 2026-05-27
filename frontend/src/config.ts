import type { AuthMode } from "./domain/auth";

export type AppConfig = {
  authMode: AuthMode;
  dataMode: "mock" | "api";
  keycloakUrl: string;
  keycloakRealm: string;
  keycloakClientId: string;
  keycloakRedirectUri: string;
  demoUser: string;
  demoRole: "admin" | "operator";
};

export const appConfig: AppConfig = {
  authMode: (import.meta.env.VITE_AUTH_MODE as AuthMode | undefined) ?? "dev",
  dataMode: import.meta.env.VITE_DATA_MODE === "api" ? "api" : "mock",
  keycloakUrl: import.meta.env.VITE_KEYCLOAK_URL ?? "http://localhost:8089",
  keycloakRealm: import.meta.env.VITE_KEYCLOAK_REALM ?? "k8s-ai",
  keycloakClientId: import.meta.env.VITE_KEYCLOAK_CLIENT_ID ?? "k8s-ai-frontend",
  keycloakRedirectUri: import.meta.env.VITE_KEYCLOAK_REDIRECT_URI ?? `${window.location.origin}/auth/callback`,
  demoUser: import.meta.env.VITE_DEMO_USER ?? "demo",
  demoRole: import.meta.env.VITE_DEMO_ROLE === "operator" ? "operator" : "admin"
};
