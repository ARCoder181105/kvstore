package store

import (
	"container/heap"
	"fmt"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

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

type Store struct {
	mu       sync.RWMutex
	data     map[string]*Entry
	ttlHeap  *TTLHeap
	ttlIndex map[string]*TTLItem
	events   chan Event
	notify   chan struct{} // wakes the eviction goroutine when a TTL key is added
	once     sync.Once
}

func New() *Store {
	s := &Store{
		data:     make(map[string]*Entry),
		ttlHeap:  &TTLHeap{},
		ttlIndex: make(map[string]*TTLItem),
		events:   make(chan Event, 256),
		notify:   make(chan struct{}, 1),
	}
	heap.Init(s.ttlHeap)
	return s
}

func (s *Store) Ping() string {
	return "PONG"
}

func (s *Store) Set(key string, value []byte, ttlNs int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = &Entry{
		Value:     value,
		ExpiresAt: expiresAt(ttlNs),
	}

	// Always remove the old TTL entry first, regardless of whether
	// the new call has a TTL. Without this, a key updated from TTL→no-TTL
	// leaves a stale item in the heap that later evicts the key incorrectly.
	if old, ok := s.ttlIndex[key]; ok {
		heap.Remove(s.ttlHeap, old.index)
		delete(s.ttlIndex, key)
	}

	if ttlNs > 0 {
		item := &TTLItem{key: key, expiresAt: s.data[key].ExpiresAt}
		heap.Push(s.ttlHeap, item)
		s.ttlIndex[key] = item
		// Wake the eviction goroutine (non-blocking; it may already be awake)
		select {
		case s.notify <- struct{}{}:
		default:
		}
	}

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

func (s *Store) Get(key string) ([]byte, bool) {
	s.mu.RLock()
	entry, ok := s.data[key]
	s.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if entry.IsExpired() {
		s.mu.Lock()
		// Double-check: another goroutine may have deleted it already
		if e, exists := s.data[key]; exists && e.IsExpired() {
			delete(s.data, key)
			if item, ok := s.ttlIndex[key]; ok {
				heap.Remove(s.ttlHeap, item.index)
				delete(s.ttlIndex, key)
			}
		}
		s.mu.Unlock()
		return nil, false
	}

	return entry.Value, true
}

func (s *Store) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.data[key]
	if !exists {
		return false
	}

	delete(s.data, key)

	if item, ok := s.ttlIndex[key]; ok {
		heap.Remove(s.ttlHeap, item.index)
		delete(s.ttlIndex, key)
	}

	select {
	case s.events <- Event{Type: EventDel, Key: key}:
	default:
	}

	return true
}

// TTL returns the remaining lifetime of a key in nanoseconds.
// Returns -1 if the key has no expiry, -2 if the key does not exist.
func (s *Store) TTL(key string) int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.data[key]
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
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.data[key]
	if !exists {
		return false
	}

	entry.ExpiresAt = expiresAt(ttlNs)

	// Always remove old TTL item first
	if old, ok := s.ttlIndex[key]; ok {
		heap.Remove(s.ttlHeap, old.index)
		delete(s.ttlIndex, key)
	}

	if ttlNs > 0 {
		item := &TTLItem{key: key, expiresAt: entry.ExpiresAt}
		heap.Push(s.ttlHeap, item)
		s.ttlIndex[key] = item
		select {
		case s.notify <- struct{}{}:
		default:
		}
	}

	select {
	case s.events <- Event{Type: EventExpire, Key: key, TTL: ttlNs}:
	default:
	}

	return true
}

func (s *Store) MGet(keys []string) [][]byte {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([][]byte, len(keys))
	for i, key := range keys {
		entry, exists := s.data[key]
		if exists && !entry.IsExpired() {
			result[i] = entry.Value
		}
	}
	return result
}

func (s *Store) MSet(entries map[string][]byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for key, value := range entries {
		// Remove old TTL entry if present
		if old, ok := s.ttlIndex[key]; ok {
			heap.Remove(s.ttlHeap, old.index)
			delete(s.ttlIndex, key)
		}
		s.data[key] = &Entry{Value: value, ExpiresAt: 0}
		select {
		case s.events <- Event{Type: EventSet, Key: key, Value: string(value)}:
		default:
		}
	}
}

func (s *Store) Incr(key string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.data[key]

	// treat an expired entry exactly like a missing key
	if !exists || entry.IsExpired() {
		if exists {
			// clean up the stale entry
			if item, ok := s.ttlIndex[key]; ok {
				heap.Remove(s.ttlHeap, item.index)
				delete(s.ttlIndex, key)
			}
		}
		s.data[key] = &Entry{Value: []byte("1"), ExpiresAt: 0}
		select {
		case s.events <- Event{Type: EventSet, Key: key, Value: "1"}:
		default:
		}
		return 1, nil
	}

	val, err := strconv.ParseInt(string(entry.Value), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("value is not an integer")
	}

	val++
	entry.Value = []byte(strconv.FormatInt(val, 10))

	select {
	case s.events <- Event{Type: EventSet, Key: key, Value: string(entry.Value)}:
	default:
	}

	return val, nil
}

// Keys returns all non-expired keys matching the glob pattern.
// snapshot keys under a short read lock, then pattern-match outside
// the lock to avoid holding it for O(n) time under write contention.
func (s *Store) Keys(pattern string) []string {
	s.mu.RLock()
	candidates := make([]string, 0, len(s.data))
	for key, entry := range s.data {
		if !entry.IsExpired() {
			candidates = append(candidates, key)
		}
	}
	s.mu.RUnlock()

	keys := make([]string, 0, len(candidates))
	for _, key := range candidates {
		matched, err := filepath.Match(pattern, key)
		if err != nil {
			continue
		}
		if matched {
			keys = append(keys, key)
		}
	}
	return keys
}

// Snapshot returns a copy of the current store map for snapshotting.
func (s *Store) Snapshot() map[string]*Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snap := make(map[string]*Entry, len(s.data))
	for k, v := range s.data {
		// Only include non-expired keys
		if !v.IsExpired() {
			snap[k] = v
		}
	}
	return snap
}

// SetRaw inserts an entry directly (used by snapshot/AOF restore).
// It does NOT emit events or touch the AOF.
func (s *Store) SetRaw(key string, entry *Entry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove any existing TTL tracking for this key
	if old, ok := s.ttlIndex[key]; ok {
		heap.Remove(s.ttlHeap, old.index)
		delete(s.ttlIndex, key)
	}

	s.data[key] = entry

	if entry.ExpiresAt > 0 {
		item := &TTLItem{key: key, expiresAt: entry.ExpiresAt}
		heap.Push(s.ttlHeap, item)
		s.ttlIndex[key] = item
	}
}
