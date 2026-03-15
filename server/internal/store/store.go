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
	once     sync.Once // ensures background goroutines start exactly once
}

func New() *Store {
	s := &Store{
		data:     make(map[string]*Entry),
		ttlHeap:  &TTLHeap{},
		ttlIndex: make(map[string]*TTLItem),
		events:   make(chan Event, 256),
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

	if ttlNs > 0 {

		if old, ok := s.ttlIndex[key]; ok {
			heap.Remove(s.ttlHeap, old.index)
			delete(s.ttlIndex, key)
		}

		ttlElement := &TTLItem{
			key:       key,
			expiresAt: s.data[key].ExpiresAt,
		}
		heap.Push(s.ttlHeap, ttlElement)
		s.ttlIndex[key] = ttlElement
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

func (s *Store) Get(key string) ([]byte, bool) {

	s.mu.RLock()
	entry, ok := s.data[key]
	s.mu.RUnlock()

	if !ok {
		return nil, false
	}

	// Lazy expiry check — do not hold the lock while checking
	if entry.IsExpired() {
		// As Entry is expired delete it
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

func (s *Store) TTL(key string) int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exist := s.data[key]

	if !exist {
		return -2 // Key doesn't Exist
	}

	if entry.ExpiresAt == 0 {
		return -1 // Key has no ttl
	}

	return entry.ExpiresAt - time.Now().UnixNano()
}

func (s *Store) Expire(key string, ttlNs int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exist := s.data[key]
	if !exist {
		return false
	}

	entry.ExpiresAt = expiresAt(ttlNs)

	// Update TTL heap/index if ttlNs > 0
	if ttlNs > 0 {
		if old, ok := s.ttlIndex[key]; ok {
			heap.Remove(s.ttlHeap, old.index)
			delete(s.ttlIndex, key)
		}

		ttlElement := &TTLItem{
			key:       key,
			expiresAt: entry.ExpiresAt,
		}

		heap.Push(s.ttlHeap, ttlElement)
		s.ttlIndex[key] = ttlElement
	} else {
		// Remove from TTL heap/index if ttlNs == 0
		if item, ok := s.ttlIndex[key]; ok {
			heap.Remove(s.ttlHeap, item.index)
			delete(s.ttlIndex, key)
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

	var returnSet [][]byte

	for _, key := range keys {
		entry, exist := s.data[key]

		if !exist || entry.IsExpired() {
			returnSet = append(returnSet, nil)
		} else {
			returnSet = append(returnSet, entry.Value)
		}

	}

	return returnSet
}

func (s *Store) MSet(entries map[string][]byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for key, value := range entries {
		s.data[key] = &Entry{
			Value:     value,
			ExpiresAt: 0, // no TTL on MSet
		}

		// if key had an old TTL, clean it up
		if old, ok := s.ttlIndex[key]; ok {
			heap.Remove(s.ttlHeap, old.index)
			delete(s.ttlIndex, key)
		}

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

	// if key doesn't exist, create it with value 1
	if !exists {
		s.data[key] = &Entry{Value: []byte("1")}
		select {
		case s.events <- Event{Type: EventSet, Key: key, Value: "1"}:
		default:
		}
		return 1, nil
	}

	val, err := parseInt64(entry.Value)
	if err != nil {
		return 0, err
	}

	val++
	entry.Value = []byte(fmt.Sprintf("%d", val))

	select {
	case s.events <- Event{Type: EventSet, Key: key, Value: string(entry.Value)}:
	default:
	}

	return val, nil
}

func parseInt64(val []byte) (int64, error) {
	return strconv.ParseInt(string(val), 10, 64)
}

func (s *Store) Keys(pattern string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := []string{}

	for key, entry := range s.data {
		// skip expired keys
		if entry.IsExpired() {
			continue
		}

		// check if key matches the pattern
		matched, err := filepath.Match(pattern, key)
		if err != nil {
			continue // invalid pattern, skip
		}

		if matched {
			keys = append(keys, key)
		}
	}

	return keys
}
