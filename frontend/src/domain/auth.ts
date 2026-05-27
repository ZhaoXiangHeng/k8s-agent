export type AuthMode = "dev" | "keycloak";

export type AuthSession = {
  accessToken: string;
  refreshToken: string;
  expiresAt: number;
  tokenType: string;
};

export type CurrentUser = {
  id: string;
  username: string;
  displayName?: string;
  email?: string;
  role: "admin" | "operator" | string;
  status?: string;
};
