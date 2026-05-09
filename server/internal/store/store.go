package store

import (
	"container/heap"
	"fmt"
	"hash/fnv"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// Entry
// ─────────────────────────────────────────────────────────────────────────────

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

// ─────────────────────────────────────────────────────────────────────────────
// Sharded Store
// ─────────────────────────────────────────────────────────────────────────────

const numShards = 16

// shard holds an independent slice of the key-space.
// Each shard has its own RWMutex, data map, TTL heap, TTL index, and
// eviction notification channel — entirely independent of every other shard.
type shard struct {
	mu       sync.RWMutex
	data     map[string]*Entry
	ttlHeap  *TTLHeap
	ttlIndex map[string]*TTLItem
	notify   chan struct{} // wakes the eviction goroutine when a TTL key is added
}

func newShard() shard {
	h := &TTLHeap{}
	heap.Init(h)
	return shard{
		data:     make(map[string]*Entry),
		ttlHeap:  h,
		ttlIndex: make(map[string]*TTLItem),
		notify:   make(chan struct{}, 1),
	}
}

// Store is the public handle to the sharded key-value store.
// All 16 shards are embedded by value so there is zero pointer chasing
// between the Store header and the shard data.
type Store struct {
	shards      [numShards]shard
	once        sync.Once
	subscribers []chan Event
	subMu       sync.RWMutex
}

// New creates a ready-to-use sharded Store.
func New() *Store {
	s := &Store{}
	for i := range s.shards {
		s.shards[i] = newShard()
	}
	return s
}

// shardFor returns the shard responsible for the given key.
// Uses FNV-1a 32-bit hash for speed and good distribution.
func (s *Store) shardFor(key string) *shard {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key)) // Write on fnv hash never errors
	return &s.shards[h.Sum32()%numShards]
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func expiresAt(ttlNs int64) int64 {
	if ttlNs == 0 {
		return 0
	}
	return time.Now().UnixNano() + ttlNs
}

// removeTTL removes the TTL tracking for a key from a shard.
// Caller must hold sh.mu (write lock).
func removeTTL(sh *shard, key string) {
	if item, ok := sh.ttlIndex[key]; ok {
		heap.Remove(sh.ttlHeap, item.index)
		delete(sh.ttlIndex, key)
	}
}

