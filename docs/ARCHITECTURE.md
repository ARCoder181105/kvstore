# System Architecture

> Complete technical design of every layer — how they connect, what they own, and why each decision was made.

---

## The Big Picture

```
┌──────────────────────────────────────────────────────────────────┐
│                         CLIENT TIER                              │
│                                                                  │
│  ┌─────────────────┐  ┌──────────────────┐  ┌────────────────┐  │
│  │  Browser        │  │  CLI (./kvcli)   │  │  curl / test   │  │
│  │  Next.js 15     │  │  Terminal REPL   │  │  clients       │  │
│  └────────┬────────┘  └────────┬─────────┘  └───────┬────────┘  │
│           │ HTTP+WS             │ TCP binary          │ HTTP      │
└───────────┼─────────────────────┼─────────────────────┼──────────┘
            │                     │                     │
┌───────────▼─────────────────────┼─────────────────────▼──────────┐
│                        SERVER TIER                               │
│                                                                  │
│  ┌──────────────────────────┐   │   ┌──────────────────────────┐ │
│  │     HTTP API Server      │   │   │     TCP Server           │ │
│  │     chi router :8080     │   │   │     net.Listener :6379   │ │
│  │                          │   │   │                          │ │
│  │  GET  /api/keys          │   │   │  Binary protocol parser  │ │
│  │  POST /api/keys/:key     │   │   │  One goroutine/conn      │ │
│  │  DEL  /api/keys/:key     │   │   │  bufio.Reader            │ │
│  │  WS   /ws/events         │   │   │                          │ │
│  └────────────┬─────────────┘   │   └────────────┬─────────────┘ │
│               │                 │                │               │
└───────────────┼─────────────────┼────────────────┼───────────────┘
                │                 │                │
┌───────────────▼─────────────────▼────────────────▼───────────────┐
│                         CORE ENGINE                               │
│                    internal/store/store.go                        │
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐   │
│  │  type Store struct                                         │   │
│  │    mu       sync.RWMutex                                   │   │
│  │    data     map[string]*Entry                              │   │
│  │    ttlHeap  *TTLHeap                                       │   │
│  │    events   chan Event          ← WebSocket fan-out        │   │
│  └────────────────────────────────────────────────────────────┘   │
│                                                                   │
│  Commands: SET · GET · DEL · EXPIRE · TTL · KEYS · INCR · MSET   │
│  Thread safety: RWMutex (reads concurrent, writes exclusive)      │
│  TTL: min-heap eviction + lazy check on GET                       │
└────────────────────────────┬──────────────────────────────────────┘
                             │
┌────────────────────────────▼──────────────────────────────────────┐
│                      PERSISTENCE LAYER                            │
│               internal/persistence/aof.go + snapshot.go           │
│                                                                   │
│  AOF Writer (goroutine)              Snapshot Engine              │
│  ─────────────────────               ──────────────               │
│  Reads from buffered channel         Triggered manually or        │
│  Appends binary entries to log       on schedule                  │
│  fsync every 1 second                Write → temp file → rename   │
│  Replay on startup                   Atomic on POSIX systems      │
└────────────────────────────┬──────────────────────────────────────┘
                             │
┌────────────────────────────▼──────────────────────────────────────┐
│                        RAFT CLUSTER                               │
│                      internal/raft/                               │
│                                                                   │
│   ┌──────────┐    RequestVote    ┌──────────┐                     │
│   │  Node A  │◄─────────────────►│  Node B  │                     │
│   │  LEADER  │    AppendEntries  │ FOLLOWER │                     │
│   └────┬─────┘                   └──────────┘                     │
│        │         Heartbeats                                       │
│        │          ┌──────────┐                                    │
│        └─────────►│  Node C  │                                    │
│                   │ FOLLOWER │                                    │
│                   └──────────┘                                    │
└───────────────────────────────────────────────────────────────────┘
```

---

## Layer 1 — Core Engine

**File:** `internal/store/store.go`

This is the heart of everything. All other layers are just different doors into this room.

### Responsibilities
- Store key-value pairs in memory
- Enforce TTL expiry on keys
- Guarantee thread safety under concurrent reads and writes
- Emit events for every mutation (SET, DEL, EXPIRE) — used by WebSocket fan-out

### Why `sync.RWMutex` and not `sync.Map`
`sync.Map` is optimized for the case where keys are written once and read many times. A KV store has unpredictable write patterns. `RWMutex` gives you explicit control — multiple readers hold the lock simultaneously, but any writer gets exclusive access. You can also upgrade to sharded maps later (partition keyspace into N buckets, each with its own mutex) for higher write throughput.

### The Event Channel
Every write operation sends to an unbuffered `events` channel. The HTTP server subscribes to this channel and fans out to all connected WebSocket clients. This is how the dashboard's live event stream works — no polling, no external message broker. Pure Go channels.

---

## Layer 2 — TTL System

**File:** `internal/store/ttl.go`

### The Problem with Naive TTL
The wrong approach is a goroutine that runs every second and iterates every key checking if it has expired. That is O(n) per tick and holds the write lock the entire time, blocking all reads.

### The Right Approach: Min-Heap + Lazy Eviction

