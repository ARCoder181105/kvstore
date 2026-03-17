package aof

import (
	"encoding/gob"
	"os"
	"time"

	"github.com/ARCoder181105/kvstore/internal/store"
)

func Save(s *store.Store, path string) error {

	data := s.Snapshot()
	file, err := os.Create(path + ".tmp")
	if err != nil {
		return err
	}

	encoder := gob.NewEncoder(file)

	if err := encoder.Encode(data); err != nil {
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

	if err := os.Rename(path+".tmp", path); err != nil {
		return err
	}

	return nil
}

func Load(path string, s *store.Store) error {

	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	var data map[string]*store.Entry
	err = decoder.Decode(&data)
	if err != nil {
		return err
	}

	for k, v := range data {
		if v.ExpiresAt > 0 && v.ExpiresAt <= time.Now().UnixNano() {
			continue // skip already expired keys
		}
		s.SetRaw(k, v)
	}

	return nil
}
