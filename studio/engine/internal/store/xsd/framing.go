package xsd

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

// frameLimit is the native framing limit shared by every ScadaBench worker.
const frameLimit = 16 * 1024 * 1024

// writeFrame sends one message: 4-byte little-endian length, then the JSON bytes.
func writeFrame(w io.Writer, message any) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	if len(payload) == 0 || len(payload) > frameLimit {
		return fmt.Errorf("frame of %d bytes is outside the 16 MiB limit", len(payload))
	}
	frame := make([]byte, 4+len(payload))
	binary.LittleEndian.PutUint32(frame, uint32(len(payload)))
	copy(frame[4:], payload)
	_, err = w.Write(frame)
	return err
}

// readFrame reads one frame's JSON payload; io.EOF means a clean end of the stream.
func readFrame(r io.Reader) ([]byte, error) {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		if err == io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("truncated frame header")
		}
		return nil, err
	}
	size := binary.LittleEndian.Uint32(header[:])
	if size == 0 || size > frameLimit {
		return nil, fmt.Errorf("invalid frame length %d", size)
	}
	payload := make([]byte, size)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, fmt.Errorf("truncated frame: %v", err)
	}
	return payload, nil
}
