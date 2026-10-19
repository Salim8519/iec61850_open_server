// Package ipc speaks the Studio native worker protocol: a 4-byte little-endian
// length followed by UTF-8 JSON, the same frames as runtime/worker_protocol.py.
package ipc

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"sync"
)

// MaxFrame matches MAX_FRAME in runtime/worker_protocol.py.
const MaxFrame = 16 * 1024 * 1024

// Request is one backend request. Params keeps the whole frame for the handler.
type Request struct {
	Version int    `json:"version"`
	ID      string `json:"id"`
	Op      string `json:"op"`
	Params  json.RawMessage
}

// ReadRequest reads one frame. io.EOF means the backend closed the pipe.
func ReadRequest(r io.Reader) (Request, error) {
	var header [4]byte
	var request Request
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return request, err
	}
	size := binary.LittleEndian.Uint32(header[:])
	if size == 0 || size > MaxFrame {
		return request, errors.New("invalid frame length")
	}
	payload := make([]byte, size)
	if _, err := io.ReadFull(r, payload); err != nil {
		return request, err
	}
	if err := json.Unmarshal(payload, &request); err != nil {
		return request, err
	}
	request.Params = payload
	return request, nil
}

// Writer serialises replies and events from many goroutines onto one pipe.
type Writer struct {
	mu sync.Mutex
	w  io.Writer
}

// NewWriter wraps the private frame output.
func NewWriter(w io.Writer) *Writer { return &Writer{w: w} }

// Write sends one JSON frame.
func (w *Writer) Write(message any) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	if len(payload) > MaxFrame {
		return errors.New("frame exceeds limit")
	}
	var header [4]byte
	binary.LittleEndian.PutUint32(header[:], uint32(len(payload)))
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, err := w.w.Write(header[:]); err != nil {
		return err
	}
	_, err = w.w.Write(payload)
	return err
}
