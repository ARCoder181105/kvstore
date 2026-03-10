# ⚡ KVStore — Build Your Own Redis

> A production-grade distributed key-value store built from scratch in Go, with a Next.js dashboard, CLI client, and Raft consensus clustering.

---

```
┌─────────────────────────────────────────────────────────────┐
│                    YOUR COMPLETE SYSTEM                      │
│                                                             │
│   Browser Dashboard (Next.js 15)                            │
│          │  HTTP + WebSocket                                 │
│   HTTP API Server (:8080)                                   │
│          │                                                  │
│   CLI Client  (./kvcli)                                     │
│          │  TCP Binary Protocol                              │
│   TCP Server (:6379)                                        │
│          │                                                  │
│   ┌──────▼──────────────────────────────────┐               │
│   │             CORE ENGINE                 │               │
│   │    HashMap + TTL Heap + RWMutex         │               │
│   └──────────────┬──────────────────────────┘               │
│                  │                                          │
│   ┌──────────────▼──────────────────────────┐               │
│   │         PERSISTENCE LAYER               │               │
│   │    AOF Writer  │  Snapshot (RDB)        │               │
│   └─────────────────────────────────────────┘               │
│                  │                                          │
│   ┌──────────────▼──────────────────────────┐               │
│   │           RAFT CLUSTER                  │               │
│   │   Node A ◄──► Node B ◄──► Node C        │               │
│   └─────────────────────────────────────────┘               │
└─────────────────────────────────────────────────────────────┘
```

---

## What You Are Building

A complete, multi-layer database system — the same concepts that power Redis, etcd, and CockroachDB — built entirely by you.

| Layer | Technology | What You Learn |
|-------|-----------|----------------|
| Core Engine | Go | Concurrency, data structures, memory layout |
| TCP Server | Go `net` package | Binary protocols, connection pooling, goroutines |
| Persistence | Go `os`, `encoding/gob` | AOF logs, atomic file writes, crash recovery |
| CLI Client | Go + `cobra` | Protocol design, terminal UX, REPL |
| HTTP API | Go `chi` router | REST design, WebSocket, middleware |
| Web Dashboard | Next.js 15, TypeScript | App Router, real-time UI, Tanstack Query |
| Clustering | Go + Raft | Distributed consensus, leader election, quorum |

---

## Quick Navigation

| Document | Description |
|----------|-------------|
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | Full system design — every layer explained with diagrams |
| [`docs/IMPLEMENTATION_PLAN.md`](docs/IMPLEMENTATION_PLAN.md) | Phase-by-phase build order with exact tasks per phase |
| [`docs/ROADMAP.md`](docs/ROADMAP.md) | Week-by-week timeline, milestones, and success criteria |
| [`docs/TECH_STACK.md`](docs/TECH_STACK.md) | Every library and tool — what it is and why it was chosen |
| [`docs/FILE_STRUCTURE.md`](docs/FILE_STRUCTURE.md) | Complete file tree for Go backend and Next.js frontend |
| [`docs/PROTOCOLS.md`](docs/PROTOCOLS.md) | Binary TCP protocol spec + HTTP API + WebSocket contracts |
| [`docs/DATA_STRUCTURES.md`](docs/DATA_STRUCTURES.md) | Every struct, algorithm, and concurrency pattern |
| [`docs/RAFT.md`](docs/RAFT.md) | Raft consensus deep-dive and step-by-step implementation |
| [`docs/FRONTEND.md`](docs/FRONTEND.md) | Next.js dashboard — pages, components, hooks, real-time |
| [`docs/RESOURCES.md`](docs/RESOURCES.md) | Books, papers, videos, and references for every phase |

---

## Success Milestones

```
Phase 1 done:  go test -race passes — 100k ops, zero data races
Phase 2 done:  TCP client sends binary frame → server replies correctly
Phase 3 done:  Kill server → restart → all keys restored from AOF log
Phase 4 done:  ./kvcli REPL — kvstore> SET name Alice → OK
Phase 5 done:  HTTP API returns correct JSON for every endpoint
Phase 6 done:  Dashboard shows live key table + real-time event stream
Phase 7 done:  Kill Raft leader → new leader elected in under 3 seconds
Phase 8 done:  Benchmark confirms > 100k ops/sec on a single node
```

---

## The Demo You Will Show

1. Open the browser dashboard
2. Type a key-value pair into the UI — it appears in the live event stream instantly
3. Open terminal, run `./kvcli GET yourkey` — it is there
4. Kill the server with `Ctrl+C`
5. Restart it — every key comes back from the AOF log
6. Start 3 nodes, kill the leader — a new one is elected in under 3 seconds

Every single line of that system is code you wrote.

---

*Golang Deep-Dive Series — Project 01 of 05*
