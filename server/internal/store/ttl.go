package store

import (
	"container/heap"
	"context"
	"time"
)

type TTLItem struct {
	key       string
	expiresAt int64 // Unix nanoseconds
	index     int   // position in the heap — needed for O(log n) removal
}

type TTLHeap []*TTLItem

func (h TTLHeap) Len() int {
	return len(h)
}

func (h TTLHeap) Less(i, j int) bool {
	return h[i].expiresAt < h[j].expiresAt
}

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
	item.index = -1 // mark as removed
	*h = old[:n-1]
	return item
}

func (s *Store) StartEviction(ctx context.Context) {
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
			sleepDuration := time.Duration(nextExpiry - now)
			select {
			case <-ctx.Done():
				return
			case <-time.After(sleepDuration):
			}
		}

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
