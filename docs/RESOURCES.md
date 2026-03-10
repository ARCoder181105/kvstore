# Resources

> Every book, paper, video, and reference you need — organized by phase.

---

## Phase 1 — Go + Core Engine

### Must Read

**Tour of Go** — https://go.dev/tour
The official interactive Go tutorial. Do the entire tour before writing your first line of this project. Pay special attention to the concurrency section (goroutines, channels, select).

**Go by Example** — https://gobyexample.com
Code snippets for every Go concept you will use: goroutines, channels, mutexes, timers, file I/O. Bookmark this — you will reference it constantly.

**Effective Go** — https://go.dev/doc/effective_go
Google's guide to idiomatic Go. Read the sections on goroutines, channels, and error handling. This is what separates code that works from code that is good.

### Data Structures

**`container/heap` package** — https://pkg.go.dev/container/heap
Read the full docs including the example. The priority queue example is almost exactly the TTL heap you will build.

**sync package** — https://pkg.go.dev/sync
`RWMutex`, `Mutex`, `WaitGroup`, `Once` — read the docs for all four. Understand the difference between `Lock` and `RLock`.

---

## Phase 2 — TCP + Binary Protocol

### Must Read

**Go net package** — https://pkg.go.dev/net
Focus on `net.Listen`, `net.Conn`, `net.Dial`. The examples show exactly the accept-loop pattern you will implement.

**`io.ReadFull`** — https://pkg.go.dev/io#ReadFull
This is how you read exactly N bytes from a TCP connection without partial reads. Essential for binary protocol parsing.

**`encoding/binary`** — https://pkg.go.dev/encoding/binary
`binary.BigEndian.Uint32`, `binary.BigEndian.PutUint32` — how to read and write multi-byte integers in a specific byte order.

### Reference

**Redis Protocol (RESP)** — https://redis.io/docs/reference/protocol-spec/
Not the protocol you are building, but understanding what Redis chose (and why) will help you design your own. Notice how their text protocol handles binary data via length prefixes — the same idea you will use.

**Beej's Guide to Network Programming** — https://beej.us/guide/bgnet/
Written for C, but the concepts — sockets, TCP framing, byte ordering — are universal. Chapters 1–5 and chapter 7 are the relevant ones.

---

## Phase 3 — Persistence

### Must Read

**`encoding/gob`** — https://pkg.go.dev/encoding/gob
Go's native binary serialization. Read the blog post "Gobs of data" (link below) before the package docs.

**"Gobs of data"** — https://go.dev/blog/gob
The Go team's explanation of why gob exists and how it works. 10-minute read.

**`os.Rename` atomicity** — search: "atomic file write rename trick"
The pattern: write to a temp file, then `os.Rename` to the final path. On POSIX systems (Linux, macOS), rename is atomic — either the whole file is swapped or nothing changes. This is how you avoid corrupt snapshots on crash.

### Deep Dive

**How Redis persistence works** — https://redis.io/docs/management/persistence/
Redis uses the same two strategies you are building: AOF and RDB (snapshots). Reading their documentation will give you the exact tradeoffs to understand — and explain in a demo.

---

## Phase 4 — CLI

### Must Read

**Cobra user guide** — https://github.com/spf13/cobra/blob/main/site/content/user_guide.md
The definitive guide to building CLIs with cobra. Your kvcli will follow exactly this pattern.

---

## Phase 5 — HTTP API

### Must Read

**chi router README** — https://github.com/go-chi/chi
How to set up routes, URL parameters, and middleware. The examples show the exact patterns you need.

**gorilla/websocket** — https://pkg.go.dev/github.com/gorilla/websocket
Read the chat example in the examples folder — it is exactly the fan-out pattern you will use for the event stream.

**`net/http/httptest`** — https://pkg.go.dev/net/http/httptest
How to test HTTP handlers without starting a real server. `httptest.NewRecorder()` and `httptest.NewRequest()` are the two functions you will use in every API test.

---

## Phase 6 — Next.js Frontend

### Must Read

**Next.js App Router docs** — https://nextjs.org/docs/app
Focus on: Routing, Server vs Client Components, the `use client` directive, and Route Handlers.

**Tanstack Query docs** — https://tanstack.com/query/latest/docs/framework/react/overview
Read: Overview, useQuery, useMutation, QueryClient, and the "Query Invalidation" guide. These four pages are everything you need.

**shadcn/ui docs** — https://ui.shadcn.com
Click each component you are using and read how to install and use it. The Table, Dialog, and Slider components need the most attention.

**MDN WebSocket API** — https://developer.mozilla.org/en-US/docs/Web/API/WebSocket
The browser's native WebSocket interface. Your `useEventStream` hook uses only this — no libraries.

---

## Phase 7 — Raft

### Must Read (in order)

**The Raft Paper** — https://raft.github.io/raft.pdf
Read sections 5.1–5.4 first. Then 5.5 and 5.6. The paper is 18 pages and unusually readable for a systems paper.

**Raft visualization** — https://raft.github.io
Run through the animation multiple times. Pause it at different points. Try to predict what will happen next before it does. This is the best way to build intuition for leader election and log replication.

**"Students' Guide to Raft"** — https://thesquareplanet.com/blog/students-guide-to-raft/
Written by a TA from MIT's distributed systems course. Lists every common implementation mistake. Read this before writing a single line of Raft code.

### Reference

**etcd raft library** — https://github.com/etcd-io/raft
A production Raft implementation in Go. Do not copy it — read it to understand how a real implementation handles edge cases.

**MIT 6.5840 Raft lab** — https://pdos.csail.mit.edu/6.5840/labs/lab-raft.html
The test cases listed here define exactly what your Raft implementation must handle. Your implementation should pass all of them even if you are not taking the course.

---

## Phase 8 — Observability

### Must Read

**Prometheus Go client** — https://pkg.go.dev/github.com/prometheus/client_golang/prometheus
Focus on `NewCounterVec`, `NewGaugeVec`, `NewHistogramVec`, and `MustRegister`.

**pprof profiling** — https://pkg.go.dev/net/http/pprof
Import it and you get `/debug/pprof/` for free. Then run `go tool pprof http://localhost:8080/debug/pprof/profile` to generate a flame graph.

---

## Books (worth buying if you are serious)

**"The Go Programming Language"** — Donovan & Kernighan
The authoritative Go book. Chapters 8 (goroutines) and 9 (concurrency) are essential reading for this project.

**"Designing Data-Intensive Applications"** — Martin Kleppmann
Chapter 3 covers storage engines (exactly what you are building). Chapter 7 covers transactions. Chapter 9 covers consistency and consensus (Raft). Reading the relevant chapters will give you context for every decision in this project.

**"Database Internals"** — Alex Petrov
Goes deeper on storage engine design — B-trees, LSM trees, WAL (which is what AOF is). Not required, but if you want to understand why these designs exist, this book explains it.

---

## Useful Go Tools

```bash
# Race detector — run every test with this
go test -race ./...

# CPU profiler
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=10

# Memory profiler
go tool pprof http://localhost:8080/debug/pprof/heap

# Trace (goroutine activity)
go tool trace trace.out

# Lint
golangci-lint run

# Vet (static analysis built into Go toolchain)
go vet ./...

# Benchmark a specific function
go test -bench=BenchmarkStore -benchmem ./internal/store/...
```