**Active eviction (background goroutine):**
A min-heap ordered by `ExpiresAt`. The goroutine runs every 100ms. It only peeks at the minimum element. If it has not expired yet, the goroutine sleeps until that expiry time. If it has expired, it pops it and deletes the key. This is O(log n) per eviction, not O(n).

**Lazy eviction (on every GET):**
When a GET comes in, before returning the value, check if `ExpiresAt` is in the past. If so, delete and return NULL. This catches any keys the background goroutine missed (e.g., on a very loaded system).

Both mechanisms together guarantee no expired key is ever returned to a client.

---

## Layer 3 — TCP Server

**File:** `internal/server/server.go` + `handler.go`

### Connection Model
```
net.Listen("tcp", ":6379")
    └── Accept loop (main goroutine)
            └── go handleConn(conn)   ← one goroutine per connection
                    └── bufio.NewReader(conn)
                    └── for { readFrame → parseCommand → execute → writeResponse }
```

Every connection gets its own goroutine. Go's scheduler multiplexes thousands of goroutines onto a small thread pool efficiently. This is the standard Go networking pattern and scales well for a KV store workload.

### Why Not HTTP for the TCP Layer
HTTP has significant framing overhead — headers, chunked encoding, status lines. A custom binary protocol is faster to parse (no string scanning) and more compact (no repeated header bytes per request). This is why Redis uses its own RESP protocol instead of HTTP.

---

## Layer 4 — Binary Protocol

**File:** `internal/protocol/parser.go` + `serializer.go`

Documented in full in [`PROTOCOLS.md`](PROTOCOLS.md).

The protocol is designed to be simple to implement on both ends — fixed-width length fields, no variable-length framing, single-byte command IDs. You can implement a client in any language in under 100 lines.

---

## Layer 5 — Persistence

**File:** `internal/persistence/aof.go` + `snapshot.go`

### Write Path (What Happens on Every SET)
```
Client sends SET command
    → TCP server receives frame
    → Parses binary frame to Command struct
    → Calls store.Set(key, value)
        → Acquires write lock
        → Updates in-memory map
        → Pushes to TTL heap if TTL specified
        → Sends event to events channel
        → Releases write lock
    → Sends AOF entry to aofChan (non-blocking, buffered)
    → Returns OK to client immediately
    
AOF goroutine (background):
    → Reads from aofChan
    → Appends binary entry to aof.log
    → Every 1 second: calls file.Sync()
```

The client gets OK before the disk write completes. This is the async write pattern — maximum throughput with at most 1 second of data loss on crash. You can make it synchronous (slower, safer) via a config flag.

### Recovery Path (What Happens on Startup)
```
1. Check if snapshot.db exists
      → If yes: deserialize it into the store map
2. Open aof.log
      → Read entries with timestamp > snapshot timestamp
      → Re-execute each command against the store
3. Start accepting connections
```

---

## Layer 6 — HTTP API

**File:** `internal/api/server.go` + `handlers.go` + `ws.go`

A thin REST layer on top of the core engine. The `chi` router is used for its composable middleware and clean URL parameter handling.

### WebSocket Event Stream
The `/ws/events` endpoint upgrades an HTTP connection to WebSocket and then reads from the core engine's event channel, forwarding every SET/DEL/EXPIRE to the browser in JSON format. Multiple browser tabs each get their own goroutine reading from a fan-out subscription.

---

## Layer 7 — CLI Client

**File:** `cmd/kvcli/main.go`

A terminal client that connects to the TCP server via the binary protocol. Two modes:
- **Single command:** `./kvcli SET name Alice`
- **REPL:** `./kvcli` → interactive `kvstore>` prompt

Uses the `cobra` library for subcommand parsing. Internally uses the same protocol package as the server — the client and server share `internal/protocol`.

---

## Layer 8 — Next.js Dashboard

**File:** `web/` (separate Next.js project)

Documented in full in [`FRONTEND.md`](FRONTEND.md).

The dashboard talks exclusively to the HTTP API — it never touches the TCP server directly. This is the correct separation: the browser speaks HTTP/WebSocket, the binary TCP protocol is for machine clients (CLI, benchmarks, other services).

---

## Layer 9 — Raft Consensus

**File:** `internal/raft/`

Documented in full in [`RAFT.md`](RAFT.md).

Raft makes a cluster of KV store nodes behave as a single system even when nodes crash. Only the leader node accepts writes. Followers redirect write requests to the leader. A command is only confirmed to the client after a majority of nodes have written it to their logs.

---

## Key Design Decisions

| Decision | Choice | Why |
|----------|--------|-----|
| Concurrency primitive | `sync.RWMutex` | Explicit, understandable, upgradeable to sharded |
| TTL algorithm | Min-heap + lazy check | O(log n) eviction, no O(n) scans |
| AOF sync mode | Every 1 second | Balance of durability and throughput |
| Snapshot format | `encoding/gob` | Zero dependencies, fast, Go-native |
| HTTP router | `chi` | Lightweight, idiomatic, composable middleware |
| Frontend framework | Next.js 15 App Router | Modern patterns, TypeScript, server components |
| Real-time transport | WebSocket | Native browser support, bidirectional, low overhead |
| Binary protocol framing | Fixed-width length prefix | Simple to parse, no delimiter scanning |
| Inter-node RPC | HTTP/JSON | Simple, debuggable; replace with gRPC as stretch goal |
