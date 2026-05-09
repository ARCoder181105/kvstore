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

// StartEviction launches one background eviction goroutine per shard.
// Each goroutine independently manages its shard's TTL heap so that
// eviction work is spread across all 16 shards concurrently.
func (s *Store) StartEviction(ctx context.Context) {
	for i := range s.shards {
		go s.runShardEviction(ctx, &s.shards[i])
	}
}

// runShardEviction is the eviction loop for a single shard.
// It blocks on sh.notify when the heap is empty to avoid spinning.
func (s *Store) runShardEviction(ctx context.Context, sh *shard) {
	for {
		// --- Wait until the shard has at least one TTL entry ---
		sh.mu.RLock()
		empty := sh.ttlHeap.Len() == 0
		sh.mu.RUnlock()

		if empty {
			// Block until a TTL key is added (or shutdown).
			select {
			case <-ctx.Done():
				return
			case <-sh.notify:
				// A TTL key was just pushed; loop back and check the heap.
				continue
			}
		}

		// --- Peek at the soonest expiry in this shard ---
		sh.mu.RLock()
		if sh.ttlHeap.Len() == 0 {
			sh.mu.RUnlock()
			continue
		}
		nextExpiry := (*sh.ttlHeap)[0].expiresAt
		sh.mu.RUnlock()

		now := time.Now().UnixNano()
		if nextExpiry > now {
			sleepDuration := time.Duration(nextExpiry - now)
			select {
			case <-ctx.Done():
				return
			case <-time.After(sleepDuration):
				// May have woken early; will re-check below.
			case <-sh.notify:
				// A new TTL key was added — it might expire sooner; re-evaluate.
				continue
			}
		}

		// --- Evict all keys in this shard whose deadline has passed ---
		expired := make([]string, 0)
		sh.mu.Lock()
		for sh.ttlHeap.Len() > 0 && (*sh.ttlHeap)[0].expiresAt <= time.Now().UnixNano() {
			item := heap.Pop(sh.ttlHeap).(*TTLItem)
			delete(sh.data, item.key)
			delete(sh.ttlIndex, item.key)
			expired = append(expired, item.key)
		}
		sh.mu.Unlock()

		for _, key := range expired {
			s.publish(Event{Type: EventExpired, Key: key, Timestamp: time.Now().UTC()})
		}
	}
}
