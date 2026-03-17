package store

import (
	"context"
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
