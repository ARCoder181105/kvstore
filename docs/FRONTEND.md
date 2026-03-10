# Frontend — Next.js Dashboard

> Complete guide to the browser dashboard — architecture, components, hooks, and real-time patterns.

---

## Stack

| Tool | Version | Role |
|------|---------|------|
| Next.js | 15 (App Router) | Framework, routing, server components |
| TypeScript | 5 | Type safety across all components and hooks |
| Tailwind CSS | 3 | Utility-first styling |
| shadcn/ui | latest | Accessible component primitives |
| Tanstack Query | 5 | Server state management, caching, mutations |
| Native WebSocket | built-in | Real-time event stream from the Go server |

---

## Project Setup

```bash
# Create the Next.js project
npx create-next-app@latest web \
  --typescript \
  --tailwind \
  --app \
  --no-src-dir \
  --import-alias "@/*"

cd web

# Install Tanstack Query
npm install @tanstack/react-query @tanstack/react-query-devtools

# Initialize shadcn/ui
npx shadcn@latest init
# When prompted: Default style, Zinc color, CSS variables: yes

# Add the shadcn components you will use
npx shadcn@latest add button input table badge card dialog slider
```

---

## Environment Variables

```bash
# .env.local
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_WS_URL=ws://localhost:8080
```

---

## Types — `lib/types.ts`

Define all types once. Use everywhere.

```typescript
export interface KeyEntry {
  key: string
  value: string
  ttl: number       // seconds remaining, -1 = no expiry
}

export interface KeysResponse {
  keys: KeyEntry[]
  count: number
}

export interface Stats {
  total_keys: number
  memory_bytes: number
  uptime: string
  aof_size_bytes: number
  snapshot_age_seconds: number
  commands_processed: number
}

export type EventType = "SET" | "DEL" | "EXPIRE" | "EXPIRED"

export interface StoreEvent {
  type: EventType
  key: string
  value?: string
  ttl?: number
  timestamp: string
}

export interface SetKeyPayload {
  value: string
  ttl?: number
}

export interface ApiError {
  error: string
  code: string
}
```

---

## API Client — `lib/api.ts`

Typed wrappers around every HTTP endpoint. Keep all fetch calls in this one file.

```typescript
const BASE = process.env.NEXT_PUBLIC_API_URL

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { "Content-Type": "application/json" },
    ...init,
  })
  if (!res.ok) {
    const err: ApiError = await res.json()
    throw new Error(err.error)
  }
  return res.json()
}

export const api = {
  getKeys: (pattern?: string) =>
    request<KeysResponse>(`/api/keys${pattern ? `?pattern=${pattern}` : ""}`),

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
}
```

---

## Hooks

### `hooks/useKeys.ts`

```typescript
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { api } from "@/lib/api"
import type { SetKeyPayload } from "@/lib/types"

export function useKeys(pattern?: string) {
  return useQuery({
    queryKey: ["keys", pattern],
    queryFn: () => api.getKeys(pattern),
    refetchInterval: 5000,   // poll every 5s as fallback for WebSocket gaps
    staleTime: 1000,
  })
}

export function useSetKey() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ key, payload }: { key: string; payload: SetKeyPayload }) =>
      api.setKey(key, payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["keys"] })
      qc.invalidateQueries({ queryKey: ["stats"] })
    },
  })
}

export function useDeleteKey() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (key: string) => api.deleteKey(key),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["keys"] })
      qc.invalidateQueries({ queryKey: ["stats"] })
    },
  })
}

export function useSetExpire() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ key, seconds }: { key: string; seconds: number }) =>
      api.setExpire(key, seconds),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["keys"] }),
  })
}
```

### `hooks/useStats.ts`

```typescript
import { useQuery } from "@tanstack/react-query"
import { api } from "@/lib/api"

export function useStats() {
  return useQuery({
    queryKey: ["stats"],
    queryFn: api.getStats,
    refetchInterval: 3000,
  })
}
```

### `hooks/useEventStream.ts`

This is the most interesting hook. It manages a WebSocket connection with automatic reconnection.

```typescript
import { useEffect, useRef, useState, useCallback } from "react"
import type { StoreEvent } from "@/lib/types"

type ConnectionState = "connecting" | "connected" | "disconnected"

export function useEventStream(maxEvents = 100) {
  const [events, setEvents] = useState<StoreEvent[]>([])
  const [connectionState, setConnectionState] = useState<ConnectionState>("connecting")
  const wsRef = useRef<WebSocket | null>(null)
  const retryTimeoutRef = useRef<ReturnType<typeof setTimeout>>()
  const retryDelayRef = useRef(1000)

  const connect = useCallback(() => {
    const wsUrl = process.env.NEXT_PUBLIC_WS_URL
    if (!wsUrl) return

    const ws = new WebSocket(`${wsUrl}/ws/events`)
    wsRef.current = ws

    ws.onopen = () => {
      setConnectionState("connected")
      retryDelayRef.current = 1000  // reset backoff on successful connect
    }

    ws.onmessage = (e) => {
      try {
        const event: StoreEvent = JSON.parse(e.data)
        setEvents((prev) => [event, ...prev].slice(0, maxEvents))
      } catch {
        // malformed message — ignore
      }
    }

    ws.onclose = () => {
      setConnectionState("disconnected")
      // Exponential backoff: 1s, 2s, 4s, 8s, ... max 30s
      retryTimeoutRef.current = setTimeout(() => {
        retryDelayRef.current = Math.min(retryDelayRef.current * 2, 30000)
        connect()
      }, retryDelayRef.current)
    }

    ws.onerror = () => {
      ws.close()
    }
  }, [maxEvents])

  useEffect(() => {
    connect()
    return () => {
      clearTimeout(retryTimeoutRef.current)
      wsRef.current?.close()
    }
  }, [connect])

  return { events, connectionState }
}
```

