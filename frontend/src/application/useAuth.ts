import { useCallback, useEffect, useMemo, useState } from "react";
import { appConfig } from "../config";
import type { CurrentUser } from "../domain/auth";
import { mockCurrentUser, mockUserPasswords, mockUsers } from "./mockData";
import {
  buildAuthorizeUrl,
  buildLogoutUrl,
  clearPkce,
  clearSession,
  createCodeChallenge,
  exchangeCodeForToken,
  randomToken,
  readPkce,
  readSession,
  savePkce,
  saveSession,
  verifyCallbackState
} from "../infrastructure/api/authApi";
import type { ApiAuth } from "../infrastructure/api/client";
import { getCurrentUser } from "../infrastructure/api/userApi";

export function useAuth() {
  const [session, setSession] = useState(readSession());
  const [user, setUser] = useState<CurrentUser | null>(null);
  const [loading, setLoading] = useState(appConfig.authMode === "keycloak" && !!session);
  const [error, setError] = useState("");
  const [devLoginRequested, setDevLoginRequested] = useState(false);
  const [devCredentials, setDevCredentials] = useState({ username: appConfig.demoUser, password: "" });

  const auth: ApiAuth = useMemo(() => {
    if (appConfig.authMode === "keycloak") {
      return { mode: "keycloak", accessToken: session?.accessToken ?? "" };
    }
    const matchedUser = mockUsers.find((item) => item.username === devCredentials.username);
    return { mode: "dev", demoUser: devCredentials.username || appConfig.demoUser, demoRole: matchedUser?.role ?? appConfig.demoRole };
  }, [devCredentials.username, session?.accessToken]);

  const loadMe = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      if (appConfig.dataMode === "mock" && auth.mode === "dev") {
        const matchedUser = mockUsers.find((item) => item.username === auth.demoUser);
        if (!matchedUser || mockUserPasswords[matchedUser.username] !== devCredentials.password) {
          throw new Error("用户名或密码错误");
        }
        setUser(mockCurrentUser(matchedUser.role, matchedUser.username));
        return;
      }
      const current = await getCurrentUser(auth);
      setUser(current);
    } catch (err) {
      setUser(null);
      setError(err instanceof Error ? err.message : "加载当前用户失败");
    } finally {
      setLoading(false);
    }
  }, [auth]);

  useEffect(() => {
    if (appConfig.authMode === "dev") {
      setLoading(false);
      return;
    }
    if (!session) {
      setLoading(false);
      return;
    }
    void loadMe();
  }, [loadMe, session]);

  const login = useCallback(async (username?: string, password?: string) => {
    if (appConfig.authMode === "dev") {
      setError("");
      const nextCredentials = { username: username ?? appConfig.demoUser, password: password ?? "" };
      setDevCredentials(nextCredentials);
      if (appConfig.dataMode === "mock") {
        const matchedUser = mockUsers.find((item) => item.username === nextCredentials.username);
        if (!matchedUser || mockUserPasswords[matchedUser.username] !== nextCredentials.password) {
          setUser(null);
          setError("用户名或密码错误");
          return;
        }
        setUser(mockCurrentUser(matchedUser.role, matchedUser.username));
        return;
      }
      setDevLoginRequested(true);
      return;
    }

    const verifier = randomToken();
    const state = randomToken(16);
    const challenge = await createCodeChallenge(verifier);
    savePkce(verifier, state);
    window.location.href = buildAuthorizeUrl(appConfig, challenge, state);
  }, [loadMe]);

  const handleCallback = useCallback(async (search: string) => {
    const params = new URLSearchParams(search);
    const code = params.get("code");
    const pkce = readPkce();
    if (!code || !pkce) {
      throw new Error("登录回调参数不完整");
    }

    verifyCallbackState(params.get("state"), pkce.state);
    const session = await exchangeCodeForToken(appConfig, code, pkce.verifier);
    saveSession(session);
    setSession(session);
    clearPkce();
    const current = await getCurrentUser({ mode: "keycloak", accessToken: session.accessToken });
    setUser(current);
  }, [loadMe]);

  const logout = useCallback(() => {
    clearSession();
    setSession(null);
    setUser(null);
    setDevLoginRequested(false);
    if (appConfig.authMode === "keycloak") {
      window.location.href = buildLogoutUrl(appConfig);
    }
  }, []);

  useEffect(() => {
    if (appConfig.authMode === "dev" && devLoginRequested) {
      void loadMe();
    }
  }, [devLoginRequested, loadMe]);

  return { user, loading, error, auth, login, logout, handleCallback, reload: loadMe };
}
