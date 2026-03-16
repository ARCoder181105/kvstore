package server

import (
	"net"
	"testing"

	"github.com/ARCoder181105/kvstore/internal/protocol"
	"github.com/ARCoder181105/kvstore/internal/store"
)

func TestSetAndGet(t *testing.T) {
	s := store.New()
	srv := New(":0", s)

	err := srv.Start()
	if err != nil {
		t.Fatalf("server failed to start: %v", err)
	}
	defer srv.Stop()

	conn, err := net.Dial("tcp", srv.Addr())
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// send SET command
	err = protocol.WriteCommand(conn, &protocol.Command{
		ID:    protocol.CmdSet,
		Key:   "name",
		Value: []byte("Alice"),
	})
	if err != nil {
		t.Fatalf("WriteCommand failed: %v", err)
	}

	// read SET response
	resp, err := protocol.ReadResponse(conn)
	if err != nil {
		t.Fatalf("ReadResponse failed: %v", err)
	}
	if resp.Status != protocol.StatusOK {
		t.Fatalf("expected StatusOK got %v", resp.Status)
	}

	// send GET command
	err = protocol.WriteCommand(conn, &protocol.Command{
		ID:  protocol.CmdGet,
		Key: "name",
	})
	if err != nil {
		t.Fatalf("WriteCommand failed: %v", err)
	}

	// read GET response
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
	s := store.New()
	srv := New(":0", s)

	err := srv.Start()
	if err != nil {
		t.Fatalf("server failed to start: %v", err)
	}
	defer srv.Stop()

	conn, err := net.Dial("tcp", srv.Addr())
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// send GET for missing key
	err = protocol.WriteCommand(conn, &protocol.Command{
		ID:  protocol.CmdGet,
		Key: "name",
	})
	if err != nil {
		t.Fatalf("WriteCommand failed: %v", err)
	}

	// read response
	resp, err := protocol.ReadResponse(conn)
	if err != nil {
		t.Fatalf("ReadResponse failed: %v", err)
	}

	// key doesn't exist so must be StatusNull
	if resp.Status != protocol.StatusNull {
		t.Fatalf("expected StatusNull got %v", resp.Status)
	}
}
func TestSetDelGet(t *testing.T) {
	s := store.New()
	srv := New(":0", s)

	err := srv.Start()
	if err != nil {
		t.Fatalf("server failed to start: %v", err)
	}
	defer srv.Stop()

	conn, err := net.Dial("tcp", srv.Addr())
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// send SET
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

	// send DEL
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

	// send GET — should be nil now
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
	s := store.New()
	srv := New(":0", s)

	err := srv.Start()
	if err != nil {
		t.Fatalf("server failed to start: %v", err)
	}
	defer srv.Stop()

	conn, err := net.Dial("tcp", srv.Addr())
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// send PING
	err = protocol.WriteCommand(conn, &protocol.Command{
		ID: protocol.CmdPing,
	})
	if err != nil {
		t.Fatalf("WriteCommand failed: %v", err)
	}

	// read response
	resp, err := protocol.ReadResponse(conn)
	if err != nil {
		t.Fatalf("ReadResponse failed: %v", err)
	}
	if string(resp.Payload) != "PONG" {
		t.Fatalf("expected PONG got %v", string(resp.Payload))
	}
}

func TestIncrNewKey(t *testing.T) {
	s := store.New()
	srv := New(":0", s)

	err := srv.Start()
	if err != nil {
		t.Fatalf("server failed to start: %v", err)
	}
	defer srv.Stop()

	conn, err := net.Dial("tcp", srv.Addr())
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	// send INCR on a key that doesn't exist yet
	err = protocol.WriteCommand(conn, &protocol.Command{
		ID:  protocol.CmdIncr,
		Key: "counter",
	})
	if err != nil {
		t.Fatalf("WriteCommand failed: %v", err)
	}

	// read response
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
