# 📘 KVStore Learning Guide & Implementation Workbook

Welcome to your learning guide for the KVStore project! The primary goal of this project is not just to have a working database, but to **make you a better systems programmer**. 

This document serves as your interactive workbook. Instead of copy-pasting code, use this guide to understand *what* to build, *why* it matters, and *how* to approach it.

---

## 🎯 Current Project State

- **Phases 1-6:** ✅ **Complete.** Core storage, TCP, persistence, API, and UI are built!
- **Phase 7:** ✅ **Complete.** Raft consensus (leader election, replication) is working.
- **Phase 8:** 🚧 **In Progress.** Observability and Benchmarks. This is your current focus!

---

## 🚀 Phase 8: Observability and Benchmarks (Your Next Challenge)

### The Goal
A production-grade database is flying blind without metrics. In this phase, you will write a Prometheus integration to track performance and a benchmarking tool to prove your system can handle the load.

**Rule of Thumb:** Do not copy code blindly. Read the hint provided, try implementing it in the file yourself, and then test.

---

### Step 1: Prepare Your Workspace
Before you write logic, set up the standard library tools used by the industry.
- **Task:** Add the Prometheus Go client to your project.
- **Command to run in `server/`:** 
  ```bash
  go get github.com/prometheus/client_golang/prometheus
  go get github.com/prometheus/client_golang/prometheus/promhttp
  ```

### Step 2: Define the Metrics (`server/internal/metrics/metrics.go`)
You need a dedicated package where all your metrics are strongly typed and globally accessible.

- **Task:** Create `metrics.go` inside the `metrics` package and define three variables using the `prometheus` module.
  - **1. CommandsTotal (CounterVec):** Name it `kvstore_commands_total`. Include a label named `"command"`.
  - **2. KeysTotal (Gauge):** Name it `kvstore_keys_total`. No labels required.
  - **3. CommandDurationSeconds (HistogramVec):** Name it `kvstore_command_duration_seconds`. Use buckets: `.0001, .0005, .001, .005, .01, .05, .1`. Include a label named `"command"`.
- **Task:** Write a single `func Register()` in this file that calls `prometheus.MustRegister` with all three of your metrics.
- 💡 **Pedagogical Hint:** Why use a *Vector* for counters and histograms? Because it allows us to group latencies and counts dynamically by command type (e.g., SET vs GET), giving us rich insights in Grafana!

### Step 3: Instrument the TCP Server (`server/internal/server/handler.go`)
Your database needs to record metrics exactly when it does work. 

- **Task:** Open `executeCommand`. Record the start time at the very top using `time.Now()`.
- **Task:** Use an anonymous `defer` function right after the start time. Inside that defer:
  1. Record duration in the histogram using `time.Since(...)`.
  2. Increment the counter.
- 💡 **Pedagogical Hint:** Why use `defer` here instead of writing it at the bottom of the function? Notice how many `return` statements exist in your big `switch` statement. A `defer` guarantees execution no matter which path the code takes to return, preventing dropped metrics!

### Step 4: Instrument the Raft State Machine (`server/internal/raft/node.go`)
Your TCP server handles single-node direct connections, but what about Raft-replicated operations across the cluster?

- **Task:** Navigate to `applyCommitted()`. This is where all committed Raft logs actually hit the store.
- **Task:** Apply the exact same pattern as Step 3 for any command processed here (incrementing the counter).
- **Task:** Every time a batch of commands is applied in `applyCommitted`, update your `KeysTotal` gauge with the current total count from the store.
- 💡 **Pedagogical Hint:** Why measure here? Because followers never execute logic in `executeCommand`—they only get data via Raft. If you only instrumented the TCP layer, you'd be missing metrics on all your Raft followers!

### Step 5: Start the Metrics Engine
Now that metrics are being tracked in memory, expose them so a Prometheus scraper can fetch them.

- **Task:** In `api/server.go` (where your HTTP routes are mounted), mount the prometheus handler: `r.Handle("/metrics", promhttp.Handler())`.
- **Task:** Open `cmd/server/main.go`. Before your server starts running, call your `metrics.Register()` function exactly once to wire up your custom metrics to the registry.

### Step 6: Expose Pprof (Go Profiling)
To find out why a Go program is slow, we use `pprof`.
- **Task:** In `cmd/server/main.go`, add this specific, blank import:
  ```go
  import _ "net/http/pprof"
  ```
- 💡 **Pedagogical Hint:** The `_` (blank identifier) runs the package's `init()` function without you needing to call its methods. For `pprof`, its `init()` automatically injects CPU/Memory profiling endpoints into the default HTTP mux.

### Step 7: Build the Benchmark Tool (`bench/bench.go`)
You claim this database is fast. Prove it.

- **Task:** Create a standalone Go program (with its own `main` package) in a new `bench/bench.go` file.
- **Task:** Setup command line flags: `--connections` (default 50), `--duration` (default 10s), etc.
- **Task:** Spawn N goroutines. Each one should establish *one* continuous TCP connection. In a `for` loop, each goroutine should pick a random string, send a `SET` command, then `GET` the same key.
- **Task:** Keep a global atomic counter of operations. Wait until `--duration` ends, then divide the counter by the seconds to print `ops/sec`.
- 💡 **Pedagogical Hint:** In benchmarks, locks (Mutexes) can be a bottleneck. Track your total operations using `sync/atomic` (like `atomic.AddInt64`). This uses lock-free hardware instructions and ensures your benchmark tool isn't slowing down your test! Also make sure your client uses one persistent connection rather than reconnecting per operation, otherwise you end up benchmarking the OS's handshake speed rather than your KV-store logic.

---

### Step 8: Visualize with Grafana & Prometheus (Bonus Challenge)
Raw text metrics at `/metrics` are great, but industry standards use dashboards for real-time monitoring.

- **Task:** Update your `docker-compose.yml` to include the official `prom/prometheus` and `grafana/grafana` images.
- **Task:** Create a `prometheus.yml` config file and mount it into your Prometheus container. Configure it to scrape all 3 of your KVStore Raft nodes (`node1:8080`, `node2:8080`, `node3:8080`).
- **Task:** Boot up Grafana on port `3001` (to avoid conflicting with Next.js), connect Prometheus as a data source, and build a beautiful dashboard showing your `kvstore_command_duration_seconds` heatmap and `kvstore_commands_total` rates!
- 💡 **Pedagogical Hint:** Using rate functions in Grafana like `rate(kvstore_commands_total[1m])` converts your raw, ever-growing counter into a clean "Operations per Second" graph!

---

### ✅ Phase 8 Definition of Done Check
- Can you `curl localhost:8080/metrics` and see raw Prometheus text output?
- Are the counters accumulating correctly when you run operations via the CLI?
- Does your `bench.go` run successfully and output a >100,000 throughput rate?
- **(Bonus)** Do you have a vibrant Grafana dashboard rendering your Prometheus data in real-time?

Whenever you get stuck on an implementation detail for these steps, think about the Go primitives you've learned. Good luck, and start writing code!
