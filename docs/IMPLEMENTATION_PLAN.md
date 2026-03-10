# Implementation Plan

> Phase-by-phase build order. Every phase has a goal, exact tasks, the right order to do them, and a definition of done.

**The rule:** Do not start the next phase until the current one fully passes its definition of done.

---

## Phase 1 — Core Storage Engine

**Goal:** A pure in-memory KV store with no networking, no disk, just a correct and thread-safe data structure.

**Why first:** Everything else is built on this. If this has race conditions or incorrect TTL behavior, every phase above it will be broken in subtle ways.

### Tasks (do in this order)

1. Initialize the Go module
   ```bash
   mkdir kvstore && cd kvstore
   go mod init github.com/yourname/kvstore
   ```

2. Create `internal/store/store.go`
   - Define `Entry` struct: `Value []byte`, `ExpiresAt int64` (Unix nanoseconds, 0 = no expiry)
   - Define `Store` struct: `mu sync.RWMutex`, `data map[string]*Entry`
   - Implement `New() *Store`
   - Implement `Set(key string, value []byte, ttlNs int64)`
   - Implement `Get(key string) ([]byte, bool)` — check expiry on every get
   - Implement `Delete(key string) bool`
   - Implement `Keys(pattern string) []string` — glob match using `filepath.Match`
   - Implement `Incr(key string) (int64, error)` — parse value as int, increment, store back

3. Create `internal/store/ttl.go`
   - Define `TTLItem` struct: `key string`, `expiresAt int64`, `index int`
   - Define `TTLHeap` type implementing `heap.Interface` (Len, Less, Swap, Push, Pop)
   - Add `ttlHeap *TTLHeap` to `Store` struct
   - In `Set`: if `ttlNs > 0`, push to heap
   - Create background goroutine `startEviction(ctx context.Context)`: peek heap, sleep until next expiry, pop and delete when expired

4. Create `internal/store/store_test.go`
   - Test Set → Get → value matches
   - Test Delete — key not found after
   - Test TTL — set key with 100ms TTL, sleep 150ms, Get returns not found
   - Test Incr — increments correctly, errors on non-integer value
   - Test Keys pattern matching
   - Test concurrent Set+Get (will catch races with `-race`)

### Definition of Done
```bash
go test ./internal/store/... -race -count=1
# PASS — zero data race warnings
```

---

## Phase 2 — TCP Server and Binary Protocol

**Goal:** A client can connect over TCP, send a binary-framed command, and get a response. The server wires received commands to the Phase 1 store.

### Tasks (do in this order)

1. Create `internal/protocol/commands.go`
   - Define command ID constants: `CmdSet = 0x01`, `CmdGet = 0x02`, etc.
   - Define `Command` struct: `ID byte`, `Key string`, `Value []byte`, `TTL int64`
   - Define `Response` struct: `Status byte`, `Payload []byte`
   - Define status constants: `StatusOK = 0x00`, `StatusError = 0x01`, `StatusValue = 0x02`, `StatusNull = 0x03`

2. Create `internal/protocol/parser.go`
   - `ReadCommand(r io.Reader) (*Command, error)`
   - Read 1 byte → command ID
   - Read 4 bytes big-endian → key length
   - Read N bytes → key
   - Read 4 bytes big-endian → value length
   - Read N bytes → value
   - Read 8 bytes → TTL (only for SET/EXPIRE commands)

3. Create `internal/protocol/serializer.go`
   - `WriteResponse(w io.Writer, resp *Response) error`
   - Write 1 byte status
   - Write 4 bytes big-endian payload length
   - Write N bytes payload

4. Create `internal/protocol/protocol_test.go`
   - Encode a SET command to bytes, decode it back — fields match exactly
   - Encode each response type, decode back — status and payload match

5. Create `internal/server/server.go`
   - `type Server struct`: holds `*store.Store`, `net.Listener`, `context.Context`
   - `New(addr string, store *store.Store) *Server`
   - `Start() error`: calls `net.Listen`, starts accept loop
   - `Stop()`: cancels context, closes listener

6. Create `internal/server/handler.go`
   - `handleConn(conn net.Conn, store *store.Store)`
   - Loop: `ReadCommand` → `executeCommand` → `WriteResponse`
   - `executeCommand`: switch on `cmd.ID` → call appropriate store method → build Response
   - Handle EOF gracefully (client disconnected)

7. Create `cmd/server/main.go`
   - Create store, create server, call Start, block on signal (SIGINT/SIGTERM)

