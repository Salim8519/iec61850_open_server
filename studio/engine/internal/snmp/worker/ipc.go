// Package worker is the body of studio_snmp.exe (ADR-0092): it answers the server's framed
// requests on stdin/stdout (the native worker framing: 4-byte little-endian length, then
// JSON) and owns every SNMP socket. Requests run concurrently; replies and events are
// written one frame at a time.
package worker

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
)

// maxFrame matches the server's limit.
const maxFrame = 16 * 1024 * 1024

type request struct {
	Version int             `json:"version"`
	ID      string          `json:"id"`
	Op      string          `json:"op"`
	Raw     json.RawMessage `json:"-"`
}

// writer serialises frames onto stdout.
type writer struct {
	mu  sync.Mutex
	out io.Writer
}

func (w *writer) write(message any) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}
	if len(payload) > maxFrame {
		return fmt.Errorf("frame of %d bytes is too large", len(payload))
	}
	frame := binary.LittleEndian.AppendUint32(make([]byte, 0, 4+len(payload)), uint32(len(payload)))
	frame = append(frame, payload...)
	w.mu.Lock()
	defer w.mu.Unlock()
	_, err = w.out.Write(frame)
	return err
}

// Emit writes one event frame (it must hold an "event" key).
func (w *writer) Emit(event map[string]any) {
	_ = w.write(event)
}

func readRequest(in io.Reader) (request, error) {
	var header [4]byte
	if _, err := io.ReadFull(in, header[:]); err != nil {
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return request{}, io.EOF
		}
		return request{}, err
	}
	length := binary.LittleEndian.Uint32(header[:])
	if length == 0 || length > maxFrame {
		return request{}, fmt.Errorf("invalid frame length %d", length)
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(in, payload); err != nil {
		return request{}, io.EOF
	}
	var r request
	if err := json.Unmarshal(payload, &r); err != nil {
		return request{}, fmt.Errorf("invalid request: %v", err)
	}
	r.Raw = payload
	return r, nil
}

// Serve answers requests until "stop" or the end of stdin, then closes everything.
func Serve(in io.Reader, out io.Writer) error {
	w := &writer{out: out}
	h := newHandlers(w.Emit)
	defer h.close()
	var running sync.WaitGroup
	defer running.Wait()
	for {
		r, err := readRequest(in)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if r.Op == "stop" {
			h.close()
			return w.write(map[string]any{"id": r.ID, "ok": true, "result": map[string]any{}})
		}
		running.Add(1)
		go func() {
			defer running.Done()
			result, err := h.dispatch(r)
			if err != nil {
				_ = w.write(map[string]any{"id": r.ID, "ok": false, "error": err.Error()})
				return
			}
			_ = w.write(map[string]any{"id": r.ID, "ok": true, "result": result})
		}()
	}
}
