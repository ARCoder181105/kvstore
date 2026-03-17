package protocol

import (
	"encoding/binary"
	"io"
)

func ReadCommand(r io.Reader) (*Command, error) {
	cmd := &Command{}

	idBuf := make([]byte, 1)
	if _, err := io.ReadFull(r, idBuf); err != nil {
		return nil, err
	}
	cmd.ID = idBuf[0]

	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(r, lenBuf); err != nil {
		return nil, err
	}
	keyLen := binary.BigEndian.Uint32(lenBuf)

	keyBuf := make([]byte, keyLen)
	if _, err := io.ReadFull(r, keyBuf); err != nil {
		return nil, err
	}
	cmd.Key = string(keyBuf)

	if _, err := io.ReadFull(r, lenBuf); err != nil {
		return nil, err
	}
	valueLen := binary.BigEndian.Uint32(lenBuf)

	valueBuf := make([]byte, valueLen)
	if _, err := io.ReadFull(r, valueBuf); err != nil {
		return nil, err
	}
	cmd.Value = valueBuf

	ttlBuf := make([]byte, 8)
	if _, err := io.ReadFull(r, ttlBuf); err != nil {
		return nil, err
	}
	cmd.TTL = int64(binary.BigEndian.Uint64(ttlBuf))

	return cmd, nil
}

func ReadResponse(r io.Reader) (*Response, error) {
	resp := &Response{}

	statusBuf := make([]byte, 1)
	if _, err := io.ReadFull(r, statusBuf); err != nil {
		return nil, err
	}
	resp.Status = statusBuf[0]

	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(r, lenBuf); err != nil {
		return nil, err
	}
	payloadLen := binary.BigEndian.Uint32(lenBuf)

	payloadBuf := make([]byte, payloadLen)
	if _, err := io.ReadFull(r, payloadBuf); err != nil {
		return nil, err
	}
	resp.Payload = payloadBuf

	return resp, nil
}
