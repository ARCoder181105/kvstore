package aof

import (
	"context"
	"encoding/binary"
	"os"
	"time"

	"github.com/ARCoder181105/kvstore/internal/store"
)

// this is a appen-only-file
// that is used create logs of operation on the redis database for persistance

// Must write to AOF — these change data:

// SET — creates or updates a key
// DEL — removes a key
// EXPIRE — changes a key's TTL
// INCR — changes a key's value
// MSET — creates or updates multiple keys

type AOFEntry struct { // timeStamp | CmdID | Key | Value | TTL
	Timestamp int64
	CmdID     byte
	Key       string
	Value     []byte
	TTL       int64
}

type AOFWriter struct {
	file   *os.File
	ch     chan AOFEntry
	ticker *time.Ticker
}

func NewAOFWriter(path string) (*AOFWriter, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return &AOFWriter{
		file:   file,
		ch:     make(chan AOFEntry, 1024),
		ticker: time.NewTicker(1 * time.Second),
	}, nil
}

func (a *AOFWriter) Start(ctx context.Context) {

	for {
		select {
		case entry := <-a.ch:
			a.writeEntry(entry)
		case <-a.ticker.C:
			a.file.Sync()
		case <-ctx.Done():
			a.file.Sync()
			a.file.Close()
			return
		}
	}

}

func (a *AOFWriter) Append(entry AOFEntry) {
	select {
	case a.ch <- entry:
	default:
		// Drop the entry if the channel is full (non-blocking)
		// Optionally, log or handle dropped entries here
		
	}
}

func Replay(path string, s *store.Store) error

func (a *AOFWriter) writeEntry(entry AOFEntry) error {
	buf := make([]byte, 8)

	// write timestamp
	binary.BigEndian.PutUint64(buf, uint64(entry.Timestamp))
	a.file.Write(buf)

	// write command ID
	a.file.Write([]byte{entry.CmdID})

	// write key length + key
	binary.BigEndian.PutUint32(buf[:4], uint32(len(entry.Key)))
	a.file.Write(buf[:4])
	a.file.Write([]byte(entry.Key))

	// write value length + value
	binary.BigEndian.PutUint32(buf[:4], uint32(len(entry.Value)))
	a.file.Write(buf[:4])
	a.file.Write(entry.Value)

	// write TTL
	binary.BigEndian.PutUint64(buf, uint64(entry.TTL))
	a.file.Write(buf)

	return nil
}
