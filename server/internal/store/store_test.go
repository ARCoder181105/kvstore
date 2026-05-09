package store

import (
	"bytes"
	"context"
	"strconv"
	"testing"
	"time"
)

func TestSetGet(t *testing.T) {
	s := New()
	s.Set("name", []byte("Alice"), 0)
	val, ok := s.Get("name")
	if !ok || string(val) != "Alice" {
		t.Fatal("expected to get Alice")
	}
}

func TestDelete(t *testing.T) {
	s := New()
	s.Set("name", []byte("Alice"), 0)
	s.Delete("name")
	_, ok := s.Get("name")
	if ok {
		t.Fatal("expected key to be deleted")
	}
}

func TestTTLExpiry(t *testing.T) {
	s := New()
	s.Set("temp", []byte("value"), int64(100*time.Millisecond))
	time.Sleep(150 * time.Millisecond)
	_, ok := s.Get("temp")
	if ok {
		t.Fatal("expected key to be expired")
	}
}

func TestIncr(t *testing.T) {
	s := New()
	val, err := s.Incr("counter")
	if err != nil || val != 1 {
		t.Fatalf("expected 1, got %d err %v", val, err)
	}
	val, err = s.Incr("counter")
	if err != nil || val != 2 {
		t.Fatalf("expected 2, got %d err %v", val, err)
	}
}

func TestKeys(t *testing.T) {
	s := New()
	s.Set("user:alice", []byte("1"), 0)
	s.Set("user:bob", []byte("2"), 0)
	s.Set("city", []byte("Mumbai"), 0)
	keys := s.Keys("user:*")
	if len(keys) != 2 {
		t.Fatalf("expected 2 user keys, got %d", len(keys))
	}
}

// Set with TTL then overwrite with no TTL — old TTL item must be removed
func TestSetOverwriteClearsTTL(t *testing.T) {
	s := New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.StartEviction(ctx)

	// Set with 200ms TTL
	s.Set("k", []byte("v1"), int64(200*time.Millisecond))

	// Immediately overwrite with no TTL
	s.Set("k", []byte("v2"), 0)

	// Wait longer than the original TTL
	time.Sleep(300 * time.Millisecond)

	// give eviction goroutine time to notice cancellation
	cancel()
	time.Sleep(10 * time.Millisecond)

	val, ok := s.Get("k")
	if !ok {
		t.Fatal("key was evicted even though TTL was cleared — heap corruption bug")
	}
	if string(val) != "v2" {
		t.Fatalf("unexpected value: %s", val)
	}
}

// TTL command returns -2 for expired keys (not a positive stale nanosecond value)
func TestTTLExpiredKey(t *testing.T) {
	s := New()
	s.Set("x", []byte("v"), int64(50*time.Millisecond))
	time.Sleep(100 * time.Millisecond)
	ttl := s.TTL("x")
	if ttl != -2 {
		t.Fatalf("expected -2 for expired key, got %d", ttl)
	}
}

// TTL returns -1 for persistent key
func TestTTLPersistentKey(t *testing.T) {
	s := New()
	s.Set("x", []byte("v"), 0)
	if s.TTL("x") != -1 {
		t.Fatal("expected -1 for persistent key")
	}
}

// TTL returns -2 for missing key
func TestTTLMissingKey(t *testing.T) {
	s := New()
	if s.TTL("noexist") != -2 {
		t.Fatal("expected -2 for missing key")
	}
}

// Incr on an expired key resets to 1 (Redis behavior)
func TestIncrOnExpiredKey(t *testing.T) {
	s := New()
	s.Set("c", []byte("42"), int64(50*time.Millisecond))
	time.Sleep(100 * time.Millisecond)
	val, err := s.Incr("c")
	if err != nil || val != 1 {
		t.Fatalf("expected Incr on expired key to return 1, got %d err %v", val, err)
	}
}

// Delete returns false for missing key, true for existing key
func TestDeleteReturnValues(t *testing.T) {
	s := New()
	if s.Delete("noexist") {
		t.Fatal("expected false for deleting missing key")
	}
	s.Set("k", []byte("v"), 0)
	if !s.Delete("k") {
		t.Fatal("expected true for deleting existing key")
	}
}

