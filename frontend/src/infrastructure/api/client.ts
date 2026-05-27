export type ApiAuth =
  | { mode: "dev"; demoUser?: string; demoRole?: string; accessToken?: never }
  | { mode: "keycloak"; accessToken: string; demoUser?: never; demoRole?: never };

export type ApiErrorPayload = {
  error?: {
    code?: string;
    message?: string;
    requestId?: string;
  };
};

export class ApiError extends Error {
  code: string;
  requestId?: string;
  status: number;

  constructor(status: number, code: string, message: string, requestId?: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.requestId = requestId;
  }
}

export function buildAuthHeaders(auth: ApiAuth): Record<string, string> {
  if (auth.mode === "keycloak") {
    return { Authorization: `Bearer ${auth.accessToken}` };
  }

  return {
    ...(auth.demoUser ? { "X-Demo-User": auth.demoUser } : {}),
    ...(auth.demoRole ? { "X-Demo-Role": auth.demoRole } : {})
  };
}

export async function apiRequest<T>(
  path: string,
  options: { method?: string; body?: unknown; auth: ApiAuth; headers?: Record<string, string> }
): Promise<T> {
  const response = await fetch(path, {
    method: options.method ?? "GET",
    headers: {
      ...buildAuthHeaders(options.auth),
      ...(options.body ? { "Content-Type": "application/json" } : {}),
      ...options.headers
    },
    body: options.body ? JSON.stringify(options.body) : undefined
  });

  if (!response.ok) {
    let payload: ApiErrorPayload = {};
    try {
      payload = await response.json();
    } catch {
      payload = {};
    }

    throw new ApiError(
      response.status,
      payload.error?.code ?? "HTTP_ERROR",
      payload.error?.message ?? `HTTP ${response.status}`,
      payload.error?.requestId
    );
  }

  return (await response.json()) as T;
}
