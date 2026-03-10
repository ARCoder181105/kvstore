# Tech Stack

> Every technology used in this project — what it is, why it was chosen, and where to learn it.

---

## Backend — Go

### Language: Go 1.22+

**Why Go for this project:**
- Goroutines make the "one goroutine per connection" model trivial to implement
- `sync` package gives you `RWMutex`, `WaitGroup`, and atomic primitives out of the box
- The standard library has everything you need for TCP (`net`), HTTP (`net/http`), file I/O (`os`), and binary encoding
- The race detector (`go test -race`) will catch every concurrency bug during development
- Compiles to a single static binary — deploy by copying one file

**Learn it:** [Tour of Go](https://go.dev/tour) → [Effective Go](https://go.dev/doc/effective_go) → [Go by Example](https://gobyexample.com)

---

### `sync` package (standard library)

**What:** Concurrency primitives — `RWMutex`, `Mutex`, `WaitGroup`, `Once`, `Map`

**Used for:** Protecting the store's `data` map from concurrent access. `RWMutex` allows multiple simultaneous readers or one exclusive writer.

**Learn it:** [sync package docs](https://pkg.go.dev/sync)

---

### `container/heap` (standard library)

**What:** A min-heap implementation. You provide a type that implements `heap.Interface` (5 methods) and the package gives you O(log n) push/pop.

**Used for:** TTL eviction heap — keys sorted by expiry time so the background goroutine knows exactly when the next key expires without scanning all keys.

**Learn it:** [container/heap docs](https://pkg.go.dev/container/heap) — read the example carefully, it is the exact pattern you will use.

---

### `encoding/gob` (standard library)

**What:** Go's native binary serialization format. Encodes Go structs to bytes and back.

**Used for:** Snapshot file format. Serialize the entire `map[string]*Entry` to disk and reload it on startup.

**Learn it:** [encoding/gob docs](https://pkg.go.dev/encoding/gob)

---

### `chi` router — `github.com/go-chi/chi/v5`

**What:** A lightweight HTTP router for Go. Handles URL parameters, method routing, and middleware composition.

**Why not standard library `net/http`:** The standard library router does not support URL parameters like `{key}` cleanly. `chi` is minimal (zero external dependencies) and idiomatic Go.

**Why not Gin or Fiber:** They add too much abstraction for this project. `chi` stays close to standard `net/http`, so you learn the actual HTTP model.

**Learn it:** [chi README](https://github.com/go-chi/chi)

---

### `gorilla/websocket` — `github.com/gorilla/websocket`

**What:** WebSocket server and client implementation for Go.

**Used for:** Upgrading HTTP connections to WebSocket for the live event stream endpoint `/ws/events`.

**Learn it:** [gorilla/websocket docs](https://pkg.go.dev/github.com/gorilla/websocket) — the chat example is the exact pattern you need.

---

### `cobra` — `github.com/spf13/cobra`

**What:** CLI framework used by kubectl, Hugo, and most major Go CLIs. Handles subcommands, flags, and help text.

**Used for:** The `kvcli` client — subcommands like `set`, `get`, `del`, plus `--host` and `--port` flags.

**Learn it:** [cobra user guide](https://github.com/spf13/cobra/blob/main/site/content/user_guide.md)

---

### `prometheus/client_golang` — `github.com/prometheus/client_golang`

**What:** Official Go client for Prometheus metrics.

**Used for:** Counters (commands processed), gauges (key count), histograms (command latency).

**Learn it:** [Getting started with Prometheus in Go](https://prometheus.io/docs/guides/go-application/)

---

## Frontend — Next.js

### Framework: Next.js 15 with App Router

**What:** React framework with file-based routing, server components, and full-stack capabilities.

**Why Next.js 15 with App Router:**
- App Router is the modern, current way to build Next.js apps — not the old Pages Router
- Server Components render on the server, reducing client bundle size
- The `use client` directive makes the mental model explicit — you know exactly what runs where
- TypeScript is first-class

**What you use from it:** App Router pages, layouts, route handlers (for the health proxy), client components for interactive parts.

**Learn it:** [Next.js docs — App Router](https://nextjs.org/docs/app)

---

### TypeScript

**What:** JavaScript with static types.

**Why:** You will define types for `KeyEntry`, `Stats`, `Event` once in `lib/types.ts` and get autocomplete and type checking everywhere — in hooks, components, and API calls.

**Learn it:** [TypeScript Handbook](https://www.typescriptlang.org/docs/handbook/intro.html)

---

### Tailwind CSS

**What:** Utility-first CSS framework — style directly with class names like `bg-zinc-900`, `text-sm`, `flex`, `gap-4`.

**Why:** No context switching between CSS files and components. Keeps all styling co-located with the component.

**Learn it:** [Tailwind docs](https://tailwindcss.com/docs)

---

### shadcn/ui

**What:** Component collection built on Radix UI primitives. You install components directly into your project (they are your code, not a package). Accessible, keyboard-navigable, customizable.

**Components used:** Button, Input, Table, Badge, Card, Dialog, Slider, Toast.

**Why not other component libraries:** shadcn/ui generates code you own and can modify. Other libraries are opaque packages. When something looks wrong, you can fix it directly.

**Learn it:** [shadcn/ui docs](https://ui.shadcn.com)

---

### Tanstack Query (React Query) — `@tanstack/react-query`

**What:** Data fetching and server state management library for React.

**Why:** Handles loading states, error states, caching, background refetching, and cache invalidation — all the boilerplate around `fetch` calls. One `useQuery` hook replaces 30 lines of `useEffect` + `useState`.

**Key concepts you use:**
- `useQuery` — fetch and cache data, auto-refetch on interval
- `useMutation` — send POST/DELETE, invalidate cache on success
- `QueryClient.invalidateQueries` — tell React Query to refetch after a mutation

**Learn it:** [Tanstack Query docs](https://tanstack.com/query/latest/docs/framework/react/overview)

---

### Native WebSocket API (browser built-in)

**What:** The browser's built-in WebSocket interface — no library needed.

**Used for:** The `useEventStream` hook connects to `ws://localhost:8080/ws/events` and receives JSON events.

**Learn it:** [MDN WebSocket docs](https://developer.mozilla.org/en-US/docs/Web/API/WebSocket)

---

## Development Tools

### golangci-lint

**What:** Go linter that runs 50+ linters in one command.

**Used for:** Catch bugs, unused imports, error handling mistakes before they become runtime issues.

**Install:** `go install github.com/golangci-lint/golangci-lint/cmd/golangci-lint@latest`

---

### Docker Compose (Phase 7+)

**What:** Tool to define and run multi-container Docker apps with a single YAML file.

**Used for:** Running a 3-node Raft cluster locally for development. One `docker-compose up` starts all three nodes.

---

## What You Are NOT Using (and Why)

| Technology | Why Not |
|------------|---------|
| Redis SDK | Defeats the purpose — you are building the database, not using one |
| gRPC | Good choice for Raft RPCs, but adds complexity. Use HTTP/JSON first, migrate as a stretch goal |
| GraphQL | REST is correct here — simple CRUD operations on keys |
| Redux | Tanstack Query handles all server state. Redux is for complex local state. You have none. |
| Gin / Fiber | Higher-level than needed. `chi` + standard `net/http` teaches the real model |
| MySQL / Postgres | External dependencies. The entire point is zero external database dependencies |
