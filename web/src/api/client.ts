import createClient from "openapi-fetch";
import type { components, paths } from "./schema";

// Types come from api/openapi.yaml: run `npm run generate:api` after changing the contract.
export type Pet = components["schemas"]["Pet"];
export type User = components["schemas"]["User"];
export type Action = components["schemas"]["Action"];
export type Reaction = components["schemas"]["Reaction"];
export type Stats = Pet["stats"];

export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: string,
    message: string,
  ) {
    super(message);
  }
}

const http = createClient<paths>({ baseUrl: "/api/v1", credentials: "include" });

async function unwrap<T>(req: Promise<{ data?: T; error?: unknown; response: Response }>): Promise<T> {
  const { data, error, response } = await req;
  if (!response.ok) {
    const e = (error ?? {}) as { error?: string; message?: string };
    throw new ApiError(response.status, e.error ?? "unknown", e.message ?? response.statusText);
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
};
