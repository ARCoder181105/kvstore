package protocol

import (
	"encoding/binary"
	"io"
)

// SET name Alice
// 01                      ← command ID (1 byte)    — 0x01 means SET
// 00 00 00 04             ← key length (4 bytes)   — number 4
// 6E 61 6D 65             ← key bytes              — "name" in ASCII
// 00 00 00 05             ← value length (4 bytes) — number 5
// 41 6C 69 63 65          ← value bytes            — "Alice" in ASCII
// 00 00 00 00 00 00 00 00 ← TTL (8 bytes)          — 0 means no expiry

func ReadCommand(r io.Reader) (*Command, error) {

	cmd := &Command{}

	// read Command ID
	idBuf := make([]byte, 1)
	_, err := io.ReadFull(r, idBuf)
	if err != nil {
		return nil, err
	}
	cmd.ID = idBuf[0]

	// read Key length
	lenBuf := make([]byte, 4)
	_, err = io.ReadFull(r, lenBuf)
	if err != nil {
		return nil, err
	}
	keyLen := binary.BigEndian.Uint32(lenBuf)

	// read Key
	keyBuf := make([]byte, keyLen)
	_, err = io.ReadFull(r, keyBuf)
	if err != nil {
		return nil, err
	}
	cmd.Key = string(keyBuf)

	// read value length
	_, err = io.ReadFull(r, lenBuf)
	if err != nil {
		return nil, err
	}
	valueLen := binary.BigEndian.Uint32(lenBuf)

	// read value
	valueBuf := make([]byte, valueLen)
	_, err = io.ReadFull(r, valueBuf)
	if err != nil {
		return nil, err
	}
	cmd.Value = valueBuf

	// read TTL
	ttlBuf := make([]byte, 8)
	_, err = io.ReadFull(r, ttlBuf)
	if err != nil {
		return nil, err
	}
	cmd.TTL = int64(binary.BigEndian.Uint64(ttlBuf))

	return cmd, nil
}

func ReadResponse(r io.Reader) (*Response, error) {
	resp := &Response{}

	statusBuf := make([]byte, 1)
	_, err := io.ReadFull(r, statusBuf)
	if err != nil {
		return nil, err
	}
	resp.Status = statusBuf[0]

	lenBuf := make([]byte, 4)
	_, err = io.ReadFull(r, lenBuf)
	if err != nil {
		return nil, err
	}
	valueLen := binary.BigEndian.Uint32(lenBuf)

	payloadBuf := make([]byte, valueLen)
	_, err = io.ReadFull(r, payloadBuf)
	if err != nil {
		return nil, err
	}
	resp.Payload = payloadBuf

	return resp, nil
}
