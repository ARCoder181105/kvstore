package server

import (
	"bufio"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	aof "github.com/ARCoder181105/kvstore/internal/persistence"
	"github.com/ARCoder181105/kvstore/internal/protocol"
)

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		cmd, err := protocol.ReadCommand(reader)
		if err == io.EOF {
			return
		}
		if err != nil {
			return
		}

		resp := s.executeCommand(cmd)
		if err := protocol.WriteResponse(conn, resp); err != nil {
			return
		}
	}
}

func (s *Server) executeCommand(cmd *protocol.Command) *protocol.Response {
	switch cmd.ID {

	case protocol.CmdSet:
		s.store.Set(cmd.Key, cmd.Value, cmd.TTL)
		if s.aofWriter != nil {
			// store the absolute ExpiresAt (not the TTL duration) so that
			// replay restores the correct deadline regardless of when it runs.
			var expiresAt int64
			if cmd.TTL > 0 {
				expiresAt = time.Now().UnixNano() + cmd.TTL
			}
			s.aofWriter.Append(aof.AOFEntry{
				Timestamp: time.Now().UnixNano(),
				CmdID:     protocol.CmdSet,
				Key:       cmd.Key,
				Value:     cmd.Value,
				ExpiresAt: expiresAt,
			})
		}
		return &protocol.Response{Status: protocol.StatusOK}

	case protocol.CmdGet:
		val, ok := s.store.Get(cmd.Key)
		if !ok {
			return &protocol.Response{Status: protocol.StatusNull}
		}
		return &protocol.Response{Status: protocol.StatusValue, Payload: val}

	case protocol.CmdDel:
		ok := s.store.Delete(cmd.Key)
		if ok && s.aofWriter != nil {
			s.aofWriter.Append(aof.AOFEntry{
				Timestamp: time.Now().UnixNano(),
				CmdID:     protocol.CmdDel,
				Key:       cmd.Key,
			})
		}
		// return StatusInt with the number of deleted keys (Redis behavior)
		// Previously returned StatusOK with "0"/"1" which printResponse ignored.
		if ok {
			return &protocol.Response{Status: protocol.StatusInt, Payload: []byte("1")}
		}
		return &protocol.Response{Status: protocol.StatusInt, Payload: []byte("0")}

	case protocol.CmdExpire:
		ok := s.store.Expire(cmd.Key, cmd.TTL)
		if !ok {
			return &protocol.Response{Status: protocol.StatusError, Payload: []byte("key not found")}
		}
		if s.aofWriter != nil {
			var expiresAt int64
			if cmd.TTL > 0 {
				expiresAt = time.Now().UnixNano() + cmd.TTL
			}
			s.aofWriter.Append(aof.AOFEntry{
				Timestamp: time.Now().UnixNano(),
				CmdID:     protocol.CmdExpire,
				Key:       cmd.Key,
				ExpiresAt: expiresAt,
			})
		}
		return &protocol.Response{Status: protocol.StatusOK}

	case protocol.CmdTTL:
		// store.TTL returns nanoseconds; convert to seconds for Redis parity.
		// -1 (no expiry) and -2 (missing/expired) pass through unchanged.
		ttlNs := s.store.TTL(cmd.Key)
		var ttlSec int64
		if ttlNs >= 0 {
			ttlSec = ttlNs / int64(time.Second)
			if ttlSec == 0 {
				ttlSec = 1 // key is alive but < 1s remaining — round up
			}
		} else {
			ttlSec = ttlNs // -1 or -2
		}
		return &protocol.Response{
			Status:  protocol.StatusInt,
			Payload: []byte(strconv.FormatInt(ttlSec, 10)),
		}

	case protocol.CmdKeys:
		keys := s.store.Keys(cmd.Key)
		return &protocol.Response{
			Status:  protocol.StatusArray,
			Payload: []byte(strings.Join(keys, "\n")),
		}

	case protocol.CmdIncr:
		val, err := s.store.Incr(cmd.Key)
		if err != nil {
			return &protocol.Response{Status: protocol.StatusError, Payload: []byte(err.Error())}
		}
		if s.aofWriter != nil {
			// log the final value as CmdSet (Redis-style AOF rewriting).
			// Replay uses CmdSet for these entries, so there is no double-increment.
			s.aofWriter.Append(aof.AOFEntry{
				Timestamp: time.Now().UnixNano(),
				CmdID:     protocol.CmdSet,
				Key:       cmd.Key,
				Value:     []byte(strconv.FormatInt(val, 10)),
				ExpiresAt: 0,
			})
		}
		return &protocol.Response{
			Status:  protocol.StatusInt,
			Payload: []byte(strconv.FormatInt(val, 10)),
		}

	case protocol.CmdMSet:
		return &protocol.Response{Status: protocol.StatusError, Payload: []byte("not implemented")}

	case protocol.CmdMGet:
		return &protocol.Response{Status: protocol.StatusError, Payload: []byte("not implemented")}

	case protocol.CmdPing:
		return &protocol.Response{Status: protocol.StatusValue, Payload: []byte(s.store.Ping())}

	default:
		return &protocol.Response{Status: protocol.StatusError, Payload: []byte("unknown command")}
	}
}