// Concurrent Set + Get race test
func TestConcurrentSetGet(t *testing.T) {
	s := New()
	done := make(chan struct{})
	for i := 0; i < 50; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				s.Set("key", []byte("value"), 0)
				s.Get("key")
			}
			done <- struct{}{}
		}()
	}
	for i := 0; i < 50; i++ {
		<-done
	}
}

// Subscribes to the store, calls Set, reads from the channel, verifies the event has the correct Type and Key.
func TestSubscribeReceivesEvent(t *testing.T) {
	s := New()
	ch := s.Subscribe()

	s.Set("name", []byte("Alice"), 0)

	select {
	case event := <-ch:
		if event.Key != "name" || !bytes.Equal([]byte(event.Value), []byte("Alice")) {
			t.Fatalf("unexpected event: got key=%s value=%s", event.Key, event.Value)
		}
		if event.Type != EventSet {
			t.Fatalf("unexpected event type: got %s", event.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for event")
	}
}

// Subscribes, then immediately unsubscribes, calls Set, verifies nothing arrives on the channel within 100ms.
func TestUnsubscribeStopsEvents(t *testing.T) {
	s := New()
	ch := s.Subscribe()

	s.Unsubscribe(ch)

	s.Set("name", []byte("Alice"), 0)

	select {
	case event, ok := <-ch:
		if !ok {
			// channel was closed, this is expected — test passes
			return
		}
		t.Fatalf("expected no event after unsubscribe, but received: key=%s value=%s type=%s", event.Key, event.Value, event.Type)
	case <-time.After(100 * time.Millisecond):
	}
}

// Subscribes twice to get two channels, calls Set once, verifies both channels received the same event. This is the fan-out test.
func TestMultipleSubscribersAllReceive(t *testing.T) {
	s := New()
	ch1 := s.Subscribe()
	ch2 := s.Subscribe()

	s.Set("name", []byte("Alice"), 0)

	select {
	case event := <-ch1:
		if event.Type != EventSet || event.Key != "name" {
			t.Fatalf("ch1 unexpected event: %+v", event)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for event on ch1")
	}

	select {
	case event := <-ch2:
		if event.Type != EventSet || event.Key != "name" {
			t.Fatalf("ch2 unexpected event: %+v", event)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timed out waiting for event on ch2")
	}
}

// Subscribes but never reads from the channel, calls Set 100 times, verifies all 100 complete within 500ms. This confirms a full subscriber channel never blocks the store.
func TestSlowSubscriberDoesNotBlockStore(t *testing.T) {

	s := New()
	_ = s.Subscribe()

	start := time.Now()

	for range 100 {
		s.Set("name", []byte("Alice"), 0)
	}

	elapsed := time.Since(start)

	if elapsed > 500*time.Millisecond {
		t.Fatalf("store blocked for %v", elapsed)
	}

}

// ─────────────────────────────────────────────────────────────────────────────
// Benchmarks — run with: go test -bench=. -benchtime=5s ./internal/store/...
// Target: 400k+ ops/sec for Set and Get with the 16-shard store.
// ─────────────────────────────────────────────────────────────────────────────

// BenchmarkSet measures the raw Set throughput using random-looking keys
// distributed across all 16 shards.
func BenchmarkSet(b *testing.B) {
	s := New()
	val := []byte("benchmark_value_1234567890")
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			// Vary the key so we exercise all shards and avoid map hot-spots.
			key := "bench:" + strconv.Itoa(i%10000)
			s.Set(key, val, 0)
			i++
		}
	})
}

// BenchmarkGet measures the raw Get throughput after pre-populating the store.
func BenchmarkGet(b *testing.B) {
	s := New()
	val := []byte("benchmark_value_1234567890")
	for i := 0; i < 10000; i++ {
		s.Set("bench:"+strconv.Itoa(i), val, 0)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			s.Get("bench:" + strconv.Itoa(i%10000))
			i++
		}
	})
}

// BenchmarkMixed simulates a realistic 50% read / 50% write workload
// spread across all shards.
func BenchmarkMixed(b *testing.B) {
	s := New()
	val := []byte("benchmark_value_1234567890")
	for i := 0; i < 10000; i++ {
		s.Set("bench:"+strconv.Itoa(i), val, 0)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "bench:" + strconv.Itoa(i%10000)
			if i%2 == 0 {
				s.Set(key, val, 0)
			} else {
				s.Get(key)
			}
			i++
		}
	})
}
