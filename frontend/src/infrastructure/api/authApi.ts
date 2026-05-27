import type { AppConfig } from "../../config";
import type { AuthSession } from "../../domain/auth";

const verifierKey = "k8s-ai-pkce-verifier";
const stateKey = "k8s-ai-pkce-state";
const sessionKey = "k8s-ai-auth-session";

function base64Url(bytes: Uint8Array): string {
  return btoa(String.fromCharCode(...bytes)).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

export function randomToken(length = 32): string {
  const bytes = new Uint8Array(length);
  crypto.getRandomValues(bytes);
  return base64Url(bytes);
}

export async function createCodeChallenge(verifier: string): Promise<string> {
  const digest = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(verifier));
  return base64Url(new Uint8Array(digest));
}

export function buildAuthorizeUrl(config: AppConfig, challenge: string, state: string): string {
  const url = new URL(`${config.keycloakUrl}/realms/${config.keycloakRealm}/protocol/openid-connect/auth`);
  url.searchParams.set("client_id", config.keycloakClientId);
  url.searchParams.set("redirect_uri", config.keycloakRedirectUri);
  url.searchParams.set("response_type", "code");
  url.searchParams.set("scope", "openid profile email");
  url.searchParams.set("code_challenge", challenge);
  url.searchParams.set("code_challenge_method", "S256");
  url.searchParams.set("state", state);
  return url.toString();
}

export function savePkce(verifier: string, state: string): void {
  sessionStorage.setItem(verifierKey, verifier);
  sessionStorage.setItem(stateKey, state);
}

export function readPkce(): { verifier: string; state: string } | null {
  const verifier = sessionStorage.getItem(verifierKey);
  const state = sessionStorage.getItem(stateKey);
  return verifier && state ? { verifier, state } : null;
}

export function clearPkce(): void {
  sessionStorage.removeItem(verifierKey);
  sessionStorage.removeItem(stateKey);
}

export function verifyCallbackState(actual: string | null, expected: string): void {
  if (!actual || actual !== expected) {
    throw new Error("登录状态校验失败");
  }
}

export async function exchangeCodeForToken(config: AppConfig, code: string, verifier: string): Promise<AuthSession> {
  const body = new URLSearchParams();
  body.set("grant_type", "authorization_code");
  body.set("client_id", config.keycloakClientId);
  body.set("redirect_uri", config.keycloakRedirectUri);
  body.set("code", code);
  body.set("code_verifier", verifier);

  const response = await fetch(`${config.keycloakUrl}/realms/${config.keycloakRealm}/protocol/openid-connect/token`, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body
  });

  if (!response.ok) {
    throw new Error("Keycloak Token 换取失败");
  }

  const payload = await response.json();
  return {
    accessToken: payload.access_token,
    refreshToken: payload.refresh_token,
    tokenType: payload.token_type ?? "Bearer",
    expiresAt: Date.now() + Number(payload.expires_in ?? 0) * 1000
  };
}

export function saveSession(session: AuthSession): void {
  sessionStorage.setItem(sessionKey, JSON.stringify(session));
}

export function readSession(): AuthSession | null {
  const raw = sessionStorage.getItem(sessionKey);
  return raw ? (JSON.parse(raw) as AuthSession) : null;
}

export function clearSession(): void {
  sessionStorage.removeItem(sessionKey);
}

export function buildLogoutUrl(config: AppConfig): string {
  const url = new URL(`${config.keycloakUrl}/realms/${config.keycloakRealm}/protocol/openid-connect/logout`);
  url.searchParams.set("client_id", config.keycloakClientId);
  url.searchParams.set("post_logout_redirect_uri", window.location.origin);
  return url.toString();
}