8. Create `cmd/kvcli/main.go`
   - Connect to TCP server
   - If args provided: send single command, print response, exit
   - If no args: start REPL loop with `bufio.Scanner` reading stdin

### Definition of Done
```bash
# Terminal 1
go run cmd/server/main.go

# Terminal 2
go run cmd/kvcli/main.go SET name Alice
# output: OK
go run cmd/kvcli/main.go GET name
# output: Alice
```

---

## Phase 3 — Persistence

**Goal:** Kill the server, restart it, all keys are still present.

### Tasks (do in this order)

1. Create `internal/persistence/aof.go`
   - Define AOF entry binary format: `[8 bytes timestamp][1 byte cmd][4 byte key_len][key][4 byte val_len][value][8 byte ttl]`
   - `type AOFWriter struct`: holds `*os.File`, `chan AOFEntry`, sync ticker
   - `Start()`: goroutine that reads from channel, writes entries, syncs on ticker
   - `Append(entry AOFEntry)`: non-blocking send to channel
   - `Replay(path string, store *store.Store) error`: open file, read all entries, re-execute each command

2. Create `internal/persistence/snapshot.go`
   - `type Snapshot struct`: path string
   - `Save(store *store.Store) error`: acquire read lock → gob-encode `map[string]*Entry` → write to temp file → `os.Rename` to real path
   - `Load(path string, store *store.Store) error`: open file → gob-decode → populate store

3. Wire persistence into `cmd/server/main.go`
   - On startup: `snapshot.Load()` then `aof.Replay()`
   - Pass `AOFWriter` into server handlers so every SET/DEL/EXPIRE appends an entry

4. Create `internal/persistence/persistence_test.go`
   - Write 100 keys → save snapshot → create new store → load snapshot → verify all keys present
   - Write 50 keys → write AOF entries → replay AOF into empty store → verify all keys present

### Definition of Done
```bash
# Set some keys
go run cmd/kvcli/main.go SET city Mumbai
go run cmd/kvcli/main.go SET count 42

# Kill server (Ctrl+C), restart it
go run cmd/server/main.go

# Keys are still there
go run cmd/kvcli/main.go GET city    # → Mumbai
go run cmd/kvcli/main.go GET count   # → 42
```

---

## Phase 4 — CLI Client Polish

**Goal:** A proper REPL experience with `cobra` subcommands and good error messages.

### Tasks

1. Add `cobra` to go.mod
   ```bash
   go get github.com/spf13/cobra
   ```

2. Refactor `cmd/kvcli/main.go` with cobra root command
   - Subcommands: `set`, `get`, `del`, `expire`, `ttl`, `keys`, `incr`
   - Each subcommand: validate args → connect → send command → print response → exit
   - `--host` and `--port` flags with defaults

3. Implement REPL in a `repl.go` file
   - `readline`-style prompt: `kvstore> `
   - Parse input line into command + args
   - Connect once, reuse connection across commands
   - Handle `EXIT`, `QUIT`, `HELP` as special commands
   - Print command history on up-arrow (use `golang.org/x/term` or `github.com/chzyer/readline`)

### Definition of Done
```bash
./kvcli
kvstore> SET name Alice
OK
kvstore> GET name
"Alice"
kvstore> EXPIRE name 5
OK
kvstore> TTL name
4
kvstore> KEYS *
1) name
kvstore> EXIT
bye
```

---

## Phase 5 — HTTP API

**Goal:** Every store operation is accessible over REST. WebSocket endpoint streams live events to connected clients.

### Tasks (do in this order)

1. Add dependencies
   ```bash
   go get github.com/go-chi/chi/v5
   go get github.com/gorilla/websocket
   ```

2. Create `internal/store/events.go`
   - `type EventType string` — constants: `EventSet`, `EventDel`, `EventExpire`
   - `type Event struct`: `Type EventType`, `Key string`, `Value string`, `TTL int64`, `Timestamp time.Time`
   - Add `subscribers []chan Event` + `subMu sync.Mutex` to Store
   - `Subscribe() chan Event` and `Unsubscribe(ch chan Event)`
   - Emit to all subscribers inside Set, Delete, Expire

3. Create `internal/api/handlers.go`
   - `GET /api/keys` → list all keys (optionally filter with `?pattern=` query param)
   - `GET /api/keys/{key}` → get value + TTL; 404 if not found
   - `POST /api/keys/{key}` → set value from JSON body `{"value":"...","ttl":0}`
   - `DELETE /api/keys/{key}` → delete key
   - `PUT /api/keys/{key}/expire` → set TTL from JSON body `{"seconds":60}`
   - `GET /api/keys/{key}/ttl` → return remaining TTL
   - `GET /api/stats` → total keys, uptime, aof size, snapshot age
   - `GET /api/health` → `{"status":"ok"}`

