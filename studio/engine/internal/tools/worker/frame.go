// Package worker talks to one native framed worker (studio_gateway, studio_master,
// studio_control, studio_live) exactly like runtime/worker_process.py: 4-byte little-endian
// length + JSON, requests by id, events delivered in order on the reader, worker_failed once.
//
// It is stream E's stand-in for stream C's internal/native framing; the tools and gateway
// code only use the Process interface, so the integrator can swap the implementation.
package worker

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// MaxFrame is the 16 MiB frame limit both sides enforce.
const MaxFrame = 16 * 1024 * 1024

// errClosed is the EOF text Python's read_exact raises ("Native worker pipe closed").
var errClosed = errors.New("Native worker pipe closed")

func readFrame(reader io.Reader) (*pyjson.Object, error) {
	var header [4]byte
	if _, err := io.ReadFull(reader, header[:]); err != nil {
		return nil, pipeError(err)
	}
	length := binary.LittleEndian.Uint32(header[:])
	if length == 0 || length > MaxFrame {
		return nil, fmt.Errorf("Invalid native frame length %d", length)
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return nil, pipeError(err)
	}
	value, err := pyjson.Decode(payload)
	if err != nil {
		return nil, err
	}
	message, ok := value.(*pyjson.Object)
	if !ok {
		// Python would fail on `"event" in message` for a list; a number is a TypeError too.
		return nil, fmt.Errorf("native frame is not an object")
	}
	return message, nil
}

func pipeError(err error) error {
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return errClosed
	}
	return err
}

// encodeFrame is json.dumps(message, allow_nan=False, separators=(",", ":")) + header.
func encodeFrame(message *pyjson.Object) ([]byte, error) {
	if !finite(message) {
		return nil, apperr.ValueError("Out of range float values are not JSON compliant")
	}
	payload, err := pyjson.Compact(message)
	if err != nil {
		return nil, err
	}
	if len(payload) > MaxFrame {
		return nil, apperr.ValueError("Native request exceeds frame limit")
	}
	frame := make([]byte, 4, 4+len(payload))
	binary.LittleEndian.PutUint32(frame, uint32(len(payload)))
	return append(frame, payload...), nil
}

func finite(value any) bool {
	switch v := value.(type) {
	case float64:
		return !math.IsNaN(v) && !math.IsInf(v, 0)
	case []any:
		for _, item := range v {
			if !finite(item) {
				return false
			}
		}
	case *pyjson.Object:
		for _, key := range v.Keys() {
			item, _ := v.Get(key)
			if !finite(item) {
				return false
			}
		}
	case nil, bool, int, int64, *big.Int, string:
	}
	return true
}
