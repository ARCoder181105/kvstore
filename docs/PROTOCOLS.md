# Protocols

> Complete specification for every communication protocol in the system — binary TCP, HTTP REST, and WebSocket.

---

## 1. Binary TCP Protocol

Used between the TCP server (`:6379`) and all TCP clients (CLI, benchmarks).

### Why Binary and Not Text

A text protocol like `SET key value\r\n` requires scanning for delimiters — you read byte by byte until you find `\n`. A binary protocol with length-prefixed fields lets you read exactly the bytes you need with `io.ReadFull` — no scanning, no edge cases around values that contain `\n`.

---

### Request Frame Format

```
Byte  0       1–4              5–(4+KL)     (5+KL)–(8+KL)    (9+KL)–(8+KL+VL)   (9+KL+VL)–(16+KL+VL)
      ┌───────┬────────────────┬────────────┬────────────────┬─────────────────────┬───────────────────┐
      │ CmdID │ Key Length     │ Key Bytes  │ Value Length   │ Value Bytes         │ TTL (nanoseconds) │
      │ 1 byte│ uint32 big-end │ KL bytes   │ uint32 big-end │ VL bytes            │ int64 big-endian  │
      └───────┴────────────────┴────────────┴────────────────┴─────────────────────┴───────────────────┘

For GET and DEL: Value Length = 0, no Value bytes, TTL = 0
For SET with no TTL: TTL = 0
For EXPIRE: Key = key name, Value Length = 0, TTL = expiry in nanoseconds from now
```

### Command IDs

```go
const (
    CmdSet    byte = 0x01
    CmdGet    byte = 0x02
    CmdDel    byte = 0x03
    CmdExpire byte = 0x04
    CmdTTL    byte = 0x05
    CmdKeys   byte = 0x06
    CmdIncr   byte = 0x07
    CmdMSet   byte = 0x08
    CmdMGet   byte = 0x09
    CmdPing   byte = 0x0A
)
```

### Response Frame Format

```
Byte  0        1–4              5–(4+PL)
      ┌────────┬────────────────┬────────────────┐
      │ Status │ Payload Length │ Payload Bytes  │
      │ 1 byte │ uint32 big-end │ PL bytes (UTF8)│
      └────────┴────────────────┴────────────────┘
```

### Status Codes

```go
const (
    StatusOK    byte = 0x00  // Command succeeded, no payload needed (SET, DEL, EXPIRE)
    StatusError byte = 0x01  // Error — payload contains error message string
    StatusValue byte = 0x02  // Success — payload contains the value bytes (GET)
    StatusNull  byte = 0x03  // Key not found (GET on missing key)
    StatusInt   byte = 0x04  // Success — payload contains int64 as string (TTL, INCR)
    StatusArray byte = 0x05  // Success — payload contains newline-separated strings (KEYS)
)
```

### Wire Examples

**SET name Alice (no TTL):**
```
01                     ← CmdSet
00 00 00 04            ← key length: 4
6E 61 6D 65            ← "name"
00 00 00 05            ← value length: 5
41 6C 69 63 65         ← "Alice"
00 00 00 00 00 00 00 00 ← TTL: 0 (no expiry)
```

**Response OK:**
```
00                     ← StatusOK
00 00 00 00            ← payload length: 0
```

**GET name:**
```
02                     ← CmdGet
00 00 00 04            ← key length: 4
6E 61 6D 65            ← "name"
00 00 00 00            ← value length: 0 (ignored for GET)
00 00 00 00 00 00 00 00 ← TTL: 0 (ignored for GET)
```

**Response VALUE:**
```
02                     ← StatusValue
00 00 00 05            ← payload length: 5
41 6C 69 63 65         ← "Alice"
```

---

## 2. AOF Entry Format

Written to `aof.log` for every SET, DEL, and EXPIRE command.

```
Bytes  0–7       8        9–12             13–(12+KL)    (13+KL)–(16+KL)   (17+KL)–(16+KL+VL)   (17+KL+VL)–(24+KL+VL)
       ┌─────────┬────────┬────────────────┬─────────────┬─────────────────┬──────────────────────┬─────────────────────┐
       │ Unix ns │ CmdID  │ Key Length     │ Key Bytes   │ Value Length    │ Value Bytes          │ TTL nanoseconds     │
       │ int64   │ 1 byte │ uint32 big-end │ KL bytes    │ uint32 big-end  │ VL bytes             │ int64 big-end       │
       └─────────┴────────┴────────────────┴─────────────┴─────────────────┴──────────────────────┴─────────────────────┘
```

