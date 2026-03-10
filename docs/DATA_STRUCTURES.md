# Data Structures

> Every struct, algorithm, and concurrency pattern used in the project — with Go type definitions and explanations.

---

## Core Store

### Entry

```go
// Entry is the value stored for every key.
// Value is raw bytes — the store does not care about encoding.
// ExpiresAt is Unix nanoseconds. Zero means no expiry.
type Entry struct {
    Value     []byte
    ExpiresAt int64
}

func (e *Entry) IsExpired() bool {
    return e.ExpiresAt > 0 && time.Now().UnixNano() > e.ExpiresAt
}
```

Why `[]byte` and not `string`? Bytes are the universal container. The store does not interpret values — a string, a JSON blob, an integer as text — they are all just bytes. The protocol layer converts to/from bytes.

---

### Store

```go
type Store struct {
    mu       sync.RWMutex
    data     map[string]*Entry
    ttlHeap  *TTLHeap
    events   chan Event
    once     sync.Once   // ensures background goroutines start exactly once
}

func New() *Store {
    s := &Store{
        data:    make(map[string]*Entry),
        ttlHeap: &TTLHeap{},
        events:  make(chan Event, 256), // buffered so writers never block
    }
    heap.Init(s.ttlHeap)
    return s
}
```

**Why `sync.RWMutex` and not `sync.Map`:**

`sync.Map` is designed for the "write once, read many" pattern — like a cache that is populated at startup. A KV store has unpredictable writes. `RWMutex` is the right choice here. It allows any number of concurrent readers (reads scale linearly with CPUs) and forces writers to wait for all readers to finish, then get exclusive access.

**Why a buffered event channel:**

The `events` channel has a buffer of 256. This means that if the WebSocket fan-out goroutine is slow (e.g., a browser client has a slow connection), the store's `Set` method will not block. If all 256 slots fill up, the next event is dropped — that is acceptable for a UI event stream.

---

### Set

```go
func (s *Store) Set(key string, value []byte, ttlNs int64) {
    s.mu.Lock()
    defer s.mu.Unlock()

    s.data[key] = &Entry{
        Value:     value,
        ExpiresAt: expiresAt(ttlNs), // 0 if ttlNs == 0
    }

    if ttlNs > 0 {
        heap.Push(s.ttlHeap, &TTLItem{key: key, expiresAt: s.data[key].ExpiresAt})
    }

    // Non-blocking event emit — drop if buffer full
    select {
    case s.events <- Event{Type: EventSet, Key: key, Value: string(value)}:
    default:
    }
}

func expiresAt(ttlNs int64) int64 {
    if ttlNs == 0 {
        return 0
    }
    return time.Now().UnixNano() + ttlNs
}
```

---

### Get

```go
func (s *Store) Get(key string) ([]byte, bool) {
    s.mu.RLock()
    entry, ok := s.data[key]
    s.mu.RUnlock()

    if !ok {
        return nil, false
    }

    // Lazy expiry check — do not hold the lock while checking
    if entry.IsExpired() {
        s.mu.Lock()
        // Double-check: another goroutine may have deleted it already
        if e, exists := s.data[key]; exists && e.IsExpired() {
            delete(s.data, key)
        }
        s.mu.Unlock()
        return nil, false
    }

    return entry.Value, true
}
```

Note the double-checked locking pattern: first acquire a read lock to check, then acquire a write lock to delete. Always re-check the condition after acquiring the write lock because another goroutine may have already done the delete.

---

## TTL System

### TTLItem

```go
type TTLItem struct {
    key       string
    expiresAt int64  // Unix nanoseconds
    index     int    // position in the heap — needed for O(log n) removal
}
```

The `index` field is maintained by the heap so you can remove an arbitrary item from the middle of the heap in O(log n) time. Without it, removal requires scanning the entire heap — O(n).

---

### TTLHeap

```go
// TTLHeap implements heap.Interface — a min-heap sorted by ExpiresAt.
// The item with the smallest ExpiresAt (soonest to expire) is at index 0.
type TTLHeap []*TTLItem

func (h TTLHeap) Len() int           { return len(h) }
func (h TTLHeap) Less(i, j int) bool { return h[i].expiresAt < h[j].expiresAt }
func (h TTLHeap) Swap(i, j int) {
    h[i], h[j] = h[j], h[i]
    h[i].index = i
    h[j].index = j
}

func (h *TTLHeap) Push(x any) {
    n := len(*h)
    item := x.(*TTLItem)
    item.index = n
    *h = append(*h, item)
}

func (h *TTLHeap) Pop() any {
    old := *h
    n := len(old)
    item := old[n-1]
    old[n-1] = nil   // avoid memory leak
    item.index = -1  // mark as removed
    *h = old[:n-1]
    return item
}
```

**Why a min-heap and not a sorted slice or map:**

