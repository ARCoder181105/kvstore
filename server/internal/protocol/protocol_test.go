package protocol

import (
	"bytes"
	"testing"
)

func TestRoundTripSetCommand(t *testing.T) {
	buf := &bytes.Buffer{}
	cmd := &Command{ID: CmdSet, Key: "name", Value: []byte("Alice"), TTL: 0}

	if err := WriteCommand(buf, cmd); err != nil {
		t.Fatalf("WriteCommand failed: %v", err)
	}
	decoded, err := ReadCommand(buf)
	if err != nil {
		t.Fatalf("ReadCommand failed: %v", err)
	}
	if decoded.ID != cmd.ID || decoded.Key != cmd.Key || string(decoded.Value) != string(cmd.Value) || decoded.TTL != cmd.TTL {
		t.Errorf("round-trip mismatch: got %+v want %+v", decoded, cmd)
	}
}

func TestRoundTripGetCommand(t *testing.T) {
	buf := &bytes.Buffer{}
	cmd := &Command{ID: CmdGet, Key: "name"}

	if err := WriteCommand(buf, cmd); err != nil {
		t.Fatalf("WriteCommand failed: %v", err)
	}
	decoded, err := ReadCommand(buf)
	if err != nil {
		t.Fatalf("ReadCommand failed: %v", err)
	}
	if decoded.ID != cmd.ID || decoded.Key != cmd.Key {
		t.Errorf("round-trip mismatch")
	}
}

func TestRoundTripResponse(t *testing.T) {
	buf := &bytes.Buffer{}
	resp := &Response{Status: StatusValue, Payload: []byte("Alice")}

	if err := WriteResponse(buf, resp); err != nil {
		t.Fatalf("WriteResponse failed: %v", err)
	}
	decoded, err := ReadResponse(buf)
	if err != nil {
		t.Fatalf("ReadResponse failed: %v", err)
	}
	if decoded.Status != resp.Status || string(decoded.Payload) != string(resp.Payload) {
		t.Errorf("round-trip mismatch")
	}
}

func TestRoundTripOKResponse(t *testing.T) {
	buf := &bytes.Buffer{}
	resp := &Response{Status: StatusOK, Payload: nil}

	if err := WriteResponse(buf, resp); err != nil {
		t.Fatalf("WriteResponse failed: %v", err)
	}
	decoded, err := ReadResponse(buf)
	if err != nil {
		t.Fatalf("ReadResponse failed: %v", err)
	}
	if decoded.Status != resp.Status {
		t.Errorf("status mismatch")
	}
}
