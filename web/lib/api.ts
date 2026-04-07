import type { KeysResponse, KeyEntry, Stats, SetKeyPayload, ApiError } from "./types";

const BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { "Content-Type": "application/json" },
    ...init,
  });
  if (!res.ok) {
    const err: ApiError = await res.json();
    throw new Error(err.error);
  }
  return res.json();
}

export const api = {
  getKeys: (pattern?: string) =>
    request<KeysResponse>(`/api/keys${pattern ? `?pattern=${encodeURIComponent(pattern)}` : ""}`),

  getKey: (key: string) =>
    request<KeyEntry>(`/api/keys/${encodeURIComponent(key)}`),

  setKey: (key: string, payload: SetKeyPayload) =>
    request<KeyEntry>(`/api/keys/${encodeURIComponent(key)}`, {
      method: "POST",
      body: JSON.stringify(payload),
    }),

  deleteKey: (key: string) =>
    request<{ key: string; deleted: boolean }>(
      `/api/keys/${encodeURIComponent(key)}`,
      { method: "DELETE" }
    ),

  setExpire: (key: string, seconds: number) =>
    request<{ key: string; ttl: number }>(
      `/api/keys/${encodeURIComponent(key)}/expire`,
      { method: "PUT", body: JSON.stringify({ seconds }) }
    ),

  getStats: () => request<Stats>("/api/stats"),

  health: () => request<{ status: string }>("/api/health"),
};