// pushTTL adds TTL tracking for a key on a shard.
// Caller must hold sh.mu (write lock).
func pushTTL(sh *shard, key string, absExpiry int64) {
	item := &TTLItem{key: key, expiresAt: absExpiry}
	heap.Push(sh.ttlHeap, item)
	sh.ttlIndex[key] = item
	// Wake the shard's eviction goroutine (non-blocking).
	select {
	case sh.notify <- struct{}{}:
	default:
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Core Operations
// ─────────────────────────────────────────────────────────────────────────────

func (s *Store) Ping() string { return "PONG" }

func (s *Store) Set(key string, value []byte, ttlNs int64) {
	sh := s.shardFor(key)
	sh.mu.Lock()

	sh.data[key] = &Entry{
		Value:     value,
		ExpiresAt: expiresAt(ttlNs),
	}

	// Always remove the old TTL entry first, regardless of whether
	// the new call has a TTL. Without this, a key updated from TTL→no-TTL
	// leaves a stale item in the heap that later evicts the key incorrectly.
	removeTTL(sh, key)

	if ttlNs > 0 {
		pushTTL(sh, key, sh.data[key].ExpiresAt)
	}

	sh.mu.Unlock()
	s.publish(Event{Type: EventSet, Key: key, Value: string(value), Timestamp: time.Now().UTC()})
}

func (s *Store) Get(key string) ([]byte, bool) {
	sh := s.shardFor(key)
	sh.mu.RLock()
	entry, ok := sh.data[key]
	sh.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if entry.IsExpired() {
		sh.mu.Lock()
		// Double-check: another goroutine may have deleted it already.
		if e, exists := sh.data[key]; exists && e.IsExpired() {
			delete(sh.data, key)
			removeTTL(sh, key)
		}
		sh.mu.Unlock()
		return nil, false
	}

	return entry.Value, true
}

func (s *Store) Delete(key string) bool {
	sh := s.shardFor(key)
	sh.mu.Lock()

	_, exists := sh.data[key]
	if !exists {
		sh.mu.Unlock()
		return false
	}

	delete(sh.data, key)
	removeTTL(sh, key)

	sh.mu.Unlock()
	s.publish(Event{Type: EventDel, Key: key, Timestamp: time.Now().UTC()})
	return true
}

// TTL returns the remaining lifetime of a key in nanoseconds.
// Returns -1 if the key has no expiry, -2 if the key does not exist.
func (s *Store) TTL(key string) int64 {
	sh := s.shardFor(key)
	sh.mu.RLock()
	defer sh.mu.RUnlock()

	entry, exists := sh.data[key]
	if !exists {
		return -2
	}
	if entry.ExpiresAt == 0 {
		return -1
	}
	remaining := entry.ExpiresAt - time.Now().UnixNano()
	if remaining <= 0 {
		return -2 // expired but not yet evicted — treat as missing
	}
	return remaining
}

func (s *Store) Expire(key string, ttlNs int64) bool {
	sh := s.shardFor(key)
	sh.mu.Lock()

	entry, exists := sh.data[key]
	if !exists {
		sh.mu.Unlock()
		return false
	}

	entry.ExpiresAt = expiresAt(ttlNs)

	removeTTL(sh, key)
	if ttlNs > 0 {
		pushTTL(sh, key, entry.ExpiresAt)
	}

	sh.mu.Unlock()
	s.publish(Event{Type: EventExpire, Key: key, TTL: ttlNs, Timestamp: time.Now().UTC()})
	return true
}

func (s *Store) Incr(key string) (int64, error) {
	sh := s.shardFor(key)
	sh.mu.Lock()

	entry, exists := sh.data[key]

	// treat an expired entry exactly like a missing key
	if !exists || entry.IsExpired() {
		if exists {
			// clean up the stale entry
			removeTTL(sh, key)
		}
		sh.data[key] = &Entry{Value: []byte("1"), ExpiresAt: 0}

		sh.mu.Unlock()
		s.publish(Event{Type: EventSet, Key: key, Value: "1", Timestamp: time.Now().UTC()})
		return 1, nil
	}

	val, err := strconv.ParseInt(string(entry.Value), 10, 64)
	if err != nil {
		sh.mu.Unlock()
		return 0, fmt.Errorf("value is not an integer")
	}

	val++
	entry.Value = []byte(strconv.FormatInt(val, 10))
	valStr := string(entry.Value)

	sh.mu.Unlock()
	s.publish(Event{Type: EventSet, Key: key, Value: valStr, Timestamp: time.Now().UTC()})
	return val, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Batch Operations
// ─────────────────────────────────────────────────────────────────────────────

// MGet fetches multiple keys. Each key is looked up independently in its own
// shard — no global lock is held, so reads are concurrent-safe across shards.
func (s *Store) MGet(keys []string) [][]byte {
	result := make([][]byte, len(keys))
	for i, key := range keys {
		val, ok := s.Get(key)
		if ok {
			result[i] = val
		}
	}
	return result
}

// MSet writes multiple keys. Each key is written independently to its shard.
func (s *Store) MSet(entries map[string][]byte) {
	events := make([]Event, 0, len(entries))
	for key, value := range entries {
		sh := s.shardFor(key)
		sh.mu.Lock()
		removeTTL(sh, key)
		sh.data[key] = &Entry{Value: value, ExpiresAt: 0}
		sh.mu.Unlock()
		events = append(events, Event{Type: EventSet, Key: key, Value: string(value), Timestamp: time.Now().UTC()})
	}
	for _, e := range events {
		s.publish(e)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Scan / Aggregate operations  (hold each shard lock independently)
// ─────────────────────────────────────────────────────────────────────────────

// Keys returns all non-expired keys matching the glob pattern.
func (s *Store) Keys(pattern string) []string {
	var keys []string
	for i := range s.shards {
		sh := &s.shards[i]
		sh.mu.RLock()
		for key, entry := range sh.data {
			if !entry.IsExpired() {
				matched, err := filepath.Match(pattern, key)
				if err == nil && matched {
					keys = append(keys, key)
				}
			}
		}
		sh.mu.RUnlock()
	}
	return keys
}

func (s *Store) Count() int {
	total := 0
	for i := range s.shards {
		sh := &s.shards[i]
		sh.mu.RLock()
		for _, entry := range sh.data {
			if !entry.IsExpired() {
				total++
			}
		}
		sh.mu.RUnlock()
	}
	return total
}

// MemoryUsage returns an estimate of the total bytes consumed by all entries.
func (s *Store) MemoryUsage() int64 {
	var total int64
	for i := range s.shards {
		sh := &s.shards[i]
		sh.mu.RLock()
		for k, entry := range sh.data {
			if !entry.IsExpired() {
				total += int64(len(k)) + int64(len(entry.Value)) + 8
			}
		}
		sh.mu.RUnlock()
	}
	return total
}

// TTLKeyCount returns the number of keys with an active TTL across all shards.
func (s *Store) TTLKeyCount() int {
	total := 0
	for i := range s.shards {
		sh := &s.shards[i]
		sh.mu.RLock()
		total += sh.ttlHeap.Len()
		sh.mu.RUnlock()
	}
	return total
}

// SubscriberCount returns the number of active event subscribers.
func (s *Store) SubscriberCount() int {
	s.subMu.RLock()
	defer s.subMu.RUnlock()
	return len(s.subscribers)
}

// Snapshot returns a copy of all non-expired entries across every shard.
func (s *Store) Snapshot() map[string]*Entry {
	snap := make(map[string]*Entry)
	for i := range s.shards {
		sh := &s.shards[i]
		sh.mu.RLock()
		for k, v := range sh.data {
			if !v.IsExpired() {
				snap[k] = v
			}
		}
		sh.mu.RUnlock()
	}
	return snap
}

// SetRaw inserts an entry directly (used by snapshot/AOF restore).
// It does NOT emit events or touch the AOF.
func (s *Store) SetRaw(key string, entry *Entry) {
	sh := s.shardFor(key)
	sh.mu.Lock()
	defer sh.mu.Unlock()

	removeTTL(sh, key)
	sh.data[key] = entry

	if entry.ExpiresAt > 0 {
		pushTTL(sh, key, entry.ExpiresAt)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Pub/Sub (top-level, unchanged)
// ─────────────────────────────────────────────────────────────────────────────

func (s *Store) Subscribe() chan Event {
	ch := make(chan Event, 64)
	s.subMu.Lock()
	s.subscribers = append(s.subscribers, ch)
	s.subMu.Unlock()
	return ch
}

func (s *Store) Unsubscribe(ch chan Event) {
	s.subMu.Lock()
	for i := range s.subscribers {
		if s.subscribers[i] == ch {
			s.subscribers = slices.Delete(s.subscribers, i, i+1)
			break
		}
	}
	s.subMu.Unlock()
	close(ch)
}

func (s *Store) publish(event Event) {
	s.subMu.RLock()
	for i := range s.subscribers {
		select {
		case s.subscribers[i] <- event:
		default:
			// LOAD SHEDDING: drop the event for a slow subscriber rather than
			// blocking the write path.
		}
	}
	s.subMu.RUnlock()
}
