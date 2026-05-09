# ⚡ KVStore

> A production-grade distributed key-value store built in Go, featuring a 16-shard concurrent engine, persistence mechanisms, a Next.js real-time dashboard, Raft consensus clustering, and a full Prometheus + Grafana monitoring stack.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Status](https://img.shields.io/badge/status-complete-brightgreen)
![Go](https://img.shields.io/badge/go-1.23-00ADD8.svg)
![Docker](https://img.shields.io/badge/docker-compose-2496ED.svg)

---

## 📌 System Architecture

KVStore is a complete, multi-layer database system built from scratch, utilizing the same concepts that power industry standards like Redis, etcd, and CockroachDB.

```text
┌─────────────────────────────────────────────────────────────────────┐
│                        YOUR COMPLETE SYSTEM                         │
│                                                                     │
│   Browser Dashboard (Next.js 15)   :3000                            │
│          │  HTTP + WebSocket                                         │
│   HTTP API Server (:8080)                                            │
│          │                                                           │
│   CLI Client  (./kvcli)                                              │
│          │  TCP Binary Protocol                                      │
│   TCP Server (:6379)                                                 │
│          │                                                           │
│   ┌──────▼──────────────────────────────────────────────────┐       │
│   │                   CORE ENGINE                           │       │
│   │   16-Shard HashMap · FNV-1a routing · Per-shard RWMutex│       │
│   └──────────────┬──────────────────────────────────────────┘       │
│                  │                                                   │
│   ┌──────────────▼──────────────────────────────────────────┐       │
│   │               PERSISTENCE LAYER                         │       │
│   │        AOF Writer  │  Snapshot (RDB / gob)              │       │
│   └──────────────┬──────────────────────────────────────────┘       │
│                  │                                                   │
│   ┌──────────────▼──────────────────────────────────────────┐       │
│   │                  RAFT CLUSTER                            │       │
│   │        Node A ◄──► Node B ◄──► Node C                   │       │
│   └──────────────┬──────────────────────────────────────────┘       │
│                  │                                                   │
│   ┌──────────────▼──────────────────────────────────────────┐       │
│   │              MONITORING STACK                            │       │
│   │   Prometheus (:9090)  ◄──  /metrics on every node       │       │
│   │   Grafana (:3001)     ◄──  auto-provisioned dashboard   │       │
│   └─────────────────────────────────────────────────────────┘       │
└─────────────────────────────────────────────────────────────────────┘
```

---

## ✨ Features

- **Blazing Fast 16-Shard Store**: Keys hash via FNV-1a into 16 independent shards, each with its own `sync.RWMutex`. Eliminates global lock contention — throughput scales linearly with CPU cores.
- **TTL & Expiry**: Native support for expiring keys using an efficient per-shard min-heap. Each shard runs its own background eviction goroutine — zero cross-shard coordination.
- **Binary TCP Protocol**: Custom binary protocol server running on `:6379` for ultra-low latency CLI and application clients.
- **REST & WebSocket API**: Built-in HTTP server (`:8080`) providing a RESTful API and real-time event streaming for UI integrations.
- **Data Persistence**: Robust Append-Only File (AOF) logging ensuring zero data loss upon crashes or restarts.
- **Distributed Consensus (Raft)**: Highly available clustering supporting automatic leader election and log replication across 3 nodes.
- **Real-time Dashboard**: A stunning, dark-terminal aesthetic Next.js 15 frontend to monitor metrics, stream events, and manage keys live.
- **Full Observability**: Prometheus metrics (`/metrics` on every node) + Grafana dashboard with ops/sec rate graph, latency heatmap, and key count — all auto-provisioned, zero manual setup.

---

## 🛠️ Technology Stack

| Component | Technology | Description |
|-----------|-----------|----------------|
| **Core Engine** | Go | 16-shard FNV hash map, per-shard RWMutex, min-heap TTL |
| **TCP Server** | Go `net` package | Binary protocol, connection pooling, goroutines |
| **Persistence** | Go `os`, `encoding/gob` | AOF logs, atomic file writes, crash recovery |
| **CLI Client** | Go + `cobra` | Protocol design, terminal UX, REPL |
| **HTTP API** | Go `chi` router | REST design, WebSocket routing, middleware |
| **Web Dashboard** | Next.js 15, TypeScript | App Router, React 19, pure server-side rendering |
| **UI Components** | Tailwind v4, shadcn/ui | Premium, high data-density "terminal" aesthetics |
| **State Management** | Tanstack Query | Optimized client-side data fetching and cache invalidation |
| **Clustering** | Go + Raft | Distributed consensus, leader election, quorum |
| **Metrics** | Prometheus client_golang | `kvstore_commands_total`, `kvstore_keys_total`, latency histogram |
| **Visualization** | Grafana | Auto-provisioned dashboard — ops/sec rate + latency heatmap |

---

## 🚀 Getting Started

### Option A — Full Docker Cluster (Recommended)

Starts all 3 Raft nodes, the Next.js dashboard, Prometheus, and Grafana in one command:

```bash
docker compose up --build -d
```

| Service | URL | Credentials |
|---------|-----|-------------|
| Next.js Dashboard | http://localhost:3000 | — |
| Grafana | http://localhost:3001 | admin / admin |
| Prometheus | http://localhost:9090 | — |
| Node1 HTTP API | http://localhost:8080 | — |
| Node1 TCP | localhost:6379 | — |

```bash
# Tail all logs
docker compose logs -f

# Stop and remove all volumes
docker compose down -v
```

### Option B — Local Dev (Single Node)

```bash
# 1. Start the backend server
cd server
go run cmd/server/main.go
# TCP :6379, HTTP :8080

# 2. Use the CLI client (new terminal)
./kvcli
# kvstore> SET mykey hello
# kvstore> GET mykey
# kvstore> INCR counter

# 3. Start the Next.js dashboard (new terminal)
cd web
npm install
npm run dev
# Open http://localhost:3000
```

---

## 📊 Benchmark Results

Measured on **12th Gen Intel i7-12650H** (16 logical cores) with `go test -bench=. -benchtime=5s ./internal/store/...` using `b.RunParallel` (GOMAXPROCS=16):

| Benchmark | Parallelism | ns/op | Throughput |
|-----------|-------------|-------|-----------|
| `BenchmarkSet-16` | 16 goroutines | 91.15 ns | **~11M ops/sec** |
| `BenchmarkGet-16` | 16 goroutines | 19.47 ns | **~51M ops/sec** |
| `BenchmarkMixed-16` | 16 goroutines | 135.0 ns | **~7.4M ops/sec** |

> The 16-shard FNV design scales linearly with CPU cores — **27× above the 400k/sec target**. Get is faster than Set because reads only acquire an `RLock`, allowing unlimited concurrent readers within each shard.

Run it yourself:
```bash
cd server
go test -race -count=1 ./internal/store/...          # correctness + race detector
go test -bench=. -benchtime=5s ./internal/store/...  # throughput
```

---

## 🔮 Roadmap

### v1.1 — Hardening
- [ ] **TLS support** — encrypt all node-to-node and client-to-server traffic
- [ ] **Authentication** — API key / token-based auth for the HTTP API and TCP server
- [ ] **Automatic snapshotting** — periodic RDB snapshots with configurable interval
- [ ] **Raft log compaction** — trim the Raft log after snapshots to bound disk usage
- [ ] **Graceful node removal** — dynamic cluster membership changes without downtime

### v1.2 — Operations
- [ ] **`kvstore-operator`** — Kubernetes operator for deploying and managing clusters
- [ ] **Helm chart** — one-command Kubernetes deployment
- [ ] **Config file support** — YAML/TOML config as an alternative to env vars
- [ ] **Admin HTTP API** — cluster health, node promotion, live config reload

### v2.0 — Scale-out
- [ ] **Hash-based sharding across nodes** — distribute keyspace across N nodes (not just replication)
- [ ] **Multi-region replication** — async cross-region replication with conflict resolution
- [ ] **Pub/Sub channels** — Redis-style publish/subscribe messaging
- [ ] **Lua scripting** — atomic multi-command scripts executed server-side
- [ ] **Secondary indexes** — query by value ranges, not just exact key lookups

