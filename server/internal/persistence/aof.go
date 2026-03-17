package aof

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ARCoder181105/kvstore/internal/protocol"
	"github.com/ARCoder181105/kvstore/internal/store"
)

// AOFEntry is written to the append-only log for every mutation.
//
// The field is now ExpiresAt (absolute Unix nanoseconds), NOT a TTL
// duration. This means Replay can call store.SetRaw with the original
// deadline rather than re-computing it relative to the replay time, which
// would give keys an unintended extra lease after a crash.
//
// Wire format (binary, big-endian):
//
//	[8] Timestamp  int64   — wall clock when the command was received
//	[1] CmdID      byte    — protocol.CmdSet / CmdDel / CmdExpire
//	[4] KeyLen     uint32
//	[N] Key        bytes
//	[4] ValueLen   uint32
//	[M] Value      bytes
//	[8] ExpiresAt  int64   — absolute Unix nanoseconds; 0 = no expiry
type AOFEntry struct {
	Timestamp int64
	CmdID     byte
	Key       string
	Value     []byte
	ExpiresAt int64 // absolute, NOT a duration
}

type AOFWriter struct {
	file       *os.File
	ch         chan AOFEntry
	syncTicker *time.Ticker
}

func NewAOFWriter(path string) (*AOFWriter, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &AOFWriter{
		file:       file,
		ch:         make(chan AOFEntry, 1024),
		syncTicker: time.NewTicker(time.Second),
	}, nil
}

func (a *AOFWriter) Start(ctx context.Context) {
	for {
		select {
		case entry := <-a.ch:
			if err := a.writeEntry(entry); err != nil {
				fmt.Printf("AOF write error: %v\n", err)
			}
		case <-a.syncTicker.C:
			a.file.Sync()
		case <-ctx.Done():
			// Drain remaining entries before closing
			for {
				select {
				case entry := <-a.ch:
					a.writeEntry(entry)
				default:
					a.file.Sync()
					a.file.Close()
					return
				}
			}
		}
	}
}

// Append queues an AOF entry for async writing.
// Non-blocking: drops the entry if the buffer is full (at most 1024 ops at risk).
func (a *AOFWriter) Append(entry AOFEntry) {
	select {
	case a.ch <- entry:
	default:
	}
}

// Replay reads the AOF log and restores its entries into the store.
//
// Uses store.SetRaw for SET entries, passing the stored absolute
// ExpiresAt directly — no call to expiresAt(duration) which would shift
// the deadline forward by the replay wall-clock time.
// DEL and EXPIRE entries are applied via the normal store methods.
func Replay(path string, s *store.Store) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	buf8 := make([]byte, 8)
	buf4 := make([]byte, 4)
	buf1 := make([]byte, 1)

	for {
		// Timestamp (8 bytes) — read but not used for anything currently
		if _, err := io.ReadFull(file, buf8); err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("AOF replay: reading timestamp: %w", err)
		}
		// (timestamp not used — just consumed)

		// CmdID (1 byte)
		if _, err := io.ReadFull(file, buf1); err != nil {
			return fmt.Errorf("AOF replay: reading cmdID: %w", err)
		}
		cmdID := buf1[0]

		// KeyLen (4 bytes)
		if _, err := io.ReadFull(file, buf4); err != nil {
			return fmt.Errorf("AOF replay: reading key len: %w", err)
		}
		keyLen := binary.BigEndian.Uint32(buf4)

		// Key
		keyBuf := make([]byte, keyLen)
		if _, err := io.ReadFull(file, keyBuf); err != nil {
			return fmt.Errorf("AOF replay: reading key: %w", err)
		}
		key := string(keyBuf)

		// ValueLen (4 bytes)
		if _, err := io.ReadFull(file, buf4); err != nil {
			return fmt.Errorf("AOF replay: reading value len: %w", err)
		}
		valueLen := binary.BigEndian.Uint32(buf4)

		// Value
		value := make([]byte, valueLen)
		if _, err := io.ReadFull(file, value); err != nil {
			return fmt.Errorf("AOF replay: reading value: %w", err)
		}

		// ExpiresAt (8 bytes) — absolute Unix nanoseconds
		if _, err := io.ReadFull(file, buf8); err != nil {
			return fmt.Errorf("AOF replay: reading expiresAt: %w", err)
		}
		expiresAt := int64(binary.BigEndian.Uint64(buf8))

		// Apply entry to store
		switch cmdID {
		case protocol.CmdSet:
			// Use SetRaw with the stored absolute ExpiresAt.
			// This avoids re-adding to time.Now(), which was the core TTL replay bug.
			s.SetRaw(key, &store.Entry{Value: value, ExpiresAt: expiresAt})

		case protocol.CmdDel:
			s.Delete(key)

		case protocol.CmdExpire:
			// ExpiresAt is absolute. Calculate remaining nanoseconds and call Expire
			// only if the key hasn't already expired; otherwise delete it.
			remaining := expiresAt - time.Now().UnixNano()
			if remaining <= 0 {
				s.Delete(key) // already past deadline
			} else {
				s.Expire(key, remaining)
			}
		}
	}

	return nil
}

func (a *AOFWriter) writeEntry(entry AOFEntry) error {
	buf8 := make([]byte, 8)
	buf4 := make([]byte, 4)

	// Timestamp
	binary.BigEndian.PutUint64(buf8, uint64(entry.Timestamp))
	if _, err := a.file.Write(buf8); err != nil {
		return err
	}

	// CmdID
	if _, err := a.file.Write([]byte{entry.CmdID}); err != nil {
		return err
	}

	// KeyLen + Key
	binary.BigEndian.PutUint32(buf4, uint32(len(entry.Key)))
	if _, err := a.file.Write(buf4); err != nil {
		return err
	}
	if _, err := a.file.Write([]byte(entry.Key)); err != nil {
		return err
	}

	// ValueLen + Value
	binary.BigEndian.PutUint32(buf4, uint32(len(entry.Value)))
	if _, err := a.file.Write(buf4); err != nil {
		return err
	}
	if len(entry.Value) > 0 {
		if _, err := a.file.Write(entry.Value); err != nil {
			return err
		}
	}

	// ExpiresAt (absolute)
	binary.BigEndian.PutUint64(buf8, uint64(entry.ExpiresAt))
	if _, err := a.file.Write(buf8); err != nil {
		return err
	}

	return nil
}