The 8-byte timestamp at the front is used during recovery to replay only AOF entries newer than the last snapshot.

---

## 3. HTTP REST API

Base URL: `http://localhost:8080`

All request and response bodies are JSON. All errors return the same error envelope.

### Error Envelope

```json
{
  "error": "key not found",
  "code": "NOT_FOUND"
}
```

### Endpoints

---

#### `GET /api/health`

**Description:** Health check — returns OK if the server is running.

**Response 200:**
```json
{
  "status": "ok",
  "uptime": "2h 34m 12s"
}
```

---

#### `GET /api/stats`

**Description:** Server statistics.

**Response 200:**
```json
{
  "total_keys": 42,
  "memory_bytes": 8192,
  "uptime": "2h 34m 12s",
  "aof_size_bytes": 204800,
  "snapshot_age_seconds": 3600,
  "commands_processed": 150000
}
```

---

#### `GET /api/keys`

**Description:** List all keys. Optionally filter with a glob pattern.

**Query params:** `?pattern=user:*` (optional, defaults to `*`)

**Response 200:**
```json
{
  "keys": [
    { "key": "name", "value": "Alice", "ttl": -1 },
    { "key": "session:abc", "value": "{...}", "ttl": 47 },
    { "key": "counter", "value": "42", "ttl": -1 }
  ],
  "count": 3
}
```

`ttl` is seconds remaining, or `-1` for no expiry.

---

#### `GET /api/keys/:key`

**Description:** Get a single key's value and TTL.

**Response 200:**
```json
{
  "key": "name",
  "value": "Alice",
  "ttl": -1
}
```

**Response 404:**
```json
{
  "error": "key not found",
  "code": "NOT_FOUND"
}
```

---

#### `POST /api/keys/:key`

**Description:** Set a key's value. Optionally set a TTL in seconds.

**Request body:**
```json
{
  "value": "Alice",
  "ttl": 60
}
```

`ttl` is optional. Omit or set to `0` for no expiry.

**Response 200:**
```json
{
  "key": "name",
  "value": "Alice",
  "ttl": 60
}
```

---

#### `DELETE /api/keys/:key`

**Description:** Delete a key.

**Response 200:**
```json
{
  "key": "name",
  "deleted": true
}
```

**Response 404:** Key not found envelope.

---

#### `PUT /api/keys/:key/expire`

**Description:** Set or update TTL on an existing key.

**Request body:**
```json
{
  "seconds": 60
}
```

**Response 200:**
```json
{
  "key": "name",
  "ttl": 60
}
```

---

#### `GET /api/keys/:key/ttl`

**Description:** Get remaining TTL of a key.

**Response 200:**
```json
{
  "key": "name",
  "ttl": 47
}
```

`ttl` of `-1` means no expiry. `ttl` of `-2` means key does not exist.

---

## 4. WebSocket Event Stream

**Endpoint:** `ws://localhost:8080/ws/events`

**Protocol:** Upgrade HTTP GET to WebSocket. The server sends JSON text frames. The client only receives — no client-to-server messages are processed.

### Event Envelope

```typescript
type EventType = "SET" | "DEL" | "EXPIRE" | "EXPIRED"

interface Event {
  type: EventType
  key: string
  value?: string       // present for SET events
  ttl?: number         // present for SET with TTL and EXPIRE events
  timestamp: string    // ISO 8601
}
```

### Example Events

**SET event:**
```json
{
  "type": "SET",
  "key": "name",
  "value": "Alice",
  "ttl": -1,
  "timestamp": "2025-04-15T10:01:23.456Z"
}
```

**DEL event:**
```json
{
  "type": "DEL",
  "key": "name",
  "timestamp": "2025-04-15T10:01:45.789Z"
}
```

**EXPIRED event (TTL eviction):**
```json
{
  "type": "EXPIRED",
  "key": "session:abc",
  "timestamp": "2025-04-15T10:01:52.001Z"
}
```

### Connection Behavior

- Server keeps the connection open indefinitely
- Server sends a ping frame every 30 seconds to detect dead connections
- Client should implement reconnect logic with exponential backoff (start at 1s, max at 30s)
- On reconnect, the client should refetch all keys via HTTP to resync state (do not rely on events for full state)
