import createClient from "openapi-fetch";
import type { components, paths } from "./schema";

export type Pet = components["schemas"]["Pet"];
export type User = components["schemas"]["User"];
export type Action = components["schemas"]["Action"];
export type Reaction = components["schemas"]["Reaction"];
export type Stats = Pet["stats"];
export type AuthProvider = components["schemas"]["AuthProvider"];
export type NotificationKind = components["schemas"]["NotificationKind"];
export type NotificationPreferences = components["schemas"]["NotificationPreferences"];
export type PushSubscriptionBody = components["schemas"]["PushSubscription"];

export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: string,
    message: string,
    readonly retryAfter = 0,
  ) {
    super(message);
  }
}

const AUTH_PATHS = ["/auth/login", "/auth/register", "/auth/refresh", "/auth/logout"];

let inflightRefresh: Promise<boolean> | null = null;

function refreshSession(): Promise<boolean> {
  if (!inflightRefresh) {
    inflightRefresh = fetch("/api/v1/auth/refresh", { method: "POST", credentials: "include" })
        .then((res) => res.ok)
        .catch(() => false)
        .finally(() => {
          inflightRefresh = null;
        });
  }
  return inflightRefresh;
}

async function fetchWithRefresh(input: Request): Promise<Response> {
  const retry = input.clone();
  const res = await fetch(input);
  const { pathname } = new URL(input.url);
  if (res.status !== 401 || AUTH_PATHS.some((p) => pathname.endsWith(p))) return res;
  return (await refreshSession()) ? fetch(retry) : res;
}

const http = createClient<paths>({ baseUrl: "/api/v1", credentials: "include", fetch: fetchWithRefresh });

async function unwrap<T>(req: Promise<{ data?: T; error?: unknown; response: Response }>): Promise<T> {
  const { data, error, response } = await req;
  if (!response.ok) {
    const e = (error ?? {}) as { error?: string; message?: string };
    const retryAfter = Number(response.headers.get("Retry-After") ?? 0) || 0;
    throw new ApiError(response.status, e.error ?? "unknown", e.message ?? response.statusText, retryAfter);
  }
  return data as T;
}

export const api = {
  register: (email: string, password: string, nickname: string) =>
    unwrap(http.POST("/auth/register", { body: { email, password, nickname } })),
  login: (email: string, password: string) => unwrap(http.POST("/auth/login", { body: { email, password } })),
  logout: () => unwrap(http.POST("/auth/logout")),
  me: () => unwrap(http.GET("/me")),
  myPet: () => unwrap(http.GET("/pets/me")),
  createPet: (name: string, appearance: string) => unwrap(http.POST("/pets", { body: { name, appearance } })),
  act: (action: Action) => unwrap(http.POST("/pets/me/actions", { body: { action } })),
  inventory: () => unwrap(http.GET("/inventory")),
  claimDailyReward: () => unwrap(http.POST("/inventory/daily-reward")),
  authProviders: () => unwrap(http.GET("/auth/providers")),
  pushKey: () => unwrap(http.GET("/notifications/push-key")),
  subscribePush: (body: PushSubscriptionBody) => unwrap(http.POST("/notifications/subscriptions", { body })),
  unsubscribePush: (endpoint: string) => unwrap(http.DELETE("/notifications/subscriptions", { body: { endpoint } })),
  notificationPreferences: () => unwrap(http.GET("/notifications/preferences")),
  setNotificationPreferences: (body: NotificationPreferences) => unwrap(http.PUT("/notifications/preferences", { body })),
};

export const oauthStartURL = (provider: AuthProvider) => `/api/v1/auth/oauth/${provider}`;
