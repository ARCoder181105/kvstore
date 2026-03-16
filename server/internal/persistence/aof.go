package aof

import (
	"context"
	"encoding/binary"
	"io"
	"os"
	"time"

	"github.com/ARCoder181105/kvstore/internal/protocol"
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
		// Buffer full — entry dropped. At most 1024 ops at risk on crash.
		// Acceptable for this project. In production, you'd block or alert.
	}
}

func Replay(path string, s *store.Store) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no AOF log yet, that's fine on first startup
		}
		return err
	}
	defer file.Close()

	buff := make([]byte, 8)
	
	for {
		entry := &AOFEntry{}

		// timeStamp
		_, err := io.ReadFull(file, buff)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		// entry.Timestamp = int64(binary.BigEndian.Uint64(buff))

		// Cmd
		_, err = io.ReadFull(file, buff[:1])
		if err != nil {
			return err
		}
		entry.CmdID = buff[0]

		//keyLen
		_, err = io.ReadFull(file, buff[:4])
		if err != nil {
			return err
		}
		keyLen := binary.BigEndian.Uint32(buff[:4])

		// Key
		keyBuf := make([]byte, keyLen)
		_, err = io.ReadFull(file, keyBuf)
		if err != nil {
			return err
		}
		entry.Key = string(keyBuf)

		// valueLen
		_, err = io.ReadFull(file, buff[:4])
		if err != nil {
			return err
		}
		valueLen := binary.BigEndian.Uint32(buff[:4])

		// Value
		value := make([]byte, valueLen)
		_, err = io.ReadFull(file, value)
		if err != nil {
			return err
		}
		entry.Value = value

		// TTL
		_, err = io.ReadFull(file, buff)
		if err != nil {
			return err
		}
		entry.TTL = int64(binary.BigEndian.Uint64(buff))

		switch entry.CmdID {
		case protocol.CmdSet:
			s.Set(entry.Key, entry.Value, entry.TTL)
		case protocol.CmdDel:
			s.Delete(entry.Key)
		case protocol.CmdExpire:
			s.Expire(entry.Key, entry.TTL)
		case protocol.CmdIncr:
			s.Incr(entry.Key)
		}
	}

	return nil
}

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
