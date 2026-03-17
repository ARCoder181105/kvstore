package store

import (
	"container/heap"
	"context"
	"time"
)

type TTLItem struct {
	key       string
	expiresAt int64 // Unix nanoseconds (absolute)
	index     int   // position in the heap — needed for O(log n) removal
}

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
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // mark as removed from heap
	*h = old[:n-1]
	return item
}

// StartEviction runs the background TTL eviction loop.
// instead of spinning every 100ms on an empty heap, it blocks on
// s.notify until a TTL key is added — zero CPU when there is nothing to evict.
func (s *Store) StartEviction(ctx context.Context) {
	for {
		// --- Wait until the heap has at least one entry ---
		s.mu.RLock()
		empty := s.ttlHeap.Len() == 0
		s.mu.RUnlock()

		if empty {
			// Block until a TTL key is added (or shutdown)
			select {
			case <-ctx.Done():
				return
			case <-s.notify:
				// A TTL key was just pushed; loop back and check the heap
				continue
			}
		}

		// --- Peek at the soonest expiry ---
		s.mu.RLock()
		if s.ttlHeap.Len() == 0 {
			s.mu.RUnlock()
			continue
		}
		nextExpiry := (*s.ttlHeap)[0].expiresAt
		s.mu.RUnlock()

		now := time.Now().UnixNano()
		if nextExpiry > now {
			sleepDuration := time.Duration(nextExpiry - now)
			select {
			case <-ctx.Done():
				return
			case <-time.After(sleepDuration):
				// Might have woken early; will re-check below
			case <-s.notify:
				// A new TTL key was added — it might expire sooner; re-evaluate
				continue
			}
		}

		// --- Evict all keys whose deadline has passed ---
		s.mu.Lock()
		for s.ttlHeap.Len() > 0 && (*s.ttlHeap)[0].expiresAt <= time.Now().UnixNano() {
			item := heap.Pop(s.ttlHeap).(*TTLItem)
			delete(s.data, item.key)
			delete(s.ttlIndex, item.key)
			select {
			case s.events <- Event{Type: EventExpired, Key: item.key}:
			default:
			}
		}
		s.mu.Unlock()
	}
}
