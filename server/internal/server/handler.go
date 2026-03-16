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

func (s *Server) handleConn(conn net.Conn) { // ← no store param
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
		err = protocol.WriteResponse(conn, resp)
		if err != nil {
			return
		}
	}
}

func (s *Server) executeCommand(cmd *protocol.Command) *protocol.Response {
	switch cmd.ID {
	case protocol.CmdSet:
		s.store.Set(cmd.Key, cmd.Value, cmd.TTL)
		if s.aofWriter != nil {
			s.aofWriter.Append(aof.AOFEntry{
				Timestamp: time.Now().UnixNano(),
				CmdID:     protocol.CmdSet,
				Key:       cmd.Key,
				Value:     cmd.Value,
				TTL:       cmd.TTL,
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
		if !ok {
			return &protocol.Response{Status: protocol.StatusNull, Payload: []byte("0")}
		}
		return &protocol.Response{Status: protocol.StatusOK, Payload: []byte("1")}

	case protocol.CmdExpire:
		ok := s.store.Expire(cmd.Key, cmd.TTL)
		if ok && s.aofWriter != nil {
			s.aofWriter.Append(aof.AOFEntry{
				Timestamp: time.Now().UnixNano(),
				CmdID:     protocol.CmdExpire,
				Key:       cmd.Key,
				TTL:       cmd.TTL,
			})
		}
		if !ok {
			return &protocol.Response{Status: protocol.StatusError, Payload: []byte("key not found")}
		}
		return &protocol.Response{Status: protocol.StatusOK}

	case protocol.CmdTTL:
		ttl := s.store.TTL(cmd.Key)
		return &protocol.Response{Status: protocol.StatusInt, Payload: []byte(strconv.FormatInt(ttl, 10))}

	case protocol.CmdKeys:
		keys := s.store.Keys(cmd.Key)
		return &protocol.Response{Status: protocol.StatusArray, Payload: []byte(strings.Join(keys, "\n"))}

	case protocol.CmdIncr:
		val, err := s.store.Incr(cmd.Key)
		if err != nil {
			return &protocol.Response{Status: protocol.StatusError, Payload: []byte(err.Error())}
		}
		if s.aofWriter != nil {
			s.aofWriter.Append(aof.AOFEntry{
				Timestamp: time.Now().UnixNano(),
				CmdID:     protocol.CmdSet,
				Key:       cmd.Key,
				Value:     []byte(strconv.FormatInt(val, 10)),
			})
		}
		return &protocol.Response{Status: protocol.StatusInt, Payload: []byte(strconv.FormatInt(val, 10))}

	case protocol.CmdMSet:
		return &protocol.Response{Status: protocol.StatusError, Payload: []byte("not implemented")}

	case protocol.CmdMGet:
		return &protocol.Response{Status: protocol.StatusError, Payload: []byte("not implemented")}

	case protocol.CmdPing:
		resp := s.store.Ping()
		return &protocol.Response{Status: protocol.StatusValue, Payload: []byte(resp)}

	default:
		return &protocol.Response{Status: protocol.StatusError, Payload: []byte("unknown command")}
	}
}
