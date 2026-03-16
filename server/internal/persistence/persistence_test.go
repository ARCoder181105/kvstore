package aof

import (
	"os"
	"testing"
	"time"

	"github.com/ARCoder181105/kvstore/internal/protocol"
	"github.com/ARCoder181105/kvstore/internal/store"
)

func TestSnapshotSaveAndLoad(t *testing.T) {
	// create a temp path for the snapshot
	path := "./test_snapshot.db"
	defer os.Remove(path)
	defer os.Remove(path + ".tmp")

	// create store and set some keys
	s := store.New()
	s.Set("city", []byte("Mumbai"), 0)
	s.Set("name", []byte("Alice"), 0)
	s.Set("counter", []byte("42"), 0)

	// save snapshot
	if err := Save(s, path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// create a new empty store and load snapshot into it
	s2 := store.New()
	if err := Load(path, s2); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// verify all keys are present
	cases := map[string]string{
		"city":    "Mumbai",
		"name":    "Alice",
		"counter": "42",
	}

	for key, expected := range cases {
		val, ok := s2.Get(key)
		if !ok {
			t.Errorf("key %q not found after load", key)
			continue
		}
		if string(val) != expected {
			t.Errorf("key %q: got %q want %q", key, string(val), expected)
		}
	}
}

func TestSnapshotLoadMissingFile(t *testing.T) {
	// loading a non-existent snapshot should return nil, not an error
	s := store.New()
	err := Load("./does_not_exist.db", s)
	if err != nil {
		t.Fatalf("expected nil error for missing snapshot, got: %v", err)
	}
}

func TestAOFReplay(t *testing.T) {
	// create a temp path for the AOF log
	path := "./test_aof.log"
	defer os.Remove(path)

	// create AOF writer and start it
	aofWriter, err := NewAOFWriter(path)
	if err != nil {
		t.Fatalf("NewAOFWriter failed: %v", err)
	}

	// append 3 SET entries manually
	aofWriter.Append(AOFEntry{
		Timestamp: time.Now().UnixNano(),
		CmdID:     protocol.CmdSet,
		Key:       "city",
		Value:     []byte("Mumbai"),
		TTL:       0,
	})
	aofWriter.Append(AOFEntry{
		Timestamp: time.Now().UnixNano(),
		CmdID:     protocol.CmdSet,
		Key:       "name",
		Value:     []byte("Alice"),
		TTL:       0,
	})
	aofWriter.Append(AOFEntry{
		Timestamp: time.Now().UnixNano(),
		CmdID:     protocol.CmdSet,
		Key:       "counter",
		Value:     []byte("42"),
		TTL:       0,
	})

	// give the AOF writer goroutine time to flush entries to disk
	// we need to write entries directly since Start() is not running
	// so flush manually by draining the channel
	for i := 0; i < 3; i++ {
		entry := <-aofWriter.ch
		aofWriter.writeEntry(entry)
	}
	aofWriter.file.Sync()
	aofWriter.file.Close()

	// create a new empty store and replay AOF into it
	s := store.New()
	if err := Replay(path, s); err != nil {
		t.Fatalf("Replay failed: %v", err)
	}

	// verify all keys are present
	cases := map[string]string{
		"city":    "Mumbai",
		"name":    "Alice",
		"counter": "42",
	}

	for key, expected := range cases {
		val, ok := s.Get(key)
		if !ok {
			t.Errorf("key %q not found after replay", key)
			continue
		}
		if string(val) != expected {
			t.Errorf("key %q: got %q want %q", key, string(val), expected)
		}
	}
}

func TestAOFReplayMissingFile(t *testing.T) {
	// replaying a non-existent AOF should return nil, not an error
	s := store.New()
	err := Replay("./does_not_exist.log", s)
	if err != nil {
		t.Fatalf("expected nil error for missing AOF, got: %v", err)
	}
}

func TestAOFReplayDEL(t *testing.T) {
	path := "./test_aof_del.log"
	defer os.Remove(path)

	aofWriter, err := NewAOFWriter(path)
	if err != nil {
		t.Fatalf("NewAOFWriter failed: %v", err)
	}

	// SET a key then DEL it
	aofWriter.Append(AOFEntry{
		Timestamp: time.Now().UnixNano(),
		CmdID:     protocol.CmdSet,
		Key:       "temp",
		Value:     []byte("value"),
	})
	aofWriter.Append(AOFEntry{
		Timestamp: time.Now().UnixNano(),
		CmdID:     protocol.CmdDel,
		Key:       "temp",
	})

	// flush manually
	for i := 0; i < 2; i++ {
		entry := <-aofWriter.ch
		aofWriter.writeEntry(entry)
	}
	aofWriter.file.Sync()
	aofWriter.file.Close()

	// replay into new store
	s := store.New()
	if err := Replay(path, s); err != nil {
		t.Fatalf("Replay failed: %v", err)
	}

	// key should NOT exist — it was deleted
	_, ok := s.Get("temp")
	if ok {
		t.Error("expected key 'temp' to be deleted after replay, but it exists")
	}
}
