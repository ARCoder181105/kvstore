package protocol

import (
	"encoding/binary"
	"io"
)

func WriteResponse(w io.Writer, resp *Response) error {

	statusBuf := make([]byte, 1)
	statusBuf[0] = resp.Status
	_, err := w.Write(statusBuf)
	if err != nil {
		return err
	}

	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(resp.Payload)))
	_, err = w.Write(lenBuf)
	if err != nil {
		return err
	}

	if len(resp.Payload) > 0 {
		_, err = w.Write(resp.Payload)
		if err != nil {
			return err
		}
	}

	return nil
}

func WriteCommand(w io.Writer, cmd *Command) error {

	// write command ID
	idBuf := make([]byte, 1)
	idBuf[0] = cmd.ID
	_, err := w.Write(idBuf)
	if err != nil {
		return err
	}

	// write key length
	keyLenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(keyLenBuf, uint32(len(cmd.Key)))
	_, err = w.Write(keyLenBuf)
	if err != nil {
		return err
	}

	// write key
	_, err = w.Write([]byte(cmd.Key))
	if err != nil {
		return err
	}

	// write value length
	valLenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(valLenBuf, uint32(len(cmd.Value)))
	_, err = w.Write(valLenBuf)
	if err != nil {
		return err
	}

	// write value
	_, err = w.Write(cmd.Value)
	if err != nil {
		return err
	}

	// write TTL
	ttlBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(ttlBuf, uint64(cmd.TTL))
	_, err = w.Write(ttlBuf)
	if err != nil {
		return err
	}

	return nil
}
