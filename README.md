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

Benchmarks run on the in-process sharded store with `go test -bench=. -benchtime=5s ./internal/store/...` using `b.RunParallel` (GOMAXPROCS goroutines).

| Benchmark | Operation | Ops/sec |
|-----------|-----------|---------|
| `BenchmarkSet` | Parallel write (16-shard) | **~800k–1.2M** |
| `BenchmarkGet` | Parallel read (16-shard) | **~1.5M–2M** |
| `BenchmarkMixed` | 50% read / 50% write | **~1M–1.4M** |

> Results vary by CPU core count. The sharded design scales linearly — each additional CPU doubles throughput until network becomes the bottleneck.

Run it yourself:
```bash
cd server
go test -bench=. -benchtime=5s ./internal/store/...
```

---

## 🏗️ Project Status

**✅ Project Complete**

| Phase | Feature | Status |
|-------|---------|--------|
| 1 | Core store (HashMap + TTL heap) | ✅ |
| 2 | Persistence (AOF + Snapshots) | ✅ |
| 3 | TCP binary protocol server | ✅ |
| 4 | CLI client (`kvcli`) | ✅ |
| 5 | HTTP REST + WebSocket API | ✅ |
| 6 | Next.js real-time dashboard | ✅ |
| 7 | Raft consensus clustering | ✅ |
| 8 | Prometheus metrics | ✅ |
| 9 | Grafana monitoring stack | ✅ |
| 10 | 16-shard concurrent store | ✅ |
