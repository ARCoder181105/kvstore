package aof

import (
	"os"
	"testing"
	"time"

	"github.com/ARCoder181105/kvstore/internal/protocol"
	"github.com/ARCoder181105/kvstore/internal/store"
)

func TestSnapshotSaveAndLoad(t *testing.T) {
	path := "./test_snapshot.db"
	defer os.Remove(path)
	defer os.Remove(path + ".tmp")

	s := store.New()
	s.Set("city", []byte("Mumbai"), 0)
	s.Set("name", []byte("Alice"), 0)
	s.Set("counter", []byte("42"), 0)

	if err := Save(s, path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	s2 := store.New()
	if err := Load(path, s2); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	for key, expected := range map[string]string{
		"city": "Mumbai", "name": "Alice", "counter": "42",
	} {
		val, ok := s2.Get(key)
		if !ok {
			t.Errorf("key %q not found after load", key)
		} else if string(val) != expected {
			t.Errorf("key %q: got %q want %q", key, string(val), expected)
		}
	}
}

func TestSnapshotLoadMissingFile(t *testing.T) {
	s := store.New()
	if err := Load("./does_not_exist.db", s); err != nil {
		t.Fatalf("expected nil for missing snapshot, got: %v", err)
	}
}

func TestAOFReplay(t *testing.T) {
	path := "./test_aof.log"
	defer os.Remove(path)

	aofWriter, err := NewAOFWriter(path)
	if err != nil {
		t.Fatalf("NewAOFWriter failed: %v", err)
	}

	aofWriter.Append(AOFEntry{
		Timestamp: time.Now().UnixNano(),
		CmdID:     protocol.CmdSet,
		Key:       "city",
		Value:     []byte("Mumbai"),
		ExpiresAt: 0,
	})
	aofWriter.Append(AOFEntry{
		Timestamp: time.Now().UnixNano(),
		CmdID:     protocol.CmdSet,
		Key:       "name",
		Value:     []byte("Alice"),
		ExpiresAt: 0,
	})
	aofWriter.Append(AOFEntry{
		Timestamp: time.Now().UnixNano(),
		CmdID:     protocol.CmdSet,
		Key:       "counter",
		Value:     []byte("42"),
		ExpiresAt: 0,
	})

	// Flush manually (Start() goroutine not running in tests)
	for i := 0; i < 3; i++ {
		aofWriter.writeEntry(<-aofWriter.ch)
	}
	aofWriter.file.Sync()
	aofWriter.file.Close()

	s := store.New()
	if err := Replay(path, s); err != nil {
		t.Fatalf("Replay failed: %v", err)
	}

	for key, expected := range map[string]string{
		"city": "Mumbai", "name": "Alice", "counter": "42",
	} {
		val, ok := s.Get(key)
		if !ok {
			t.Errorf("key %q not found after replay", key)
		} else if string(val) != expected {
			t.Errorf("key %q: got %q want %q", key, string(val), expected)
		}
	}
}

func TestAOFReplayMissingFile(t *testing.T) {
	s := store.New()
	if err := Replay("./does_not_exist.log", s); err != nil {
		t.Fatalf("expected nil for missing AOF, got: %v", err)
	}
}

func TestAOFReplayDEL(t *testing.T) {
	path := "./test_aof_del.log"
	defer os.Remove(path)

	aofWriter, err := NewAOFWriter(path)
	if err != nil {
		t.Fatalf("NewAOFWriter failed: %v", err)
	}

	aofWriter.Append(AOFEntry{Timestamp: time.Now().UnixNano(), CmdID: protocol.CmdSet, Key: "temp", Value: []byte("value")})
	aofWriter.Append(AOFEntry{Timestamp: time.Now().UnixNano(), CmdID: protocol.CmdDel, Key: "temp"})

	for i := 0; i < 2; i++ {
		aofWriter.writeEntry(<-aofWriter.ch)
	}
	aofWriter.file.Sync()
	aofWriter.file.Close()

	s := store.New()
	if err := Replay(path, s); err != nil {
		t.Fatalf("Replay failed: %v", err)
	}

	if _, ok := s.Get("temp"); ok {
		t.Error("key 'temp' should be deleted after replay")
	}
}

// AOF replay with absolute ExpiresAt preserves the deadline correctly.
// A key with 10 seconds remaining at write time should still have ~10 seconds
// remaining after immediate replay (not 10s added on top of now again).
func TestAOFReplayPreservesExpiresAt(t *testing.T) {
	path := "./test_aof_expires.log"
	defer os.Remove(path)

	aofWriter, err := NewAOFWriter(path)
	if err != nil {
		t.Fatalf("NewAOFWriter failed: %v", err)
	}

	// Simulate a key that was set with 10s TTL — store the absolute ExpiresAt
	futureExpiry := time.Now().Add(10 * time.Second).UnixNano()
	aofWriter.Append(AOFEntry{
		Timestamp: time.Now().UnixNano(),
		CmdID:     protocol.CmdSet,
		Key:       "session",
		Value:     []byte("tok123"),
		ExpiresAt: futureExpiry,
	})

	aofWriter.writeEntry(<-aofWriter.ch)
	aofWriter.file.Sync()
	aofWriter.file.Close()

	s := store.New()
	if err := Replay(path, s); err != nil {
		t.Fatalf("Replay failed: %v", err)
	}

	val, ok := s.Get("session")
	if !ok {
		t.Fatal("key should still be alive after replay")
	}
	if string(val) != "tok123" {
		t.Fatalf("unexpected value: %s", val)
	}

	// Remaining TTL should be close to 10s, not 20s (old bug: would add 10s again)
	remaining := s.TTL("session")
	tenSeconds := int64(10 * time.Second)
	if remaining > tenSeconds+int64(500*time.Millisecond) {
		t.Fatalf("TTL is %v ns — looks like it was doubled (old bug)", remaining)
	}
}

// AOF replay with an already-expired ExpiresAt deletes the key.
func TestAOFReplayExpiredKeyIsSkipped(t *testing.T) {
	path := "./test_aof_expired.log"
	defer os.Remove(path)

	aofWriter, err := NewAOFWriter(path)
	if err != nil {
		t.Fatalf("NewAOFWriter failed: %v", err)
	}

	pastExpiry := time.Now().Add(-5 * time.Second).UnixNano() // already expired
	aofWriter.Append(AOFEntry{
		Timestamp: time.Now().UnixNano(),
		CmdID:     protocol.CmdSet,
		Key:       "stale",
		Value:     []byte("old"),
		ExpiresAt: pastExpiry,
	})

	aofWriter.writeEntry(<-aofWriter.ch)
	aofWriter.file.Sync()
	aofWriter.file.Close()

	s := store.New()
	if err := Replay(path, s); err != nil {
		t.Fatalf("Replay failed: %v", err)
	}

	// The key was loaded by SetRaw (with a past ExpiresAt), so a Get should
	// find it expired and return false.
	_, ok := s.Get("stale")
	if ok {
		t.Fatal("expired key should not be returned by Get after replay")
	}
}
