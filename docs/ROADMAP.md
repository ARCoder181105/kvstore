# Roadmap

> Week-by-week schedule. Each week has a specific deliverable and a concrete test that confirms it is done.

---

## Overview

```
Weeks 1–2    Core engine — the database itself
Weeks 3–4    TCP server — binary protocol, connections
Weeks 5–6    Persistence — AOF + snapshots, crash recovery
Week 7       CLI client — terminal REPL
Week 8       HTTP API — REST endpoints
Week 9       Next.js dashboard — browser UI + WebSocket
Weeks 10–13  Raft clustering — distributed consensus
Week 14      Observability — metrics, profiling, benchmarks
```

Total: 14 weeks at a focused pace (5–10 hours/week).

---

## Week 1 — Storage Core

**Build:** `internal/store/store.go`

| Task | Output |
|------|--------|
| Create Go module, set up project skeleton | `go.mod`, directory structure |
| Implement `Entry` struct with `Value []byte` and `ExpiresAt int64` | Core data type |
| Implement `Store` with `sync.RWMutex` and `map[string]*Entry` | Thread-safe map |
| Implement `Set`, `Get`, `Delete` | Three core operations |
| Implement `Keys` with glob pattern matching | List operations |
| Implement `Incr` | Atomic integer increment |
| Write unit tests for all operations | `store_test.go` |

**End-of-week test:**
```bash
go test ./internal/store/... -v
# All tests pass
```

---

## Week 2 — TTL System

**Build:** `internal/store/ttl.go`

| Task | Output |
|------|--------|
| Implement `TTLHeap` with `heap.Interface` | Min-heap data structure |
| Wire `TTLHeap` into `Store` struct | Heap updates on every Set |
| Implement background eviction goroutine | Keys deleted at expiry |
| Implement lazy eviction in `Get` | Expired keys never returned |
| Add TTL-specific unit tests | Test keys expire correctly |
| Run with `-race` flag | Zero data race warnings |

**End-of-week test:**
```bash
go test ./internal/store/... -race -count=3
# PASS — all three runs clean
```

---

## Week 3 — Binary Protocol

**Build:** `internal/protocol/`

| Task | Output |
|------|--------|
| Define command ID constants and command/response structs | `commands.go` |
| Implement binary frame encoder | `serializer.go` |
| Implement binary frame decoder | `parser.go` |
| Write round-trip encode/decode tests | `protocol_test.go` |

**End-of-week test:**
```bash
go test ./internal/protocol/... -v
# Every encode→decode round trip produces identical structs
```

---

## Week 4 — TCP Server

**Build:** `internal/server/`, `cmd/server/main.go`, `cmd/kvcli/main.go` (basic version)

| Task | Output |
|------|--------|
| Implement `Server` with `net.Listen` accept loop | TCP listener |
| Implement `handleConn` per-connection goroutine | Connection handler |
| Wire commands to store operations | Commands execute correctly |
| Implement basic CLI client (args only, no REPL) | `./kvcli SET GET DEL` |
| Write server integration tests | `server_test.go` |

**End-of-week test:**
```bash
# Terminal 1
go run cmd/server/main.go

# Terminal 2
go run cmd/kvcli/main.go SET hello world  # → OK
go run cmd/kvcli/main.go GET hello        # → world
go run cmd/kvcli/main.go DEL hello        # → OK
go run cmd/kvcli/main.go GET hello        # → (nil)
```

---

## Week 5 — AOF Persistence

**Build:** `internal/persistence/aof.go`

| Task | Output |
|------|--------|
| Define AOF binary entry format | Documented in `PROTOCOLS.md` |
| Implement `AOFWriter` goroutine with buffered channel | Async writes to disk |
| Implement `Replay` — read log file, re-execute commands | Crash recovery |
| Wire AOF into server — every write appends an entry | Persistent writes |
| Write AOF tests | Replay test |

**End-of-week test:**
```bash
./kvcli SET name Alice
./kvcli SET city Mumbai
# Kill server
go run cmd/server/main.go  # restart
./kvcli GET name  # → Alice
./kvcli GET city  # → Mumbai
```

---

## Week 6 — Snapshot Persistence

**Build:** `internal/persistence/snapshot.go`

| Task | Output |
|------|--------|
| Implement `Save` — gob-encode store to temp file, rename atomically | `snapshot.db` file |
| Implement `Load` — gob-decode into store on startup | Restore from snapshot |
| Wire startup sequence: load snapshot → replay AOF delta | Full recovery |
| Implement scheduled snapshot (configurable interval) | Auto-snapshots |
| Write snapshot tests | Save → load → verify |

