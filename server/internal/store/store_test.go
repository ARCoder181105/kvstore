package store

import (
	"testing"
	"time"
)

func TestSetGet(t *testing.T) {
	s := New()
	s.Set("name", []byte("Alice"), 0)
	val, ok := s.Get("name")
	if !ok || string(val) != "Alice" {
		t.Fail()
	}
}

func TestDelete(t *testing.T) {
	s := New()
	s.Set("name", []byte("Alice"), 0)
	s.Delete("name")
	_, ok := s.Get("name")
	if ok {
		t.Fail()
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
	val, _ := s.Incr("counter")
	if val != 1 {
		t.Fail()
	}
	val, _ = s.Incr("counter")
	if val != 2 {
		t.Fail()
	}
}

func TestKeys(t *testing.T) {
	s := New()
	s.Set("user:alice", []byte("1"), 0)
	s.Set("user:bob", []byte("2"), 0)
	s.Set("city", []byte("Mumbai"), 0)
	keys := s.Keys("user:*")
	if len(keys) != 2 {
		t.Fail()
	}
}
