package protocol

import (
	"bytes"
	"testing"
)

func TestRoundTripSetCommand(t *testing.T) {
	// create a bytes buffer — acts as fake TCP connection
	buf := &bytes.Buffer{}

	// write a SET command as bytes into the buffer
	cmd := &Command{
		ID:    CmdSet,
		Key:   "name",
		Value: []byte("Alice"),
		TTL:   0,
	}

	err := WriteCommand(buf, cmd)
	if err != nil {
		t.Fatalf("WriteCommand failed: %v", err)
	}

	// read it back
	decoded, err := ReadCommand(buf)
	if err != nil {
		t.Fatalf("ReadCommand failed: %v", err)
	}

	// check every field matches
	if decoded.ID != cmd.ID {
		t.Errorf("ID mismatch: got %v want %v", decoded.ID, cmd.ID)
	}
	if decoded.Key != cmd.Key {
		t.Errorf("Key mismatch: got %v want %v", decoded.Key, cmd.Key)
	}
	if string(decoded.Value) != string(cmd.Value) {
		t.Errorf("Value mismatch: got %v want %v", decoded.Value, cmd.Value)
	}
	if decoded.TTL != cmd.TTL {
		t.Errorf("TTL mismatch: got %v want %v", decoded.TTL, cmd.TTL)
	}
}

func TestRoundTripGetCommand(t *testing.T) {
	buf := &bytes.Buffer{}

	cmd := &Command{
		ID:  CmdGet,
		Key: "name",
	}

	err := WriteCommand(buf, cmd)
	if err != nil {
		t.Fatalf("WriteCommand failed: %v", err)
	}

	decoded, err := ReadCommand(buf)
	if err != nil {
		t.Fatalf("ReadCommand failed: %v", err)
	}

	if decoded.ID != cmd.ID {
		t.Errorf("ID mismatch: got %v want %v", decoded.ID, cmd.ID)
	}
	if decoded.Key != cmd.Key {
		t.Errorf("Key mismatch: got %v want %v", decoded.Key, cmd.Key)
	}
}

func TestRoundTripResponse(t *testing.T) {
	buf := &bytes.Buffer{}

	resp := &Response{
		Status:  StatusValue,
		Payload: []byte("Alice"),
	}

	err := WriteResponse(buf, resp)
	if err != nil {
		t.Fatalf("WriteResponse failed: %v", err)
	}

	decoded, err := ReadResponse(buf)
	if err != nil {
		t.Fatalf("ReadResponse failed: %v", err)
	}

	if decoded.Status != resp.Status {
		t.Errorf("Status mismatch: got %v want %v", decoded.Status, resp.Status)
	}
	if string(decoded.Payload) != string(resp.Payload) {
		t.Errorf("Payload mismatch: got %v want %v", decoded.Payload, resp.Payload)
	}
}

func TestRoundTripOKResponse(t *testing.T) {
	buf := &bytes.Buffer{}

	resp := &Response{
		Status:  StatusOK,
		Payload: nil,
	}

	err := WriteResponse(buf, resp)
	if err != nil {
		t.Fatalf("WriteResponse failed: %v", err)
	}

	decoded, err := ReadResponse(buf)
	if err != nil {
		t.Fatalf("ReadResponse failed: %v", err)
	}

	if decoded.Status != resp.Status {
		t.Errorf("Status mismatch: got %v want %v", decoded.Status, resp.Status)
	}
}
