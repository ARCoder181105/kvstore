# ⚡ KVStore

> A production-grade distributed key-value store built in Go, featuring a high-performance core engine, persistence mechanisms, a Next.js real-time dashboard, and Raft consensus clustering.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Status](https://img.shields.io/badge/status-under%20development-orange)

## 📌 System Architecture

KVStore is a complete, multi-layer database system built from scratch, utilizing the same concepts that power industry standards like Redis, etcd, and CockroachDB.

```text
┌─────────────────────────────────────────────────────────────┐
│                    YOUR COMPLETE SYSTEM                     │
│                                                             │
│   Browser Dashboard (Next.js 15)                            │
│          │  HTTP + WebSocket                                │
│   HTTP API Server (:8080)                                   │
│          │                                                  │
│   CLI Client  (./kvcli)                                     │
│          │  TCP Binary Protocol                             │
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

## ✨ Features

- **Blazing Fast In-Memory Storage**: Concurrent hash maps protected by `sync.RWMutex` for zero data-races and massive throughput.
- **TTL & Expiry**: Native support for expiring keys using an efficient min-heap implementation.
- **Binary TCP Protocol**: Custom binary protocol server running on `:6379` for ultra-low latency CLI and application clients.
- **REST & WebSocket API**: Built-in HTTP server (`:8080`) providing a RESTful API and real-time event streaming for UI integrations.
- **Data Persistence**: Robust Append-Only File (AOF) logging ensuring zero data loss upon crashes or restarts.
- **Distributed Consensus (Raft)**: Highly available clustering supporting automatic leader election and log replication.
- **Real-time Dashboard**: A stunning, dark-terminal aesthetic Next.js 15 frontend to monitor metrics, stream events, and manage keys live.

## 🛠️ Technology Stack

| Component | Technology | Description |
|-----------|-----------|----------------|
| **Core Engine** | Go | Concurrency, custom data structures, memory layout |
| **TCP Server** | Go `net` package | Binary protocols, connection pooling, goroutines |
| **Persistence** | Go `os`, `encoding/gob` | AOF logs, atomic file writes, crash recovery |
| **CLI Client** | Go + `cobra` | Protocol design, terminal UX, REPL |
| **HTTP API** | Go `chi` router | REST design, WebSocket routing, middleware |
| **Web Dashboard** | Next.js 15, TypeScript | App Router, React 19, pure server-side rendering |
| **UI Components** | Tailwind v4, shadcn/ui | Premium, high data-density "terminal" aesthetics |
| **State Management** | Tanstack Query | Optimized client-side data fetching and cache invalidation |
| **Clustering** | Go + Raft | Distributed consensus, leader election, quorum |

## 🚀 Getting Started

To run the full stack locally:

### 1. Start the Backend Server
```bash
cd server
go run cmd/server/main.go
# Starts the TCP server on :6379 and HTTP API on :8080
```

### 2. Start the CLI Client
```bash
# In a new terminal
cd cli
go run main.go
# Example: kvstore> SET mykey 123
```

### 3. Start the Next.js Dashboard
```bash
# In a new terminal
cd web
npm install
npm run dev
# Open http://localhost:3000 in your browser
```

## 🏗️ Project Status

**🚧 This project is currently under active development. 🚧**

KVStore is an evolving system. New updates, performance improvements, and critical features (like advanced Raft node management and automated snapshotting) are coming soon! Stay tuned.
