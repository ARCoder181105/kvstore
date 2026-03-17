package aof

import (
	"encoding/gob"
	"os"
	"time"

	"github.com/ARCoder181105/kvstore/internal/store"
)

// Save serialises the entire store to disk atomically (write temp → rename).
// Only non-expired keys are included (store.Snapshot() already filters them).
func Save(s *store.Store, path string) error {
	data := s.Snapshot()

	file, err := os.Create(path + ".tmp")
	if err != nil {
		return err
	}

	enc := gob.NewEncoder(file)
	if err := enc.Encode(data); err != nil {
		file.Close()
		os.Remove(path + ".tmp")
		return err
	}

	if err := file.Sync(); err != nil {
		file.Close()
		os.Remove(path + ".tmp")
		return err
	}
	file.Close()

	return os.Rename(path+".tmp", path)
}

// Load deserialises a snapshot into the store.
// Entries whose absolute ExpiresAt is already in the past are skipped.
func Load(path string, s *store.Store) error {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()

	var data map[string]*store.Entry
	if err := gob.NewDecoder(file).Decode(&data); err != nil {
		return err
	}

	now := time.Now().UnixNano()
	for k, v := range data {
		if v.ExpiresAt > 0 && v.ExpiresAt <= now {
			continue // skip already-expired keys from snapshot
		}
		s.SetRaw(k, v)
	}

	return nil
}