| Structure | Push | Pop min | Check min |
|-----------|------|---------|-----------|
| Min-heap | O(log n) | O(log n) | O(1) |
| Sorted slice | O(n) | O(1) | O(1) |
| Map | O(1) | O(n) | O(n) |

The background eviction goroutine only needs `Check min` (peek at the next key to expire) and `Pop min` (remove it when it has expired). Both are O(log n) with a heap.

---

### Background Eviction Goroutine

```go
func (s *Store) startEviction(ctx context.Context) {
    for {
        s.mu.RLock()
        if s.ttlHeap.Len() == 0 {
            s.mu.RUnlock()
            select {
            case <-ctx.Done():
                return
            case <-time.After(100 * time.Millisecond):
                continue
            }
        }
        nextExpiry := (*s.ttlHeap)[0].expiresAt
        s.mu.RUnlock()

        now := time.Now().UnixNano()
        if nextExpiry > now {
            // Sleep until the next key is about to expire
            sleepDuration := time.Duration(nextExpiry - now)
            select {
            case <-ctx.Done():
                return
            case <-time.After(sleepDuration):
            }
        }

        // Acquire write lock, pop all expired keys
        s.mu.Lock()
        for s.ttlHeap.Len() > 0 && (*s.ttlHeap)[0].expiresAt <= time.Now().UnixNano() {
            item := heap.Pop(s.ttlHeap).(*TTLItem)
            delete(s.data, item.key)
            select {
            case s.events <- Event{Type: EventExpired, Key: item.key}:
            default:
            }
        }
        s.mu.Unlock()
    }
}
```

The goroutine sleeps precisely until the next key should expire — it does not wake up every 100ms when there is nothing to do. This is efficient: if the nearest expiry is in 1 hour, the goroutine sleeps for 1 hour.

---

## Protocol Structures

### Command

```go
type Command struct {
    ID    byte
    Key   string
    Value []byte
    TTL   int64  // nanoseconds; 0 means no TTL
}
```

### Response

```go
type Response struct {
    Status  byte
    Payload []byte
}
```

---

## Raft Structures

Documented in detail in [`RAFT.md`](RAFT.md).

```go
type LogEntry struct {
    Index   uint64
    Term    uint64
    Command []byte  // serialized KV command (same binary format as TCP protocol)
}

type RaftNode struct {
    id          string
    state       NodeState  // Follower | Candidate | Leader
    currentTerm uint64
    votedFor    string
    log         []LogEntry
    commitIndex uint64
    lastApplied uint64

    // Volatile leader state (reset on election)
    nextIndex  map[string]uint64
    matchIndex map[string]uint64

    peers  []string
    store  *store.Store
    mu     sync.RWMutex
}

type NodeState int

const (
    Follower  NodeState = iota
    Candidate
    Leader
)
```

---

## Concurrency Patterns Used

### Pattern 1 — RWMutex for the Store

```go
// Multiple readers: RLock allows concurrent reads
func (s *Store) Get(key string) ([]byte, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    // ... read only
}

// Single writer: Lock blocks until all readers release
func (s *Store) Set(key string, value []byte, ttl int64) {
    s.mu.Lock()
    defer s.mu.Unlock()
    // ... write
}
```

### Pattern 2 — Goroutine per TCP Connection

```go
func (s *Server) acceptLoop() {
    for {
        conn, err := s.listener.Accept()
        if err != nil {
            return // listener closed, shutting down
        }
        go s.handleConn(conn) // each connection owns its goroutine
    }
}
```

Each `handleConn` goroutine blocks on `io.ReadFull`, waiting for the next frame. Go's scheduler parks it cheaply until data arrives.

### Pattern 3 — Background Goroutine with Context Cancellation

```go
func startBackground(ctx context.Context, work func()) {
    go func() {
        for {
            select {
            case <-ctx.Done():
                return // clean shutdown
            default:
                work()
            }
        }
    }()
}
```

Pass a `context.Context` derived from `context.WithCancel`. Call cancel on shutdown — all goroutines listening on `ctx.Done()` will exit cleanly.

### Pattern 4 — Fan-out with Go Channels

```go
type EventBus struct {
    mu          sync.Mutex
    subscribers []chan Event
}

func (b *EventBus) Subscribe() chan Event {
    ch := make(chan Event, 64)
    b.mu.Lock()
    b.subscribers = append(b.subscribers, ch)
    b.mu.Unlock()
    return ch
}

func (b *EventBus) Publish(event Event) {
    b.mu.Lock()
    defer b.mu.Unlock()
    for _, ch := range b.subscribers {
        select {
        case ch <- event:
        default: // slow subscriber — drop the event rather than block
        }
    }
}
```

Each WebSocket connection subscribes and gets its own channel. The store publishes to the bus. No subscriber can slow down the store.