---

## Components

### `StatsPanel.tsx`

Displays three stat cards — total keys, memory, uptime.

```typescript
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { useStats } from "@/hooks/useStats"

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export function StatsPanel() {
  const { data: stats, isLoading } = useStats()

  const items = [
    { label: "Total Keys", value: stats?.total_keys ?? "—" },
    { label: "Memory", value: stats ? formatBytes(stats.memory_bytes) : "—" },
    { label: "Uptime", value: stats?.uptime ?? "—" },
  ]

  return (
    <div className="grid grid-cols-3 gap-4">
      {items.map((item) => (
        <Card key={item.label}>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-zinc-400">
              {item.label}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">{item.value}</p>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
```

### `EventStream.tsx`

Renders the live WebSocket event log with auto-scroll.

```typescript
"use client"

import { useEffect, useRef } from "react"
import { Badge } from "@/components/ui/badge"
import { useEventStream } from "@/hooks/useEventStream"
import type { StoreEvent } from "@/lib/types"

const eventColors: Record<string, string> = {
  SET: "bg-green-900 text-green-300",
  DEL: "bg-red-900 text-red-300",
  EXPIRE: "bg-yellow-900 text-yellow-300",
  EXPIRED: "bg-zinc-700 text-zinc-300",
}

function EventRow({ event }: { event: StoreEvent }) {
  const time = new Date(event.timestamp).toLocaleTimeString()
  const colorClass = eventColors[event.type] ?? "bg-zinc-700 text-zinc-300"

  return (
    <div className="flex items-center gap-3 py-1 text-sm font-mono border-b border-zinc-800">
      <span className="text-zinc-500 w-20 shrink-0">{time}</span>
      <Badge className={`w-16 justify-center text-xs ${colorClass}`}>
        {event.type}
      </Badge>
      <span className="text-zinc-300">{event.key}</span>
      {event.value && (
        <span className="text-zinc-500 truncate">→ {event.value}</span>
      )}
    </div>
  )
}

export function EventStream() {
  const { events, connectionState } = useEventStream(50)
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" })
  }, [events])

  return (
    <div className="rounded-lg border border-zinc-800 bg-zinc-950">
      <div className="flex items-center justify-between px-4 py-2 border-b border-zinc-800">
        <h2 className="text-sm font-semibold">Live Events</h2>
        <span className={`text-xs ${connectionState === "connected" ? "text-green-400" : "text-red-400"}`}>
          {connectionState === "connected" ? "● Connected" : "○ Disconnected"}
        </span>
      </div>
      <div className="h-64 overflow-y-auto px-4 py-2">
        {events.length === 0 && (
          <p className="text-zinc-600 text-sm text-center mt-8">
            Waiting for events...
          </p>
        )}
        {events.map((event, i) => (
          <EventRow key={i} event={event} />
        ))}
        <div ref={bottomRef} />
      </div>
    </div>
  )
}
```

---

## Dashboard Page — `app/page.tsx`

Assemble all components into the full dashboard.

```typescript
import { StatsPanel } from "@/components/dashboard/StatsPanel"
import { KeysTable } from "@/components/dashboard/KeysTable"
import { EventStream } from "@/components/dashboard/EventStream"
import { AddKeyForm } from "@/components/dashboard/AddKeyForm"
import { Header } from "@/components/layout/Header"

export default function DashboardPage() {
  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-100">
      <Header />
      <main className="max-w-7xl mx-auto px-4 py-8 space-y-6">
        <StatsPanel />
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2 space-y-4">
            <AddKeyForm />
            <KeysTable />
          </div>
          <div>
            <EventStream />
          </div>
        </div>
      </main>
    </div>
  )
}
```

---

## Providers Setup — `providers/QueryProvider.tsx`

```typescript
"use client"

import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { ReactQueryDevtools } from "@tanstack/react-query-devtools"
import { useState } from "react"

export function QueryProvider({ children }: { children: React.ReactNode }) {
  const [client] = useState(() => new QueryClient({
    defaultOptions: {
      queries: {
        retry: 1,
        staleTime: 1000,
      },
    },
  }))

  return (
    <QueryClientProvider client={client}>
      {children}
      <ReactQueryDevtools initialIsOpen={false} />
    </QueryClientProvider>
  )
}
```

Add to `app/layout.tsx`:
```typescript
import { QueryProvider } from "@/providers/QueryProvider"

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        <QueryProvider>{children}</QueryProvider>
      </body>
    </html>
  )
}
```

---

## Design Principles for the Dashboard

**Dark terminal aesthetic** — use `zinc-950` as background, `zinc-900` for cards, `zinc-800` for borders. This matches the systems-programming nature of the project.

**Monospace for data** — key names, values, TTL counts, and the event stream should use `font-mono`. Data is data — it should look like it.

**Density over whitespace** — the keys table should show as many rows as possible. This is a developer tool, not a landing page.

**Status is always visible** — the connection badge should always be in the top-right corner. If the Go server dies, you see it instantly.

**Never block the UI** — mutations (set, delete, expire) use optimistic updates via Tanstack Query's `onMutate` hook. The table updates immediately, even before the server confirms.
