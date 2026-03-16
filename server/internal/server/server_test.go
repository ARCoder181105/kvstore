package server

import (
	"net"
	"testing"

	"github.com/ARCoder181105/kvstore/internal/protocol"
	"github.com/ARCoder181105/kvstore/internal/store"
)

// helper to create a test server with no AOF writer
func newTestServer(t *testing.T) *Server {
	t.Helper()
	s := store.New()
	srv := New(":0", s, nil) // nil = no AOF in tests
	if err := srv.Start(); err != nil {
		t.Fatalf("server failed to start: %v", err)
	}
	return srv
}

func TestSetAndGet(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Stop()

	conn, err := net.Dial("tcp", srv.Addr())
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	err = protocol.WriteCommand(conn, &protocol.Command{
		ID:    protocol.CmdSet,
		Key:   "name",
		Value: []byte("Alice"),
	})
	if err != nil {
		t.Fatalf("WriteCommand failed: %v", err)
	}

	resp, err := protocol.ReadResponse(conn)
	if err != nil {
		t.Fatalf("ReadResponse failed: %v", err)
	}
	if resp.Status != protocol.StatusOK {
		t.Fatalf("expected StatusOK got %v", resp.Status)
	}

	err = protocol.WriteCommand(conn, &protocol.Command{
		ID:  protocol.CmdGet,
		Key: "name",
	})
	if err != nil {
		t.Fatalf("WriteCommand failed: %v", err)
	}

	resp, err = protocol.ReadResponse(conn)
	if err != nil {
		t.Fatalf("ReadResponse failed: %v", err)
	}
	if resp.Status != protocol.StatusValue {
		t.Fatalf("expected StatusValue got %v", resp.Status)
	}
	if string(resp.Payload) != "Alice" {
		t.Fatalf("expected Alice got %v", string(resp.Payload))
	}
}

func TestGetMissingKey(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Stop()

	conn, err := net.Dial("tcp", srv.Addr())
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	err = protocol.WriteCommand(conn, &protocol.Command{
		ID:  protocol.CmdGet,
		Key: "name",
	})
	if err != nil {
		t.Fatalf("WriteCommand failed: %v", err)
	}

	resp, err := protocol.ReadResponse(conn)
	if err != nil {
		t.Fatalf("ReadResponse failed: %v", err)
	}
	if resp.Status != protocol.StatusNull {
		t.Fatalf("expected StatusNull got %v", resp.Status)
	}
}

func TestSetDelGet(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Stop()

	conn, err := net.Dial("tcp", srv.Addr())
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	err = protocol.WriteCommand(conn, &protocol.Command{
		ID:    protocol.CmdSet,
		Key:   "name",
		Value: []byte("Alice"),
	})
	if err != nil {
		t.Fatalf("WriteCommand failed: %v", err)
	}
	resp, err := protocol.ReadResponse(conn)
	if err != nil {
		t.Fatalf("ReadResponse failed: %v", err)
	}
	if resp.Status != protocol.StatusOK {
		t.Fatalf("expected StatusOK got %v", resp.Status)
	}

	err = protocol.WriteCommand(conn, &protocol.Command{
		ID:  protocol.CmdDel,
		Key: "name",
	})
	if err != nil {
		t.Fatalf("WriteCommand failed: %v", err)
	}
	resp, err = protocol.ReadResponse(conn)
	if err != nil {
		t.Fatalf("ReadResponse failed: %v", err)
	}
	if resp.Status != protocol.StatusOK {
		t.Fatalf("expected StatusOK got %v", resp.Status)
	}

	err = protocol.WriteCommand(conn, &protocol.Command{
		ID:  protocol.CmdGet,
		Key: "name",
	})
	if err != nil {
		t.Fatalf("WriteCommand failed: %v", err)
	}
	resp, err = protocol.ReadResponse(conn)
	if err != nil {
		t.Fatalf("ReadResponse failed: %v", err)
	}
	if resp.Status != protocol.StatusNull {
		t.Fatalf("expected StatusNull got %v", resp.Status)
	}
}

func TestPing(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Stop()

	conn, err := net.Dial("tcp", srv.Addr())
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	err = protocol.WriteCommand(conn, &protocol.Command{
		ID: protocol.CmdPing,
	})
	if err != nil {
		t.Fatalf("WriteCommand failed: %v", err)
	}

	resp, err := protocol.ReadResponse(conn)
	if err != nil {
		t.Fatalf("ReadResponse failed: %v", err)
	}
	if string(resp.Payload) != "PONG" {
		t.Fatalf("expected PONG got %v", string(resp.Payload))
	}
}

func TestIncrNewKey(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Stop()

	conn, err := net.Dial("tcp", srv.Addr())
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	err = protocol.WriteCommand(conn, &protocol.Command{
		ID:  protocol.CmdIncr,
		Key: "counter",
	})
	if err != nil {
		t.Fatalf("WriteCommand failed: %v", err)
	}

	resp, err := protocol.ReadResponse(conn)
	if err != nil {
		t.Fatalf("ReadResponse failed: %v", err)
	}
	if resp.Status != protocol.StatusInt {
		t.Fatalf("expected StatusInt got %v", resp.Status)
	}
	if string(resp.Payload) != "1" {
		t.Fatalf("expected 1 got %v", string(resp.Payload))
	}
}
