package protocol

import (
	"encoding/binary"
	"io"
)

func WriteResponse(w io.Writer, resp *Response) error {
	if _, err := w.Write([]byte{resp.Status}); err != nil {
		return err
	}

	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(resp.Payload)))
	if _, err := w.Write(lenBuf); err != nil {
		return err
	}

	if len(resp.Payload) > 0 {
		if _, err := w.Write(resp.Payload); err != nil {
			return err
		}
	}
	return nil
}

func WriteCommand(w io.Writer, cmd *Command) error {
	if _, err := w.Write([]byte{cmd.ID}); err != nil {
		return err
	}

	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(cmd.Key)))
	if _, err := w.Write(lenBuf); err != nil {
		return err
	}
	if _, err := w.Write([]byte(cmd.Key)); err != nil {
		return err
	}

	binary.BigEndian.PutUint32(lenBuf, uint32(len(cmd.Value)))
	if _, err := w.Write(lenBuf); err != nil {
		return err
	}
	if len(cmd.Value) > 0 {
		if _, err := w.Write(cmd.Value); err != nil {
			return err
		}
	}

	ttlBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(ttlBuf, uint64(cmd.TTL))
	if _, err := w.Write(ttlBuf); err != nil {
		return err
	}

	return nil
}