4. Create `internal/api/ws.go`
   - `GET /ws/events` — upgrade to WebSocket
   - Subscribe to store events
   - Write JSON event to WebSocket on each event
   - Unsubscribe on disconnect

5. Create `internal/api/server.go`
   - Build `chi.Router`, mount all routes, add middleware: CORS, logging, recovery
   - `ListenAndServe` on `:8080`

6. Create `internal/api/api_test.go`
   - Use `net/http/httptest` to test every handler
   - Test 404 on missing key
   - Test correct JSON structure on GET

### Definition of Done
```bash
# Set via CLI, read via HTTP
./kvcli SET city Mumbai
curl http://localhost:8080/api/keys/city
# → {"key":"city","value":"Mumbai","ttl":-1}

# Set via HTTP, read via CLI
curl -X POST http://localhost:8080/api/keys/lang \
  -H "Content-Type: application/json" \
  -d '{"value":"Go"}'
./kvcli GET lang
# → Go
```

---

## Phase 6 — Next.js Dashboard

**Goal:** A browser UI that shows all keys, stats, allows editing, and streams live events.

Documented in detail in [`FRONTEND.md`](FRONTEND.md).

### High-Level Tasks

1. Create Next.js app: `npx create-next-app@latest web --typescript --tailwind --app`
2. Install shadcn/ui: `npx shadcn@latest init`
3. Install Tanstack Query: `npm install @tanstack/react-query`
4. Build in this order:
   - `lib/types.ts` + `lib/api.ts` — typed HTTP client
   - `hooks/useKeys.ts` + `hooks/useStats.ts` — data fetching
   - `hooks/useEventStream.ts` — WebSocket hook
   - `components/dashboard/StatsPanel.tsx` — stat cards
   - `components/dashboard/KeysTable.tsx` — key list
   - `components/dashboard/EventStream.tsx` — live log
   - `app/page.tsx` — assemble the full dashboard

### Definition of Done
- Dashboard loads and shows all keys
- Adding a key via the form appears in the table instantly (after query invalidation)
- The event stream shows every SET/DEL as it happens
- Connection badge turns red when the Go server is unreachable

---

## Phase 7 — Raft Clustering

**Goal:** Three nodes agree on state. Kill the leader — a new one is elected within 3 seconds. Writes to any node succeed.

Documented in detail in [`RAFT.md`](RAFT.md).

**Important:** Do not start this phase until Phase 1–3 have zero known bugs. Raft bugs that appear to be consensus bugs are often actually storage bugs underneath.

### High-Level Tasks (in order)

1. Week 10: Node struct, state machine, term numbers, vote tracking
2. Week 11: `RequestVote` RPC, election timeout, leader detection
3. Week 12: `AppendEntries` RPC, log replication, heartbeats
4. Week 13: Commit index, applying log entries to store, leader proxy for writes

### Definition of Done
```bash
# Start 3 nodes
./kvstore-server --config node1.yaml &
./kvstore-server --config node2.yaml &
./kvstore-server --config node3.yaml &

# Write to leader
./kvcli --port 6379 SET distributed true

# Kill the leader
kill <leader-pid>

# Within 3 seconds, a new leader is elected
# Read from a remaining node
./kvcli --port 6380 GET distributed
# → true
```

---

## Phase 8 — Observability and Benchmarks

**Goal:** Prometheus metrics, pprof profiling endpoint, benchmark that confirms >100k ops/sec.

### Tasks

1. Add Prometheus client
   ```bash
   go get github.com/prometheus/client_golang/prometheus
   ```

2. Instrument key metrics:
   - `kvstore_commands_total` — counter by command type
   - `kvstore_keys_total` — gauge (current key count)
   - `kvstore_command_duration_seconds` — histogram of command latency

3. Expose `/metrics` endpoint on HTTP server

4. Enable pprof:
   ```go
   import _ "net/http/pprof"
   ```
   This auto-registers `/debug/pprof/` routes.

5. Write `bench/bench.go`:
   - Connect N goroutines to the TCP server
   - Each goroutine runs a loop: SET random key → GET same key
   - Count ops/sec across all goroutines
   - Run for 10 seconds, print ops/sec

### Definition of Done
```bash
go run bench/bench.go --connections 50 --duration 10s
# Throughput: 143,200 ops/sec
```
