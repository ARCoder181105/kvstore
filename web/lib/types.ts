export interface KeyEntry {
  key: string;
  value: string;
  ttl: number; // seconds remaining, -1 = no expiry
}

export interface KeysResponse {
  keys: KeyEntry[];
  count: number;
}

export interface Stats {
  total_keys: number;
  uptime: string;
  memory_bytes: number;
  ttl_keys: number;
  connected_clients: number;
}

export type EventType = "SET" | "DEL" | "EXPIRE" | "EXPIRED";

export interface StoreEvent {
  type: EventType;
  key: string;
  value?: string;
  ttl?: number;
  timestamp: string;
}

export interface SetKeyPayload {
  value: string;
  ttl?: number;
}

export interface ApiError {
  error: string;
  code: string;
}