**End-of-week test:**
```bash
go test ./internal/persistence/... -v -race
# PASS — snapshot and AOF tests pass cleanly
```

---

## Week 7 — CLI Client Polish

**Build:** Refactor `cmd/kvcli/main.go` with `cobra` + REPL

| Task | Output |
|------|--------|
| Add `cobra` subcommands: set, get, del, expire, ttl, keys, incr | Proper CLI UX |
| Implement REPL mode — `kvstore>` prompt | Interactive terminal |
| Add `--host` and `--port` flags | Configurable connection |
| Add `HELP` command listing all commands | Discoverability |
| Test REPL manually end-to-end | Demo-ready CLI |

**End-of-week test:**
```bash
./kvcli
kvstore> SET counter 0
OK
kvstore> INCR counter
1
kvstore> INCR counter
2
kvstore> KEYS *
1) counter
kvstore> EXIT
bye
```

---

## Week 8 — HTTP API

**Build:** `internal/api/`

| Task | Output |
|------|--------|
| Add `chi` router + `gorilla/websocket` dependencies | Dependencies ready |
| Implement all REST handlers | 8 endpoints working |
| Implement WebSocket event stream endpoint | `/ws/events` |
| Add store event emission on Set/Del/Expire | Events flow to subscribers |
| Add CORS middleware (for browser requests) | Dashboard can connect |
| Write handler tests with `httptest` | `api_test.go` |

**End-of-week test:**
```bash
curl http://localhost:8080/api/health
# → {"status":"ok"}

curl -X POST http://localhost:8080/api/keys/lang \
  -H "Content-Type: application/json" -d '{"value":"Go"}'
# → {"key":"lang","value":"Go","ttl":-1}

curl http://localhost:8080/api/stats
# → {"total_keys":1,"uptime":"0h 0m 12s","aof_size_bytes":48}
```

---

## Week 9 — Next.js Dashboard

**Build:** `web/` — complete frontend

| Task | Output |
|------|--------|
| Init Next.js 15 project with TypeScript + Tailwind | Project scaffolded |
| Install shadcn/ui + Tanstack Query | UI components + data fetching ready |
| Build `lib/api.ts` — typed HTTP client | Typed fetch wrappers |
| Build `hooks/useKeys.ts` + `useStats.ts` | Data fetching hooks |
| Build `hooks/useEventStream.ts` — WebSocket with reconnect | Live events hook |
| Build `StatsPanel` + `KeysTable` + `EventStream` components | Core UI done |
| Build `AddKeyForm` with TTL slider | Key creation UI |
| Assemble `app/page.tsx` dashboard | Full dashboard running |

**End-of-week test:**
- Open browser → see all keys, stats, and live event stream
- Add a key in the UI → appears in the table immediately
- Delete a key → disappears from table, DEL event in stream
- Kill the Go server → connection badge turns red

---

## Weeks 10–13 — Raft Consensus

Detailed breakdown in [`RAFT.md`](RAFT.md).

| Week | Build |
|------|-------|
| 10 | Node struct, state machine, term numbers, persistent vote state |
| 11 | `RequestVote` RPC, election timeout, vote counting |
| 12 | `AppendEntries` RPC, log replication, heartbeats |
| 13 | Commit index, applying entries to store, leader proxy |

**End-of-week-13 test:**
```bash
# Start 3 nodes, kill the leader, verify election under 3 seconds
```

---

## Week 14 — Observability and Final Polish

| Task | Output |
|------|--------|
| Add Prometheus metrics (commands_total, keys_total, latency histogram) | `/metrics` endpoint |
| Enable `pprof` endpoint | `/debug/pprof/` |
| Write `bench/bench.go` | Throughput benchmark |
| Write `README.md` with setup instructions | GitHub-ready |
| Add `docker-compose.yml` for 3-node cluster | One-command demo setup |

**End-of-week test:**
```bash
go run bench/bench.go --connections 50 --duration 10s
# Throughput: >100,000 ops/sec
```

---

## Milestone Summary

| Milestone | Week | Proof |
|-----------|------|-------|
| Core engine correct and race-free | 2 | `go test -race` PASS |
| TCP client-server works | 4 | CLI SET/GET round-trip |
| Data survives restarts | 6 | Kill → restart → keys present |
| REPL terminal client working | 7 | Interactive demo |
| HTTP API complete | 8 | `curl` all endpoints |
| Browser dashboard live | 9 | Real-time event stream visible |
| Raft cluster fault-tolerant | 13 | Leader fail → re-elect < 3s |
| Benchmark >100k ops/sec | 14 | Bench output confirms it |
