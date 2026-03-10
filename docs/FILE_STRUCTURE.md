# File Structure

> Complete file tree for every package — Go backend and Next.js frontend. Every file listed with its purpose.

---

## Go Backend

```
kvstore/                          ← Git root / Go module root
│
├── cmd/                          ← Executable entry points (main packages)
│   ├── server/
│   │   └── main.go               ← Starts TCP server + HTTP API + AOF + TTL goroutines
│   └── kvcli/
│       └── main.go               ← CLI client — single command or REPL mode
│
├── internal/                     ← Private packages (not importable by outside code)
│   │
│   ├── store/
│   │   ├── store.go              ← Core KV engine: Store struct, SET/GET/DEL/INCR/KEYS
│   │   ├── ttl.go                ← TTLHeap (min-heap), background eviction goroutine
│   │   ├── events.go             ← Event type, event channel, subscriber fan-out
│   │   └── store_test.go         ← Unit tests: correctness + go test -race
│   │
│   ├── protocol/
│   │   ├── commands.go           ← Command ID constants, Command/Response structs
│   │   ├── parser.go             ← Binary frame → Command struct (decode)
│   │   ├── serializer.go         ← Command/Response struct → binary frame (encode)
│   │   └── protocol_test.go      ← Round-trip encode/decode tests
│   │
│   ├── server/
│   │   ├── server.go             ← TCP net.Listener, Accept loop, graceful shutdown
│   │   ├── handler.go            ← Per-connection goroutine: read → parse → execute → respond
│   │   └── server_test.go        ← Integration tests: real TCP connections
│   │
│   ├── persistence/
│   │   ├── aof.go                ← AOF writer goroutine, entry format, replay on startup
│   │   ├── snapshot.go           ← Full state serialize/deserialize with encoding/gob
│   │   └── persistence_test.go   ← Write → kill → restore tests
│   │
│   ├── api/
│   │   ├── server.go             ← chi router setup, middleware (CORS, logging, recovery)
│   │   ├── handlers.go           ← HTTP handler funcs for every REST endpoint
│   │   ├── ws.go                 ← WebSocket upgrade, event fan-out to browser clients
│   │   └── api_test.go           ← HTTP handler tests with httptest
│   │
│   └── raft/                     ← Phase 4 — add after single-node is solid
│       ├── node.go               ← RaftNode struct, state machine (Follower/Candidate/Leader)
│       ├── log.go                ← LogEntry struct, in-memory log, commit index
│       ├── rpc.go                ← RequestVote + AppendEntries HTTP handlers and client calls
│       └── election.go           ← Election timer, vote counting, term management
│
├── config/
│   └── config.go                 ← YAML config loader (port, AOF path, sync mode, node peers)
│
├── bench/
│   └── bench.go                  ← Benchmark: measures SET/GET ops/sec against running server
│
├── go.mod                        ← Module: github.com/yourname/kvstore
├── go.sum
├── Makefile                      ← build, test, run, bench, lint targets
└── .env.example                  ← Example config: ports, AOF path, snapshot interval
```

---

## Next.js Frontend

```
web/                              ← Next.js project root (separate from Go module)
│
├── app/                          ← Next.js 15 App Router
│   ├── layout.tsx                ← Root layout: font, global providers, metadata
│   ├── page.tsx                  ← Dashboard home — keys table + stats + event stream
│   ├── globals.css               ← Tailwind base + CSS variables
│   │
│   └── api/                      ← Next.js route handlers (proxy layer to Go API)
│       └── health/
│           └── route.ts          ← Ping the Go server to check connectivity
│
├── components/                   ← All UI components
│   ├── ui/                       ← shadcn/ui primitive components (auto-generated)
│   │   ├── button.tsx
│   │   ├── input.tsx
│   │   ├── table.tsx
│   │   ├── badge.tsx
│   │   ├── card.tsx
│   │   ├── dialog.tsx
│   │   ├── slider.tsx
│   │   └── toast.tsx
│   │
│   ├── dashboard/
│   │   ├── StatsPanel.tsx        ← Total keys, memory usage, uptime cards
│   │   ├── KeysTable.tsx         ← Sortable table: key / value / TTL / actions
│   │   ├── KeyRow.tsx            ← Single row with inline edit + delete + TTL slider
│   │   ├── AddKeyForm.tsx        ← Form to set a new key with optional TTL
│   │   ├── SearchBar.tsx         ← Filter keys by prefix or pattern
│   │   └── EventStream.tsx       ← Live WebSocket event log (auto-scroll)
│   │
│   └── layout/
│       ├── Header.tsx            ← Title + connection status indicator
│       └── ConnectionBadge.tsx   ← Green/red dot showing WebSocket state
│
├── hooks/
│   ├── useKeys.ts                ← Tanstack Query: fetch + poll /api/keys
│   ├── useStats.ts               ← Tanstack Query: fetch /api/stats
│   ├── useEventStream.ts         ← Native WebSocket hook with reconnect logic
│   └── useKeyMutations.ts        ← Tanstack Query mutations: set, delete, expire
│
├── lib/
│   ├── api.ts                    ← Typed fetch wrappers for every HTTP endpoint
│   ├── types.ts                  ← TypeScript interfaces: KeyEntry, Stats, Event
│   └── utils.ts                  ← formatTTL, formatBytes, cn() (tailwind merge)
│
├── providers/
│   └── QueryProvider.tsx         ← Tanstack Query client provider (client component)
│
├── public/
│   └── favicon.ico
│
├── tailwind.config.ts
├── tsconfig.json
├── next.config.ts
├── components.json               ← shadcn/ui config
├── package.json
└── .env.local.example            ← NEXT_PUBLIC_API_URL, NEXT_PUBLIC_WS_URL
```

---

## Makefile Targets

```makefile
make build          # compile both server and kvcli binaries
make test           # go test ./... with -race flag
make run            # start the server (TCP + HTTP)
make bench          # run the benchmark against a running server
make lint           # golangci-lint run
make snapshot       # trigger a manual snapshot via HTTP API
make clean          # remove build artifacts
```

---

## Config File

```yaml
# config.yaml
server:
  tcp_port: 6379
  http_port: 8080

persistence:
  aof_path: ./data/aof.log
  snapshot_path: ./data/snapshot.db
  aof_sync: interval          # options: always | interval | never
  sync_interval_seconds: 1
  snapshot_interval_minutes: 60

raft:
  enabled: false
  node_id: node-1
  peers:
    - http://localhost:7001
    - http://localhost:7002
```

---

## What Each `internal/` Package Owns

| Package | Owns | Does NOT own |
|---------|------|--------------|
| `store` | Data, TTL, events | Network, disk |
| `protocol` | Encoding/decoding binary frames | Network I/O |
| `server` | TCP connections, goroutines | Data storage |
| `persistence` | Disk reads/writes | In-memory state |
| `api` | HTTP/WebSocket routing | Data storage |
| `raft` | Consensus state machine | Storage engine directly |

This separation means you can test each package in isolation with no real network or disk required.
