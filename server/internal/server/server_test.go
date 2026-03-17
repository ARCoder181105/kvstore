package server

import (
	"net"
	"testing"

	"github.com/ARCoder181105/kvstore/internal/protocol"
	"github.com/ARCoder181105/kvstore/internal/store"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	s := store.New()
	srv := New(":0", s, nil)
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

	if err := protocol.WriteCommand(conn, &protocol.Command{
		ID: protocol.CmdSet, Key: "name", Value: []byte("Alice"),
	}); err != nil {
		t.Fatal(err)
	}
	resp, err := protocol.ReadResponse(conn)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != protocol.StatusOK {
		t.Fatalf("expected StatusOK got %v", resp.Status)
	}

	if err := protocol.WriteCommand(conn, &protocol.Command{
		ID: protocol.CmdGet, Key: "name",
	}); err != nil {
		t.Fatal(err)
	}
	resp, err = protocol.ReadResponse(conn)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != protocol.StatusValue || string(resp.Payload) != "Alice" {
		t.Fatalf("expected Alice, got status=%v payload=%s", resp.Status, resp.Payload)
	}
}

func TestGetMissingKey(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Stop()

	conn, _ := net.Dial("tcp", srv.Addr())
	defer conn.Close()

	protocol.WriteCommand(conn, &protocol.Command{ID: protocol.CmdGet, Key: "missing"})
	resp, _ := protocol.ReadResponse(conn)
	if resp.Status != protocol.StatusNull {
		t.Fatalf("expected StatusNull got %v", resp.Status)
	}
}

// DEL now returns StatusInt with count, not StatusOK
func TestDelReturnsCount(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Stop()

	conn, _ := net.Dial("tcp", srv.Addr())
	defer conn.Close()

	// Delete non-existent key → 0
	protocol.WriteCommand(conn, &protocol.Command{ID: protocol.CmdDel, Key: "ghost"})
	resp, _ := protocol.ReadResponse(conn)
	if resp.Status != protocol.StatusInt || string(resp.Payload) != "0" {
		t.Fatalf("expected StatusInt/0, got status=%v payload=%s", resp.Status, resp.Payload)
	}

	// Set then delete → 1
	protocol.WriteCommand(conn, &protocol.Command{ID: protocol.CmdSet, Key: "k", Value: []byte("v")})
	protocol.ReadResponse(conn)
	protocol.WriteCommand(conn, &protocol.Command{ID: protocol.CmdDel, Key: "k"})
	resp, _ = protocol.ReadResponse(conn)
	if resp.Status != protocol.StatusInt || string(resp.Payload) != "1" {
		t.Fatalf("expected StatusInt/1, got status=%v payload=%s", resp.Status, resp.Payload)
	}
}

func TestSetDelGet(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Stop()

	conn, _ := net.Dial("tcp", srv.Addr())
	defer conn.Close()

	protocol.WriteCommand(conn, &protocol.Command{ID: protocol.CmdSet, Key: "name", Value: []byte("Alice")})
	protocol.ReadResponse(conn)
	protocol.WriteCommand(conn, &protocol.Command{ID: protocol.CmdDel, Key: "name"})
	protocol.ReadResponse(conn)
	protocol.WriteCommand(conn, &protocol.Command{ID: protocol.CmdGet, Key: "name"})
	resp, _ := protocol.ReadResponse(conn)
	if resp.Status != protocol.StatusNull {
		t.Fatalf("expected StatusNull after delete")
	}
}

func TestPing(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Stop()

	conn, _ := net.Dial("tcp", srv.Addr())
	defer conn.Close()

	protocol.WriteCommand(conn, &protocol.Command{ID: protocol.CmdPing})
	resp, _ := protocol.ReadResponse(conn)
	if string(resp.Payload) != "PONG" {
		t.Fatalf("expected PONG got %s", resp.Payload)
	}
}

func TestIncrNewKey(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Stop()

	conn, _ := net.Dial("tcp", srv.Addr())
	defer conn.Close()

	protocol.WriteCommand(conn, &protocol.Command{ID: protocol.CmdIncr, Key: "counter"})
	resp, _ := protocol.ReadResponse(conn)
	if resp.Status != protocol.StatusInt || string(resp.Payload) != "1" {
		t.Fatalf("expected StatusInt/1, got %v/%s", resp.Status, resp.Payload)
	}
}

// TTL returns seconds, not nanoseconds
func TestTTLReturnSeconds(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Stop()

	conn, _ := net.Dial("tcp", srv.Addr())
	defer conn.Close()

	// Set with 10-second TTL
	protocol.WriteCommand(conn, &protocol.Command{
		ID: protocol.CmdSet, Key: "temp", Value: []byte("v"), TTL: 10_000_000_000,
	})
	protocol.ReadResponse(conn)

	protocol.WriteCommand(conn, &protocol.Command{ID: protocol.CmdTTL, Key: "temp"})
	resp, _ := protocol.ReadResponse(conn)
	if resp.Status != protocol.StatusInt {
		t.Fatalf("expected StatusInt")
	}
	ttl := string(resp.Payload)
	// Should be "10" or "9" (never a nanosecond value like "9999...")
	if len(ttl) > 3 {
		t.Fatalf("TTL looks like nanoseconds, not seconds: %s", ttl)
	}
}
