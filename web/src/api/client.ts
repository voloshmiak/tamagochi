export type Action = "feed" | "play" | "sleep" | "wash";

export interface Stats {
  satiety: number;
  mood: number;
  energy: number;
  cleanliness: number;
}

export interface Pet {
  id: string;
  name: string;
  appearance: string;
  stats: Stats;
  traits: Record<string, number>;
  cooldowns: Record<Action, number>;
  updatedAt: string;
}

export interface User {
  id: string;
  email: string;
  nickname: string;
}

export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: string,
    message: string,
  ) {
    super(message);
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch("/api/v1" + path, {
    method,
    credentials: "include",
    headers: body === undefined ? undefined : { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: "unknown", message: res.statusText }));
    throw new ApiError(res.status, err.error, err.message);
  }
  return res.status === 204 ? (undefined as T) : res.json();
}

export const api = {
  register: (email: string, password: string, nickname: string) =>
    request<User>("POST", "/auth/register", { email, password, nickname }),
  login: (email: string, password: string) => request<User>("POST", "/auth/login", { email, password }),
  logout: () => request<void>("POST", "/auth/logout"),
  me: () => request<User>("GET", "/me"),
  myPet: () => request<Pet>("GET", "/pets/me"),
  createPet: (name: string, appearance: string) => request<Pet>("POST", "/pets", { name, appearance }),
  act: (action: Action) => request<{ pet: Pet; points: number }>("POST", "/pets/me/actions", { action }),
};
