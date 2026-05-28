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
  if (globalThis.crypto?.getRandomValues) {
    globalThis.crypto.getRandomValues(bytes);
  } else {
    for (let i = 0; i < bytes.length; i += 1) {
      bytes[i] = Math.floor(Math.random() * 256);
    }
  }
  return base64Url(bytes);
}

function rightRotate(value: number, bits: number): number {
  return (value >>> bits) | (value << (32 - bits));
}

function sha256Fallback(input: Uint8Array): Uint8Array {
  const hash = new Uint32Array([
    0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a,
    0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19,
  ]);
  const constants = new Uint32Array([
    0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
    0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
    0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
    0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
    0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
    0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
    0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
    0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
  ]);
  const bitLength = input.length * 8;
  const withOne = input.length + 1;
  const paddedLength = Math.ceil((withOne + 8) / 64) * 64;
  const padded = new Uint8Array(paddedLength);
  padded.set(input);
  padded[input.length] = 0x80;
  const view = new DataView(padded.buffer);
  view.setUint32(paddedLength - 4, bitLength, false);

  const words = new Uint32Array(64);
  for (let offset = 0; offset < paddedLength; offset += 64) {
    for (let i = 0; i < 16; i += 1) {
      words[i] = view.getUint32(offset + i * 4, false);
    }
    for (let i = 16; i < 64; i += 1) {
      const s0 = rightRotate(words[i - 15], 7) ^ rightRotate(words[i - 15], 18) ^ (words[i - 15] >>> 3);
      const s1 = rightRotate(words[i - 2], 17) ^ rightRotate(words[i - 2], 19) ^ (words[i - 2] >>> 10);
      words[i] = (words[i - 16] + s0 + words[i - 7] + s1) >>> 0;
    }
    let [a, b, c, d, e, f, g, h] = hash;
    for (let i = 0; i < 64; i += 1) {
      const s1 = rightRotate(e, 6) ^ rightRotate(e, 11) ^ rightRotate(e, 25);
      const ch = (e & f) ^ (~e & g);
      const temp1 = (h + s1 + ch + constants[i] + words[i]) >>> 0;
      const s0 = rightRotate(a, 2) ^ rightRotate(a, 13) ^ rightRotate(a, 22);
      const maj = (a & b) ^ (a & c) ^ (b & c);
      const temp2 = (s0 + maj) >>> 0;
      h = g;
      g = f;
      f = e;
      e = (d + temp1) >>> 0;
      d = c;
      c = b;
      b = a;
      a = (temp1 + temp2) >>> 0;
    }
    hash[0] = (hash[0] + a) >>> 0;
    hash[1] = (hash[1] + b) >>> 0;
    hash[2] = (hash[2] + c) >>> 0;
    hash[3] = (hash[3] + d) >>> 0;
    hash[4] = (hash[4] + e) >>> 0;
    hash[5] = (hash[5] + f) >>> 0;
    hash[6] = (hash[6] + g) >>> 0;
    hash[7] = (hash[7] + h) >>> 0;
  }

  const output = new Uint8Array(32);
  const outputView = new DataView(output.buffer);
  hash.forEach((value, index) => outputView.setUint32(index * 4, value, false));
  return output;
}

export async function createCodeChallenge(verifier: string): Promise<string> {
  const bytes = new TextEncoder().encode(verifier);
  if (globalThis.crypto?.subtle?.digest) {
    const digest = await globalThis.crypto.subtle.digest("SHA-256", bytes);
    return base64Url(new Uint8Array(digest));
  }
  return base64Url(sha256Fallback(bytes));
}

function keycloakEndpoint(config: AppConfig, path: string): URL {
  const base = new URL(config.keycloakUrl, window.location.origin);
  const prefix = base.pathname.replace(/\/$/, "");
  return new URL(`${prefix}/realms/${config.keycloakRealm}${path}`, base.origin);
}

export function buildAuthorizeUrl(config: AppConfig, challenge: string, state: string): string {
  const url = keycloakEndpoint(config, "/protocol/openid-connect/auth");
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

  const response = await fetch(keycloakEndpoint(config, "/protocol/openid-connect/token"), {
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
  const url = keycloakEndpoint(config, "/protocol/openid-connect/logout");
  url.searchParams.set("client_id", config.keycloakClientId);
  url.searchParams.set("post_logout_redirect_uri", window.location.origin);
  return url.toString();
}
